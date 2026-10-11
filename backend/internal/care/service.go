package care

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/laboratory"
	"github.com/mrizalbasri/Hygeia/backend/internal/policy"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("invalid episode state")
)

type DB interface {
	Begin(context.Context) (pgx.Tx, error)
}
type Service struct {
	DB    DB
	Audit *audit.Writer
}

type Episode struct {
	ID               string     `json:"id"`
	PatientID        string     `json:"patient_id"`
	PatientRef       string     `json:"patient_ref"`
	DoctorID         string     `json:"doctor_id"`
	DoctorName       string     `json:"doctor_name"`
	OrganizationID   string     `json:"organization_id"`
	OrganizationName string     `json:"organization_name"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	AcceptedAt       *time.Time `json:"accepted_at"`
	ClosedAt         *time.Time `json:"closed_at"`
	patientUserID    string
}

func (e Episode) access() policy.Episode {
	return policy.Episode{DoctorID: e.DoctorID, PatientUserID: e.patientUserID, OrganizationID: e.OrganizationID, Status: e.Status}
}

const episodeSelect = `SELECT e.id::text,e.patient_id::text,p.patient_ref,e.doctor_id::text,d.full_name,
    e.organization_id::text,o.name,e.status,e.created_at,e.updated_at,e.accepted_at,e.closed_at,p.user_id::text
    FROM care_episodes e JOIN patients p ON p.id=e.patient_id
    JOIN users d ON d.id=e.doctor_id JOIN organizations o ON o.id=e.organization_id `

func scanEpisode(row interface{ Scan(...any) error }) (Episode, error) {
	var e Episode
	err := row.Scan(&e.ID, &e.PatientID, &e.PatientRef, &e.DoctorID, &e.DoctorName, &e.OrganizationID, &e.OrganizationName,
		&e.Status, &e.CreatedAt, &e.UpdatedAt, &e.AcceptedAt, &e.ClosedAt, &e.patientUserID)
	return e, err
}

// transact resolves identity from the database for every operation. Business
// rejections commit their audit records; SQL/audit failures roll back everything.
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

func (s *Service) record(ctx context.Context, tx pgx.Tx, actor auth.User, e Episode, action, outcome, basis string) error {
	return s.Audit.Append(ctx, tx, actor, audit.Event{ChainID: e.PatientID, Action: action, ResourceType: "care_episode", ResourceID: e.ID, Outcome: outcome, Basis: basis})
}

func (s *Service) reject(ctx context.Context, tx pgx.Tx, actor auth.User, e Episode, action string, reason error) error {
	basis := "no_access"
	if errors.Is(reason, ErrConflict) {
		basis = "invalid_state"
	}
	if err := s.record(ctx, tx, actor, e, action, "deny", basis); err != nil {
		return err
	}
	return reason
}

func basis(actor auth.User) string {
	if actor.Role == "patient" {
		return "patient_self"
	}
	return "episode_doctor"
}

func (s *Service) Invite(ctx context.Context, token []byte, patientRef string) (Episode, error) {
	var result Episode
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if actor.Role != "doctor" || actor.Organization == nil || actor.Organization.Type != "clinic" {
			return ErrForbidden
		}
		var patientID string
		err := tx.QueryRow(ctx, `SELECT p.id::text FROM patients p JOIN users u ON u.id=p.user_id WHERE p.patient_ref=$1 AND u.role='patient' AND u.disabled_at IS NULL`, patientRef).Scan(&patientID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		var id string
		err = tx.QueryRow(ctx, `INSERT INTO care_episodes(patient_id,doctor_id,organization_id) VALUES($1,$2,$3)
            ON CONFLICT (doctor_id,patient_id) WHERE status IN ('pending','active') DO NOTHING RETURNING id::text`, patientID, actor.ID, actor.Organization.ID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			existing, err := scanEpisode(tx.QueryRow(ctx, episodeSelect+`WHERE e.patient_id=$1 AND e.doctor_id=$2 AND e.status IN ('pending','active')`, patientID, actor.ID))
			if err != nil {
				return err
			}
			return s.reject(ctx, tx, actor, existing, "episode.invite", ErrConflict)
		}
		if err != nil {
			return err
		}
		result, err = scanEpisode(tx.QueryRow(ctx, episodeSelect+`WHERE e.id=$1`, id))
		if err != nil {
			return err
		}
		return s.record(ctx, tx, actor, result, "episode.invite", "allow", "doctor_invitation")
	})
	return result, err
}

func (s *Service) Get(ctx context.Context, token []byte, id string) (Episode, error) {
	var result Episode
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		e, err := scanEpisode(tx.QueryRow(ctx, episodeSelect+`WHERE e.id=$1`, id))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		if !policy.CanViewEpisode(actor, e.access()) {
			return s.reject(ctx, tx, actor, e, "episode.view", ErrForbidden)
		}
		if err := s.record(ctx, tx, actor, e, "episode.view", "allow", basis(actor)); err != nil {
			return err
		}
		result = e
		return nil
	})
	return result, err
}

func (s *Service) Transition(ctx context.Context, token []byte, id, action string) (Episode, error) {
	var result Episode
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		e, err := scanEpisode(tx.QueryRow(ctx, episodeSelect+`WHERE e.id=$1 FOR UPDATE OF e`, id))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		event := "episode." + action
		if !policy.CanTransitionEpisode(actor, e.access(), action) {
			return s.reject(ctx, tx, actor, e, event, ErrForbidden)
		}
		next := ""
		switch {
		case e.Status == "pending" && action == "accept":
			next = "active"
		case e.Status == "pending" && action == "reject":
			next = "rejected"
		case e.Status == "active" && action == "end":
			next = "ended"
		default:
			return s.reject(ctx, tx, actor, e, event, ErrConflict)
		}
		if _, err := tx.Exec(ctx, `UPDATE care_episodes SET status=$2,updated_at=clock_timestamp(),
            accepted_at=CASE WHEN $2='active' THEN clock_timestamp() ELSE accepted_at END,
            closed_at=CASE WHEN $2 IN ('rejected','ended') THEN clock_timestamp() ELSE NULL END WHERE id=$1`, id, next); err != nil {
			return err
		}
		result, err = scanEpisode(tx.QueryRow(ctx, episodeSelect+`WHERE e.id=$1`, id))
		if err != nil {
			return err
		}
		if next == "ended" {
			if err := laboratory.CancelPending(ctx, tx, s.Audit, actor, id, result.PatientID); err != nil {
				return err
			}
		}
		return s.record(ctx, tx, actor, result, event, "allow", basis(actor))
	})
	return result, err
}

func (s *Service) List(ctx context.Context, token []byte, status string, limit, offset int) ([]Episode, error) {
	result := make([]Episode, 0)
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		var condition string
		switch actor.Role {
		case "patient":
			condition = `p.user_id=$1`
		case "doctor":
			if actor.Organization == nil {
				return ErrForbidden
			}
			condition = `e.doctor_id=$1 AND e.organization_id=$5`
		default:
			return ErrForbidden
		}
		args := []any{actor.ID, status, limit, offset}
		if actor.Role == "doctor" {
			args = append(args, actor.Organization.ID)
		}
		rows, err := tx.Query(ctx, episodeSelect+`WHERE `+condition+` AND ($2='' OR e.status=$2) ORDER BY e.created_at DESC,e.id DESC LIMIT $3 OFFSET $4`, args...)
		if err != nil {
			return err
		}
		for rows.Next() {
			e, err := scanEpisode(rows)
			if err != nil {
				rows.Close()
				return err
			}
			result = append(result, e)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		// Deterministic order avoids deadlocks when a page spans several patients.
		chains := make(map[string]Episode)
		for _, e := range result {
			chains[e.PatientID] = e
		}
		ids := make([]string, 0, len(chains))
		for id := range chains {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if err := s.record(ctx, tx, actor, chains[id], "episode.list", "allow", basis(actor)); err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

func patientChain(ctx context.Context, tx pgx.Tx, actor auth.User) (string, error) {
	if actor.Role != "patient" {
		return "", ErrForbidden
	}
	var id string
	err := tx.QueryRow(ctx, `SELECT id::text FROM patients WHERE user_id=$1`, actor.ID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrForbidden
	}
	return id, err
}

func (s *Service) AccessSummary(ctx context.Context, token []byte, limit, offset int) ([]audit.Summary, error) {
	var result []audit.Summary
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		chain, err := patientChain(ctx, tx, actor)
		if err != nil {
			return err
		}
		result, err = audit.Summaries(ctx, tx, chain, limit, offset)
		return err
	})
	return result, err
}

func (s *Service) Verify(ctx context.Context, token []byte, requestedChain string) (audit.Verification, error) {
	var result audit.Verification
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		chain := requestedChain
		if actor.Role == "admin" {
			if chain == "" {
				return ErrForbidden
			}
			var exists bool
			if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM patients WHERE id=$1)`, chain).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return ErrForbidden
			}
		} else {
			own, err := patientChain(ctx, tx, actor)
			if err != nil {
				return err
			}
			if chain != "" && chain != own {
				return ErrForbidden
			}
			chain = own
		}
		var err error
		result, err = s.Audit.Verify(ctx, tx, chain)
		return err
	})
	return result, err
}
