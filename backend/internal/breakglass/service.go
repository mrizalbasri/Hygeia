// Package breakglass implements bounded, accountable emergency read access.
package breakglass

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"strings"
	"time"
)

type Service struct {
	DB    apiutil.DB
	Audit *audit.Writer
	Now   func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

type Session struct {
	ID                       string     `json:"id"`
	PatientRef               string     `json:"patient_ref"`
	DoctorID                 string     `json:"doctor_user_id"`
	OrganizationID           string     `json:"organization_id"`
	ReasonCode               string     `json:"reason_code"`
	ReasonText               string     `json:"reason_text"`
	StartsAt                 time.Time  `json:"starts_at"`
	ExpiresAt                time.Time  `json:"expires_at"`
	ReviewedBy               *string    `json:"reviewed_by,omitempty"`
	ReviewedAt               *time.Time `json:"reviewed_at,omitempty"`
	ReviewDecision           *string    `json:"review_decision,omitempty"`
	ReviewNote               *string    `json:"review_note,omitempty"`
	patientID, patientUserID string
}

const selection = `SELECT g.id::text,p.patient_ref,g.doctor_user_id::text,g.organization_id::text,g.reason_code,g.reason_text,g.starts_at,g.expires_at,g.reviewed_by::text,g.reviewed_at,g.review_decision,g.review_note,p.id::text,p.user_id::text FROM break_glass_sessions g JOIN patients p ON p.id=g.patient_id `

func scan(row interface{ Scan(...any) error }) (Session, error) {
	var g Session
	err := row.Scan(&g.ID, &g.PatientRef, &g.DoctorID, &g.OrganizationID, &g.ReasonCode, &g.ReasonText, &g.StartsAt, &g.ExpiresAt, &g.ReviewedBy, &g.ReviewedAt, &g.ReviewDecision, &g.ReviewNote, &g.patientID, &g.patientUserID)
	return g, err
}
func doctor(a auth.User) bool {
	return a.Role == "doctor" && a.Organization != nil && a.Organization.Type == "clinic"
}
func owns(a auth.User, g Session) bool {
	return doctor(a) && a.ID == g.DoctorID && a.Organization.ID == g.OrganizationID
}
func (s *Service) record(ctx context.Context, tx pgx.Tx, a auth.User, g Session, action, outcome string) error {
	return s.Audit.Append(ctx, tx, a, audit.Event{ChainID: g.patientID, Action: action, ResourceType: "break_glass_session", ResourceID: g.ID, Outcome: outcome, Basis: "break_glass", BreakGlassID: g.ID, BreakGlassExpiresAt: &g.ExpiresAt})
}
func (s *Service) Start(ctx context.Context, token []byte, ref, code, reason string) (Session, error) {
	var result Session
	if !apiutil.Text(ref, 1, 128) || !apiutil.Text(reason, 10, 1000) || (code != "unconscious" && code != "unreachable_patient" && code != "life_threatening") {
		return result, apiutil.ErrInvalid
	}
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, a auth.User) error {
		if !doctor(a) {
			return apiutil.ErrForbidden
		}
		var patient, user string
		err := tx.QueryRow(ctx, `SELECT p.id::text,p.user_id::text FROM patients p JOIN users u ON u.id=p.user_id WHERE p.patient_ref=$1 AND u.disabled_at IS NULL FOR UPDATE OF p`, ref).Scan(&patient, &user)
		if errors.Is(err, pgx.ErrNoRows) {
			return apiutil.ErrForbidden
		}
		if err != nil {
			return err
		}
		now := s.now().UTC().Truncate(time.Microsecond)
		var duplicate bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM break_glass_sessions WHERE patient_id=$1 AND doctor_user_id=$2 AND organization_id=$3 AND expires_at>$4)`, patient, a.ID, a.Organization.ID, now).Scan(&duplicate); err != nil {
			return err
		}
		if duplicate {
			return apiutil.ErrConflict
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO break_glass_sessions(patient_id,doctor_user_id,organization_id,reason_code,reason_text,starts_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id::text`, patient, a.ID, a.Organization.ID, code, strings.TrimSpace(reason), now, now.Add(time.Hour)).Scan(&id); err != nil {
			return err
		}
		result, err = scan(tx.QueryRow(ctx, selection+`WHERE g.id=$1`, id))
		if err != nil {
			return err
		}
		return s.record(ctx, tx, a, result, "break_glass.start", "allow")
	})
	return result, err
}
func (s *Service) List(ctx context.Context, token []byte, admin bool, limit, offset int) ([]Session, error) {
	result := make([]Session, 0)
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, a auth.User) error {
		condition := `p.user_id=$1`
		org := ""
		if admin {
			if a.Role != "admin" {
				return apiutil.ErrForbidden
			}
			condition = `$1::uuid IS NOT NULL`
		} else if doctor(a) {
			condition = `g.doctor_user_id=$1 AND g.organization_id=$2::uuid`
			org = a.Organization.ID
		} else if a.Role != "patient" {
			return apiutil.ErrForbidden
		}
		rows, err := tx.Query(ctx, selection+`WHERE `+condition+` AND ($2::text IS NOT NULL) ORDER BY g.starts_at DESC,g.id LIMIT $3 OFFSET $4`, a.ID, org, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			g, err := scan(rows)
			if err != nil {
				return err
			}
			result = append(result, g)
		}
		return rows.Err()
	})
	return result, err
}
func (s *Service) Review(ctx context.Context, token []byte, id, decision, note string) (Session, error) {
	var result Session
	if !apiutil.UUID(id) || !apiutil.Text(note, 10, 1000) || (decision != "appropriate" && decision != "needs_followup") {
		return result, apiutil.ErrInvalid
	}
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, a auth.User) error {
		if a.Role != "admin" {
			return apiutil.ErrForbidden
		}
		g, err := scan(tx.QueryRow(ctx, selection+`WHERE g.id=$1 FOR UPDATE OF g`, id))
		if errors.Is(err, pgx.ErrNoRows) {
			return apiutil.ErrForbidden
		}
		if err != nil {
			return err
		}
		if g.ReviewedAt != nil {
			return apiutil.ErrConflict
		}
		if _, err := tx.Exec(ctx, `UPDATE break_glass_sessions SET reviewed_by=$2,reviewed_at=$3,review_decision=$4,review_note=$5 WHERE id=$1`, id, a.ID, s.now(), decision, strings.TrimSpace(note)); err != nil {
			return err
		}
		result, err = scan(tx.QueryRow(ctx, selection+`WHERE g.id=$1`, id))
		if err != nil {
			return err
		}
		return s.record(ctx, tx, a, result, "break_glass.review", "allow")
	})
	return result, err
}

type Resource struct {
	ID   string `json:"id"`
	Type string `json:"resource_type"`
}

func (s *Service) Resources(ctx context.Context, token []byte, id string, limit, offset int) ([]Resource, error) {
	result := make([]Resource, 0)
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, a auth.User) error {
		g, err := scan(tx.QueryRow(ctx, selection+`WHERE g.id=$1 FOR SHARE OF g`, id))
		if errors.Is(err, pgx.ErrNoRows) {
			return apiutil.ErrForbidden
		}
		if err != nil {
			return err
		}
		if !owns(a, g) || s.now().Before(g.StartsAt) || !s.now().Before(g.ExpiresAt) {
			if err := s.record(ctx, tx, a, g, "break_glass.resources", "deny"); err != nil {
				return err
			}
			return apiutil.ErrForbidden
		}
		if err := audit.LockChain(ctx, tx, g.patientID); err != nil {
			return err
		}
		if !s.now().Before(g.ExpiresAt) {
			if err := s.record(ctx, tx, a, g, "break_glass.resources", "deny"); err != nil {
				return err
			}
			return apiutil.ErrForbidden
		}
		rows, err := tx.Query(ctx, `SELECT id,resource_type FROM (
 SELECT l.id::text AS id,'lab_result' AS resource_type FROM lab_results l JOIN lab_requests r ON r.id=l.lab_request_id JOIN care_episodes e ON e.id=r.episode_id WHERE e.patient_id=$1 AND r.status='completed'
 UNION ALL SELECT r.id::text,'prescription' FROM prescriptions r JOIN care_episodes e ON e.id=r.episode_id WHERE e.patient_id=$1 AND r.status IN ('sent','accepted','preparing','ready')) resources ORDER BY resource_type,id LIMIT $2 OFFSET $3`, g.patientID, limit, offset)
		if err != nil {
			return err
		}
		for rows.Next() {
			var r Resource
			if err := rows.Scan(&r.ID, &r.Type); err != nil {
				rows.Close()
				return err
			}
			result = append(result, r)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		return s.record(ctx, tx, a, g, "break_glass.resources", "allow")
	})
	return result, err
}
