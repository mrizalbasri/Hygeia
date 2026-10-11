package consent

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("invalid consent state")
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
		return s.Now().UTC().Truncate(time.Microsecond)
	}
	return time.Now().UTC().Truncate(time.Microsecond)
}

type Consent struct {
	ID               string     `json:"id"`
	PatientRef       string     `json:"patient_ref"`
	GranteeID        string     `json:"grantee_user_id"`
	GranteeName      string     `json:"grantee_name"`
	OrganizationID   string     `json:"grantee_org_id"`
	OrganizationName string     `json:"grantee_org_name"`
	ResourceType     string     `json:"resource_type"`
	ResourceID       string     `json:"resource_id"`
	AllowedAction    string     `json:"allowed_action"`
	Purpose          string     `json:"purpose"`
	Status           string     `json:"status"`
	RequestedAt      time.Time  `json:"requested_at"`
	DecidedAt        *time.Time `json:"decided_at"`
	ExpiresAt        *time.Time `json:"expires_at"`
	RevokedAt        *time.Time `json:"revoked_at"`
	TestCode         string     `json:"test_code,omitempty"`
	TestName         string     `json:"test_name,omitempty"`
	patientID        string
	patientUserID    string
}

type CreateInput struct {
	PatientRef   string `json:"patient_ref"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Purpose      string `json:"purpose"`
}

const consentSelect = `SELECT c.id::text,p.patient_ref,c.grantee_user_id::text,u.full_name,c.grantee_org_id::text,o.name,
    c.resource_type,c.resource_id::text,c.allowed_action,c.purpose,c.status,c.requested_at,c.decided_at,c.expires_at,c.revoked_at,
    r.test_code,r.test_name,p.id::text,p.user_id::text FROM consents c JOIN patients p ON p.id=c.patient_id
    JOIN users u ON u.id=c.grantee_user_id JOIN organizations o ON o.id=c.grantee_org_id
    JOIN lab_results x ON x.id=c.resource_id JOIN lab_requests r ON r.id=x.lab_request_id `

func scanConsent(row interface{ Scan(...any) error }, now time.Time) (Consent, error) {
	var c Consent
	err := row.Scan(&c.ID, &c.PatientRef, &c.GranteeID, &c.GranteeName, &c.OrganizationID, &c.OrganizationName,
		&c.ResourceType, &c.ResourceID, &c.AllowedAction, &c.Purpose, &c.Status, &c.RequestedAt, &c.DecidedAt, &c.ExpiresAt, &c.RevokedAt,
		&c.TestCode, &c.TestName, &c.patientID, &c.patientUserID)
	if c.Status == "active" && c.ExpiresAt != nil && !c.ExpiresAt.After(now) {
		c.Status = "expired"
	}
	return c, err
}
func (c Consent) visible(actor auth.User) bool {
	return actor.Role == "patient" && actor.ID == c.patientUserID || actor.Role == "doctor" && actor.ID == c.GranteeID && actor.Organization != nil && actor.Organization.ID == c.OrganizationID
}
func (c Consent) view(actor auth.User) Consent {
	if actor.Role != "patient" {
		c.TestCode = ""
		c.TestName = ""
	}
	return c
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

func (s *Service) record(ctx context.Context, tx pgx.Tx, actor auth.User, c Consent, action, outcome, basis string) error {
	return s.Audit.Append(ctx, tx, actor, audit.Event{ChainID: c.patientID, Action: action, ResourceType: "consent", ResourceID: c.ID, Outcome: outcome, Basis: basis})
}
func (s *Service) reject(ctx context.Context, tx pgx.Tx, actor auth.User, c Consent, action string, reason error) error {
	basis := "no_access"
	if errors.Is(reason, ErrConflict) {
		basis = "invalid_state"
	}
	if err := s.record(ctx, tx, actor, c, action, "deny", basis); err != nil {
		return err
	}
	return reason
}

// Episode lock comes before a consent row lock, matching laboratory result reads.
func lockResultEpisode(ctx context.Context, tx pgx.Tx, resultID string) error {
	var episodeID string
	err := tx.QueryRow(ctx, `SELECT r.episode_id::text FROM lab_results x JOIN lab_requests r ON r.id=x.lab_request_id WHERE x.id=$1`, resultID).Scan(&episodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrForbidden
	}
	if err != nil {
		return err
	}
	return tx.QueryRow(ctx, `SELECT id::text FROM care_episodes WHERE id=$1 FOR SHARE`, episodeID).Scan(&episodeID)
}

func (s *Service) Create(ctx context.Context, token []byte, input CreateInput) (Consent, error) {
	var result Consent
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if actor.Role != "doctor" || actor.Organization == nil {
			return ErrForbidden
		}
		if err := lockResultEpisode(ctx, tx, input.ResourceID); err != nil {
			return err
		}
		var patientID string
		err := tx.QueryRow(ctx, `SELECT p.id::text FROM lab_results x JOIN lab_requests r ON r.id=x.lab_request_id
            JOIN care_episodes e ON e.id=r.episode_id JOIN patients p ON p.id=e.patient_id JOIN users u ON u.id=p.user_id
            WHERE x.id=$1 AND p.patient_ref=$2 AND r.status='completed' AND u.role='patient' AND u.disabled_at IS NULL`, input.ResourceID, input.PatientRef).Scan(&patientID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		now := s.now()
		// Persist expiration only when reopening the same request. Reads always
		// compute effective expiration, so access never depends on a cleanup job.
		if _, err := tx.Exec(ctx, `UPDATE consents SET status='expired' WHERE grantee_user_id=$1 AND grantee_org_id=$2 AND resource_id=$3 AND status='active' AND expires_at<=$4`, actor.ID, actor.Organization.ID, input.ResourceID, now); err != nil {
			return err
		}
		var id string
		err = tx.QueryRow(ctx, `INSERT INTO consents(patient_id,grantee_user_id,grantee_org_id,resource_id,purpose)
            VALUES($1,$2,$3,$4,$5) ON CONFLICT(grantee_user_id,grantee_org_id,resource_id) WHERE status IN ('pending','active') DO NOTHING RETURNING id::text`, patientID, actor.ID, actor.Organization.ID, input.ResourceID, input.Purpose).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			existing, err := scanConsent(tx.QueryRow(ctx, consentSelect+`WHERE c.grantee_user_id=$1 AND c.grantee_org_id=$2 AND c.resource_id=$3 AND c.status IN ('pending','active')`, actor.ID, actor.Organization.ID, input.ResourceID), s.now())
			if err != nil {
				return err
			}
			return s.reject(ctx, tx, actor, existing, "consent.request", ErrConflict)
		}
		if err != nil {
			return err
		}
		result, err = scanConsent(tx.QueryRow(ctx, consentSelect+`WHERE c.id=$1`, id), s.now())
		if err != nil {
			return err
		}
		if err := s.record(ctx, tx, actor, result, "consent.request", "allow", "doctor_request"); err != nil {
			return err
		}
		result = result.view(actor)
		return nil
	})
	return result, err
}

func (s *Service) Get(ctx context.Context, token []byte, id string) (Consent, error) {
	var result Consent
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		c, err := scanConsent(tx.QueryRow(ctx, consentSelect+`WHERE c.id=$1`, id), s.now())
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		if !c.visible(actor) {
			return s.reject(ctx, tx, actor, c, "consent.view", ErrForbidden)
		}
		if err := s.record(ctx, tx, actor, c, "consent.view", "allow", "consent_participant"); err != nil {
			return err
		}
		result = c.view(actor)
		return nil
	})
	return result, err
}

func (s *Service) List(ctx context.Context, token []byte, status string, limit, offset int) ([]Consent, error) {
	result := make([]Consent, 0)
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		now := s.now()
		args := []any{actor.ID, status, now, limit, offset}
		condition := "p.user_id=$1"
		if actor.Role == "doctor" && actor.Organization != nil {
			condition = `c.grantee_user_id=$1 AND c.grantee_org_id=$6`
			args = append(args, actor.Organization.ID)
		} else if actor.Role != "patient" {
			return ErrForbidden
		}
		rows, err := tx.Query(ctx, consentSelect+`WHERE `+condition+` AND ($2='' OR (CASE WHEN c.status='active' AND c.expires_at<=$3 THEN 'expired' ELSE c.status END)=$2) ORDER BY c.requested_at DESC,c.id DESC LIMIT $4 OFFSET $5`, args...)
		if err != nil {
			return err
		}
		chains := map[string]Consent{}
		for rows.Next() {
			c, err := scanConsent(rows, now)
			if err != nil {
				rows.Close()
				return err
			}
			result = append(result, c.view(actor))
			chains[c.patientID] = c
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		ids := make([]string, 0, len(chains))
		for id := range chains {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			if err := s.record(ctx, tx, actor, chains[id], "consent.list", "allow", "consent_participant"); err != nil {
				return err
			}
		}
		return nil
	})
	return result, err
}

func (s *Service) Decide(ctx context.Context, token []byte, id, action string, hours int) (Consent, error) {
	var result Consent
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		var resourceID string
		err := tx.QueryRow(ctx, `SELECT resource_id::text FROM consents WHERE id=$1`, id).Scan(&resourceID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrForbidden
		}
		if err != nil {
			return err
		}
		if err := lockResultEpisode(ctx, tx, resourceID); err != nil {
			return err
		}
		c, err := scanConsent(tx.QueryRow(ctx, consentSelect+`WHERE c.id=$1 FOR UPDATE OF c`, id), s.now())
		if err != nil {
			return err
		}
		event := "consent." + action
		if actor.Role != "patient" || actor.ID != c.patientUserID {
			return s.reject(ctx, tx, actor, c, event, ErrForbidden)
		}
		now := s.now()
		// The row lock may have waited across the expiry boundary.
		if c.Status == "active" && c.ExpiresAt != nil && !c.ExpiresAt.After(now) {
			c.Status = "expired"
		}
		switch {
		case action == "approve" && c.Status == "pending" && (hours == 1 || hours == 24 || hours == 168):
			_, err = tx.Exec(ctx, `UPDATE consents SET status='active',decided_at=$2,expires_at=$3 WHERE id=$1`, id, now, now.Add(time.Duration(hours)*time.Hour))
		case action == "deny" && c.Status == "pending" && hours == 0:
			_, err = tx.Exec(ctx, `UPDATE consents SET status='denied',decided_at=$2 WHERE id=$1`, id, now)
		case action == "revoke" && c.Status == "active" && hours == 0:
			_, err = tx.Exec(ctx, `UPDATE consents SET status='revoked',revoked_at=$2 WHERE id=$1`, id, now)
		default:
			return s.reject(ctx, tx, actor, c, event, ErrConflict)
		}
		if err != nil {
			return err
		}
		result, err = scanConsent(tx.QueryRow(ctx, consentSelect+`WHERE c.id=$1`, id), s.now())
		if err != nil {
			return err
		}
		return s.record(ctx, tx, actor, result, event, "allow", "patient_self")
	})
	return result, err
}
