package prescription

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/policy"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("invalid prescription state")
)

type DB interface {
	Begin(context.Context) (pgx.Tx, error)
}
type Service struct {
	DB    DB
	Audit *audit.Writer
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Business denials are audited and committed. Infrastructure failures roll back
// both the business mutation and its audit, and handlers never return the payload.
func (s *Service) transact(ctx context.Context, token []byte, fn func(context.Context, pgx.Tx, auth.User) error) error {
	if len(token) != 32 {
		return ErrUnauthenticated
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	actor, err := (auth.Store{DB: tx}).UserBySession(ctx, token)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUnauthenticated
	}
	if err != nil {
		return err
	}
	decision := fn(ctx, tx, actor)
	if decision != nil && !errors.Is(decision, ErrForbidden) && !errors.Is(decision, ErrConflict) {
		return decision
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return decision
}

const prescriptionSelect = `SELECT r.id::text,r.episode_id::text,p.patient_ref,r.pharmacy_org_id::text,o.name,
    COALESCE((SELECT jsonb_agg(jsonb_build_object('medication_name',i.medication_name,'dosage',i.dosage,
        'instructions',i.instructions,'quantity',i.quantity) ORDER BY i.position)
        FROM prescription_items i WHERE i.prescription_id=r.id),'[]'::jsonb),
    r.status,r.created_at,r.updated_at,r.approved_at,r.completed_at,r.cancelled_at,r.cancellation_reason,
    p.id::text,e.doctor_id::text,p.user_id::text,e.organization_id::text,e.status
    FROM prescriptions r JOIN care_episodes e ON e.id=r.episode_id JOIN patients p ON p.id=e.patient_id
    LEFT JOIN organizations o ON o.id=r.pharmacy_org_id `

func scanPrescription(row interface{ Scan(...any) error }) (Prescription, error) {
	var p Prescription
	err := row.Scan(&p.ID, &p.EpisodeID, &p.PatientRef, &p.PharmacyOrgID, &p.PharmacyName, &p.Items,
		&p.Status, &p.CreatedAt, &p.UpdatedAt, &p.ApprovedAt, &p.CompletedAt, &p.CancelledAt, &p.CancellationReason,
		&p.patientID, &p.episode.DoctorID, &p.episode.PatientUserID, &p.episode.OrganizationID, &p.episode.Status)
	return p, err
}

// Lock order matches care's episode-end transaction: episode, prescription, audit.
func loadEpisode(ctx context.Context, tx pgx.Tx, id string, mutate bool) (policy.Episode, string, error) {
	var e policy.Episode
	var patient string
	lock := "FOR SHARE OF e"
	if mutate {
		lock = "FOR UPDATE OF e"
	}
	err := tx.QueryRow(ctx, `SELECT e.doctor_id::text,p.user_id::text,e.organization_id::text,e.status,p.id::text
        FROM care_episodes e JOIN patients p ON p.id=e.patient_id WHERE e.id=$1 `+lock, id).
		Scan(&e.DoctorID, &e.PatientUserID, &e.OrganizationID, &e.Status, &patient)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, "", ErrForbidden
	}
	return e, patient, err
}

func loadPrescription(ctx context.Context, tx pgx.Tx, id string, mutate bool) (Prescription, error) {
	var episode string
	err := tx.QueryRow(ctx, `SELECT episode_id::text FROM prescriptions WHERE id=$1`, id).Scan(&episode)
	if errors.Is(err, pgx.ErrNoRows) {
		return Prescription{}, ErrForbidden
	}
	if err != nil {
		return Prescription{}, err
	}
	if _, _, err := loadEpisode(ctx, tx, episode, mutate); err != nil {
		return Prescription{}, err
	}
	lock := "FOR SHARE OF r"
	if mutate {
		lock = "FOR UPDATE OF r"
	}
	return scanPrescription(tx.QueryRow(ctx, prescriptionSelect+`WHERE r.id=$1 `+lock, id))
}

func accessBasis(actor auth.User) string {
	switch actor.Role {
	case "patient":
		return "patient_self"
	case "pharmacy":
		return "approved_pharmacy_assignment"
	default:
		return "episode_doctor"
	}
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, actor auth.User, p Prescription, action, outcome, basis string) error {
	return s.Audit.Append(ctx, tx, actor, audit.Event{ChainID: p.patientID, Action: action, ResourceType: "prescription", ResourceID: p.ID, Outcome: outcome, Basis: basis})
}

func (s *Service) reject(ctx context.Context, tx pgx.Tx, actor auth.User, p Prescription, action string, reason error) error {
	basis := "no_access"
	if errors.Is(reason, ErrConflict) {
		basis = "invalid_state"
	}
	if err := s.record(ctx, tx, actor, p, action, "deny", basis); err != nil {
		return err
	}
	return reason
}

func (s *Service) Pharmacies(ctx context.Context, token []byte, limit, offset int) ([]Pharmacy, error) {
	result := make([]Pharmacy, 0)
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if actor.Role != "patient" {
			return ErrForbidden
		}
		rows, err := tx.Query(ctx, `SELECT id::text,name FROM organizations WHERE type='pharmacy' ORDER BY name,id LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var p Pharmacy
			if err := rows.Scan(&p.ID, &p.Name); err != nil {
				return err
			}
			result = append(result, p)
		}
		return rows.Err()
	})
	return result, err
}

func (s *Service) Create(ctx context.Context, token []byte, input CreateInput) (Prescription, error) {
	var result Prescription
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		e, patient, err := loadEpisode(ctx, tx, input.EpisodeID, true)
		if err != nil {
			return err
		}
		if !policy.HasActiveEpisode(actor, e) {
			if err := s.Audit.Append(ctx, tx, actor, audit.Event{ChainID: patient, Action: "prescription.create", ResourceType: "care_episode", ResourceID: input.EpisodeID, Outcome: "deny", Basis: "no_access"}); err != nil {
				return err
			}
			return ErrForbidden
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO prescriptions(episode_id) VALUES($1) RETURNING id::text`, input.EpisodeID).Scan(&id); err != nil {
			return err
		}
		for position, item := range input.Items {
			if _, err := tx.Exec(ctx, `INSERT INTO prescription_items(prescription_id,position,medication_name,dosage,instructions,quantity)
                VALUES($1,$2,$3,$4,$5,$6)`, id, position+1, item.MedicationName, item.Dosage, item.Instructions, item.Quantity); err != nil {
				return err
			}
		}
		result, err = scanPrescription(tx.QueryRow(ctx, prescriptionSelect+`WHERE r.id=$1`, id))
		if err != nil {
			return err
		}
		return s.record(ctx, tx, actor, result, "prescription.create", "allow", accessBasis(actor))
	})
	return result, err
}

func (s *Service) Get(ctx context.Context, token []byte, id string) (any, error) {
	var result any
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		p, err := loadPrescription(ctx, tx, id, false)
		if err != nil {
			return err
		}
		event := audit.Event{ChainID: p.patientID, Action: "prescription.view", ResourceType: "prescription", ResourceID: p.ID, Outcome: "allow", Basis: accessBasis(actor)}
		allowed := policy.CanViewPrescription(actor, p.access())
		if !allowed {
			grant, err := policy.FindCarePassGrant(ctx, tx, actor, p.episode.PatientUserID, "prescription", p.ID, s.now)
			if err != nil {
				return err
			}
			if grant.ID != "" {
				allowed = true
				event.Basis = "carepass"
				event.CarePassID = grant.ID
				event.CarePassExpiresAt = &grant.ExpiresAt
			}
		}
		if !allowed && policy.IsActivePrescription(p.Status) {
			grant, err := policy.FindEmergencyGrant(ctx, tx, actor, p.episode.PatientUserID, s.now)
			if err != nil {
				return err
			}
			if grant.ID != "" {
				allowed = true
				event.Basis = "break_glass"
				event.BreakGlassID = grant.ID
				event.BreakGlassExpiresAt = &grant.ExpiresAt
			}
		}
		if !allowed {
			return s.reject(ctx, tx, actor, p, "prescription.view", ErrForbidden)
		}
		if err := s.Audit.Append(ctx, tx, actor, event); err != nil {
			return err
		}
		result = p.view(actor)
		return nil
	})
	return result, err
}

func (s *Service) List(ctx context.Context, token []byte, status string, limit, offset int) ([]any, error) {
	result := make([]any, 0)
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		condition := ""
		args := []any{actor.ID, status, limit, offset}
		switch actor.Role {
		case "patient":
			condition = `p.user_id=$1`
		case "doctor":
			if actor.Organization == nil || actor.Organization.Type != "clinic" {
				return ErrForbidden
			}
			condition = `e.doctor_id=$1 AND e.organization_id=$5 AND e.status='active'`
			args = append(args, actor.Organization.ID)
		case "pharmacy":
			if actor.Organization == nil || actor.Organization.Type != "pharmacy" {
				return ErrForbidden
			}
			condition = `r.pharmacy_org_id=$1 AND r.status IN ('sent','accepted','preparing','ready')`
			args[0] = actor.Organization.ID
		default:
			return ErrForbidden
		}
		// Select IDs first, then lock episodes in a stable order and re-read. This
		// prevents an episode-end/queue-completion race leaking stale clinical data.
		rows, err := tx.Query(ctx, `SELECT r.id::text,r.episode_id::text FROM prescriptions r
            JOIN care_episodes e ON e.id=r.episode_id JOIN patients p ON p.id=e.patient_id
            WHERE `+condition+` AND ($2='' OR r.status=$2) ORDER BY r.created_at DESC,r.id DESC LIMIT $3 OFFSET $4`, args...)
		if err != nil {
			return err
		}
		ids := make([]string, 0)
		episodes := map[string]bool{}
		for rows.Next() {
			var id, episode string
			if err := rows.Scan(&id, &episode); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
			episodes[episode] = true
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		ordered := make([]string, 0, len(episodes))
		for id := range episodes {
			ordered = append(ordered, id)
		}
		sort.Strings(ordered)
		for _, id := range ordered {
			if _, _, err := loadEpisode(ctx, tx, id, false); err != nil {
				return err
			}
		}
		chains := map[string]Prescription{}
		for _, id := range ids {
			p, err := loadPrescription(ctx, tx, id, false)
			if err != nil {
				return err
			}
			if !policy.CanViewPrescription(actor, p.access()) || (status != "" && p.Status != status) {
				continue
			}
			result = append(result, p.view(actor))
			chains[p.patientID] = p
		}
		ordered = ordered[:0]
		for id := range chains {
			ordered = append(ordered, id)
		}
		sort.Strings(ordered)
		for _, id := range ordered {
			if err := s.record(ctx, tx, actor, chains[id], "prescription.list", "allow", accessBasis(actor)); err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

func (s *Service) Approve(ctx context.Context, token []byte, id string, input ApprovalInput) (Prescription, error) {
	var result Prescription
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		p, err := loadPrescription(ctx, tx, id, true)
		if err != nil {
			return err
		}
		action := "prescription." + input.Decision
		if !policy.CanDecidePrescription(actor, p.access()) {
			return s.reject(ctx, tx, actor, p, action, ErrForbidden)
		}
		if p.Status != "pending_patient_approval" || p.episode.Status != "active" {
			return s.reject(ctx, tx, actor, p, action, ErrConflict)
		}
		switch input.Decision {
		case "approve":
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1 AND type='pharmacy')`, input.PharmacyOrgID).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return s.reject(ctx, tx, actor, p, action, ErrForbidden)
			}
			_, err = tx.Exec(ctx, `UPDATE prescriptions SET status='sent',pharmacy_org_id=$2,approved_at=clock_timestamp(),updated_at=clock_timestamp() WHERE id=$1`, id, input.PharmacyOrgID)
		case "reject":
			_, err = tx.Exec(ctx, `UPDATE prescriptions SET status='cancelled',cancelled_at=clock_timestamp(),cancellation_reason='patient_rejected',updated_at=clock_timestamp() WHERE id=$1`, id)
		default:
			return s.reject(ctx, tx, actor, p, "prescription.approval", ErrConflict)
		}
		if err != nil {
			return err
		}
		result, err = scanPrescription(tx.QueryRow(ctx, prescriptionSelect+`WHERE r.id=$1`, id))
		if err != nil {
			return err
		}
		return s.record(ctx, tx, actor, result, action, "allow", accessBasis(actor))
	})
	return result, err
}

func (s *Service) UpdateStatus(ctx context.Context, token []byte, id, status string) (QueuePrescription, error) {
	var result QueuePrescription
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		p, err := loadPrescription(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if !policy.IsAssignedPharmacy(actor, p.access()) {
			return s.reject(ctx, tx, actor, p, "prescription.status", ErrForbidden)
		}
		if !policy.CanTransitionPrescription(p.Status, status) {
			return s.reject(ctx, tx, actor, p, "prescription.status", ErrConflict)
		}
		if _, err := tx.Exec(ctx, `UPDATE prescriptions SET status=$2,updated_at=clock_timestamp(),
            completed_at=CASE WHEN $2='completed' THEN clock_timestamp() ELSE NULL END WHERE id=$1`, id, status); err != nil {
			return err
		}
		p.Status = status
		if err := s.record(ctx, tx, actor, p, "prescription."+status, "allow", accessBasis(actor)); err != nil {
			return err
		}
		result = p.view(actor).(QueuePrescription)
		return nil
	})
	return result, err
}
