package laboratory

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
	ErrConflict        = errors.New("invalid lab request state")
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
		return s.Now().UTC()
	}
	return time.Now().UTC()
}

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

const requestSelect = `SELECT r.id::text,r.episode_id::text,p.patient_ref,r.lab_org_id::text,l.name,
    r.test_code,r.test_name,r.status,r.created_at,r.updated_at,r.approved_at,r.completed_at,r.cancelled_at,r.cancellation_reason,x.id::text,
    p.id::text,e.doctor_id::text,p.user_id::text,e.organization_id::text,e.status
    FROM lab_requests r JOIN care_episodes e ON e.id=r.episode_id JOIN patients p ON p.id=e.patient_id
    JOIN organizations l ON l.id=r.lab_org_id LEFT JOIN lab_results x ON x.lab_request_id=r.id `

func scanRequest(row interface{ Scan(...any) error }) (Request, error) {
	var r Request
	err := row.Scan(&r.ID, &r.EpisodeID, &r.PatientRef, &r.LabOrgID, &r.LabName, &r.TestCode, &r.TestName, &r.Status,
		&r.CreatedAt, &r.UpdatedAt, &r.ApprovedAt, &r.CompletedAt, &r.CancelledAt, &r.CancellationReason, &r.ResultID,
		&r.patientID, &r.episode.DoctorID, &r.episode.PatientUserID, &r.episode.OrganizationID, &r.episode.Status)
	return r, err
}

// All resource operations acquire the episode first. End/approval/process races
// therefore have a single order, shared with care's episode-end transaction.
func loadEpisode(ctx context.Context, tx pgx.Tx, id string, mutate bool) (policy.Episode, string, error) {
	var e policy.Episode
	var patient string
	lock := "FOR SHARE OF e"
	if mutate {
		lock = "FOR UPDATE OF e"
	}
	err := tx.QueryRow(ctx, `SELECT e.doctor_id::text,p.user_id::text,e.organization_id::text,e.status,p.id::text
        FROM care_episodes e JOIN patients p ON p.id=e.patient_id WHERE e.id=$1 `+lock, id).Scan(&e.DoctorID, &e.PatientUserID, &e.OrganizationID, &e.Status, &patient)
	if errors.Is(err, pgx.ErrNoRows) {
		return e, "", ErrForbidden
	}
	return e, patient, err
}

func loadRequest(ctx context.Context, tx pgx.Tx, id string, mutate bool) (Request, error) {
	var episodeID string
	err := tx.QueryRow(ctx, `SELECT episode_id::text FROM lab_requests WHERE id=$1`, id).Scan(&episodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Request{}, ErrForbidden
	}
	if err != nil {
		return Request{}, err
	}
	if _, _, err := loadEpisode(ctx, tx, episodeID, mutate); err != nil {
		return Request{}, err
	}
	lock := "FOR SHARE OF r"
	if mutate {
		lock = "FOR UPDATE OF r"
	}
	return scanRequest(tx.QueryRow(ctx, requestSelect+`WHERE r.id=$1 `+lock, id))
}

func accessBasis(actor auth.User) string {
	switch actor.Role {
	case "patient":
		return "patient_self"
	case "lab":
		return "approved_lab_assignment"
	default:
		return "episode_doctor"
	}
}

func (s *Service) record(ctx context.Context, tx pgx.Tx, actor auth.User, r Request, action, outcome, basis string) error {
	return s.Audit.Append(ctx, tx, actor, audit.Event{ChainID: r.patientID, Action: action, ResourceType: "lab_request", ResourceID: r.ID, Outcome: outcome, Basis: basis})
}

func (s *Service) reject(ctx context.Context, tx pgx.Tx, actor auth.User, r Request, action string, reason error) error {
	basis := "no_access"
	if errors.Is(reason, ErrConflict) {
		basis = "invalid_state"
	}
	if err := s.record(ctx, tx, actor, r, action, "deny", basis); err != nil {
		return err
	}
	return reason
}

func (s *Service) Laboratories(ctx context.Context, token []byte, limit, offset int) ([]Laboratory, error) {
	result := make([]Laboratory, 0)
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if actor.Role != "doctor" {
			return ErrForbidden
		}
		rows, err := tx.Query(ctx, `SELECT id::text,name FROM organizations WHERE type='laboratory' ORDER BY name,id LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var lab Laboratory
			if err := rows.Scan(&lab.ID, &lab.Name); err != nil {
				return err
			}
			result = append(result, lab)
		}
		return rows.Err()
	})
	return result, err
}

func (s *Service) Create(ctx context.Context, token []byte, input CreateInput) (Request, error) {
	var result Request
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		e, patient, err := loadEpisode(ctx, tx, input.EpisodeID, true)
		if err != nil {
			return err
		}
		if !policy.HasActiveEpisode(actor, e) {
			if err := s.Audit.Append(ctx, tx, actor, audit.Event{ChainID: patient, Action: "lab_request.create", ResourceType: "care_episode", ResourceID: input.EpisodeID, Outcome: "deny", Basis: "no_access"}); err != nil {
				return err
			}
			return ErrForbidden
		}
		var exists bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1 AND type='laboratory')`, input.LabOrgID).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrForbidden
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO lab_requests(episode_id,lab_org_id,test_code,test_name) VALUES($1,$2,$3,$4) RETURNING id::text`, input.EpisodeID, input.LabOrgID, input.TestCode, input.TestName).Scan(&id); err != nil {
			return err
		}
		result, err = scanRequest(tx.QueryRow(ctx, requestSelect+`WHERE r.id=$1`, id))
		if err != nil {
			return err
		}
		return s.record(ctx, tx, actor, result, "lab_request.create", "allow", accessBasis(actor))
	})
	return result, err
}

func (s *Service) Get(ctx context.Context, token []byte, id string) (any, error) {
	var result any
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		r, err := loadRequest(ctx, tx, id, false)
		if err != nil {
			return err
		}
		if !policy.CanViewLabRequest(actor, r.access()) {
			return s.reject(ctx, tx, actor, r, "lab_request.view", ErrForbidden)
		}
		if err := s.record(ctx, tx, actor, r, "lab_request.view", "allow", accessBasis(actor)); err != nil {
			return err
		}
		result = r.view(actor)
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
			if actor.Organization == nil {
				return ErrForbidden
			}
			condition = `e.doctor_id=$1 AND e.organization_id=$5 AND e.status='active'`
			args = append(args, actor.Organization.ID)
		case "lab":
			if actor.Organization == nil {
				return ErrForbidden
			}
			condition = `r.lab_org_id=$1 AND r.status IN ('sent','accepted','in_progress')`
			args[0] = actor.Organization.ID
		default:
			return ErrForbidden
		}
		rows, err := tx.Query(ctx, requestSelect+`WHERE `+condition+` AND ($2='' OR r.status=$2) ORDER BY r.created_at DESC,r.id DESC LIMIT $3 OFFSET $4`, args...)
		if err != nil {
			return err
		}
		requests := make([]Request, 0)
		for rows.Next() {
			r, err := scanRequest(rows)
			if err != nil {
				rows.Close()
				return err
			}
			requests = append(requests, r)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		chains := map[string]Request{}
		for _, r := range requests {
			if !policy.CanViewLabRequest(actor, r.access()) {
				return ErrForbidden
			}
			result = append(result, r.view(actor))
			chains[r.patientID] = r
		}
		ids := make([]string, 0, len(chains))
		for id := range chains {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if err := s.record(ctx, tx, actor, chains[id], "lab_request.list", "allow", accessBasis(actor)); err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

func (s *Service) Approve(ctx context.Context, token []byte, id, decision string) (Request, error) {
	var result Request
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		r, err := loadRequest(ctx, tx, id, true)
		if err != nil {
			return err
		}
		action := "lab_request." + decision
		if !policy.CanDecideLabRequest(actor, r.access()) {
			return s.reject(ctx, tx, actor, r, action, ErrForbidden)
		}
		if r.Status != "pending_patient_approval" || r.episode.Status != "active" {
			return s.reject(ctx, tx, actor, r, action, ErrConflict)
		}
		switch decision {
		case "approve":
			_, err = tx.Exec(ctx, `UPDATE lab_requests SET status='sent',approved_at=clock_timestamp(),updated_at=clock_timestamp() WHERE id=$1`, id)
		case "reject":
			_, err = tx.Exec(ctx, `UPDATE lab_requests SET status='cancelled',cancelled_at=clock_timestamp(),cancellation_reason='patient_rejected',updated_at=clock_timestamp() WHERE id=$1`, id)
		default:
			return ErrConflict
		}
		if err != nil {
			return err
		}
		result, err = scanRequest(tx.QueryRow(ctx, requestSelect+`WHERE r.id=$1`, id))
		if err != nil {
			return err
		}
		return s.record(ctx, tx, actor, result, action, "allow", accessBasis(actor))
	})
	return result, err
}

func (s *Service) UpdateStatus(ctx context.Context, token []byte, id, status string) (QueueRequest, error) {
	var result QueueRequest
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		r, err := loadRequest(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if !policy.CanProcessLabRequest(actor, r.access()) {
			return s.reject(ctx, tx, actor, r, "lab_request.status", ErrForbidden)
		}
		if !((r.Status == "sent" && status == "accepted") || (r.Status == "accepted" && status == "in_progress")) {
			return s.reject(ctx, tx, actor, r, "lab_request.status", ErrConflict)
		}
		if _, err := tx.Exec(ctx, `UPDATE lab_requests SET status=$2,updated_at=clock_timestamp() WHERE id=$1`, id, status); err != nil {
			return err
		}
		r.Status = status
		if err := s.record(ctx, tx, actor, r, "lab_request."+status, "allow", accessBasis(actor)); err != nil {
			return err
		}
		result = r.view(actor).(QueueRequest)
		return nil
	})
	return result, err
}

func (s *Service) SubmitResult(ctx context.Context, token []byte, id string, input ResultInput) (Receipt, error) {
	var result Receipt
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		r, err := loadRequest(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if !policy.IsAssignedLab(actor, r.access()) {
			return s.reject(ctx, tx, actor, r, "lab_result.submit", ErrForbidden)
		}
		if r.Status == "completed" {
			return s.reject(ctx, tx, actor, r, "lab_result.submit", ErrConflict)
		}
		if !policy.CanProcessLabRequest(actor, r.access()) {
			return s.reject(ctx, tx, actor, r, "lab_result.submit", ErrForbidden)
		}
		if r.Status != "in_progress" {
			return s.reject(ctx, tx, actor, r, "lab_result.submit", ErrConflict)
		}
		if err := tx.QueryRow(ctx, `INSERT INTO lab_results(lab_request_id,value,unit,reference_range,flag,note,submitted_by)
            VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id::text,submitted_at`, id, input.Value, input.Unit, input.ReferenceRange, input.Flag, input.Note, actor.ID).Scan(&result.ID, &result.SubmittedAt); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE lab_requests SET status='completed',completed_at=$2,updated_at=clock_timestamp() WHERE id=$1`, id, result.SubmittedAt); err != nil {
			return err
		}
		result.LabRequestID = id
		result.Status = "completed"
		return s.Audit.Append(ctx, tx, actor, audit.Event{ChainID: r.patientID, Action: "lab_result.submit", ResourceType: "lab_result", ResourceID: result.ID, Outcome: "allow", Basis: accessBasis(actor)})
	})
	return result, err
}

func (s *Service) GetResult(ctx context.Context, token []byte, id string) (Result, error) {
	var result Result
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		var requestID string
		err := tx.QueryRow(ctx, `SELECT lab_request_id::text FROM lab_results WHERE id=$1`, id).Scan(&requestID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		r, err := loadRequest(ctx, tx, requestID, false)
		if err != nil {
			return err
		}
		event := audit.Event{ChainID: r.patientID, Action: "lab_result.view", ResourceType: "lab_result", ResourceID: id, Outcome: "allow", Basis: accessBasis(actor)}
		allowed := policy.CanViewLabResult(actor, r.access())
		if !allowed && r.Status == "completed" {
			grant, err := policy.FindLabResultConsent(ctx, tx, actor, r.episode.PatientUserID, id, s.now())
			if err != nil {
				return err
			}
			allowed = policy.CanUseLabResultConsent(actor, grant, r.episode.PatientUserID, id, s.now())
			if allowed {
				event.Basis = "consent"
				event.ConsentID = grant.ID
				event.ConsentExpiresAt = &grant.ExpiresAt
			}
		}
		if !allowed {
			event.Outcome = "deny"
			event.Basis = "no_access"
			if err := s.Audit.Append(ctx, tx, actor, event); err != nil {
				return err
			}
			return ErrForbidden
		}
		if err := tx.QueryRow(ctx, `SELECT id::text,lab_request_id::text,value,unit,reference_range,flag,note,submitted_at FROM lab_results WHERE id=$1`, id).Scan(&result.ID, &result.LabRequestID, &result.Value, &result.Unit, &result.ReferenceRange, &result.Flag, &result.Note, &result.SubmittedAt); err != nil {
			return err
		}
		result.TestCode = r.TestCode
		result.TestName = r.TestName
		return s.Audit.Append(ctx, tx, actor, event)
	})
	return result, err
}
