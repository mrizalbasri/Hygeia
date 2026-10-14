package admin

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

type Service struct{ DB apiutil.DB }
type Account struct {
	ID             string     `json:"id"`
	Email          string     `json:"email"`
	FullName       string     `json:"full_name"`
	Role           string     `json:"role"`
	OrganizationID *string    `json:"organization_id"`
	PatientRef     *string    `json:"patient_ref"`
	DisabledAt     *time.Time `json:"disabled_at"`
}
type CreateAccount struct {
	Email          string `json:"email"`
	FullName       string `json:"full_name"`
	Password       string `json:"password"`
	Role           string `json:"role"`
	OrganizationID string `json:"organization_id"`
}
type UpdateAccount struct {
	FullName *string `json:"full_name"`
	Disabled *bool   `json:"disabled"`
}
type Activity struct {
	ID         int64     `json:"id"`
	ActorID    string    `json:"actor_id"`
	Action     string    `json:"action"`
	ResourceID string    `json:"resource_id"`
	OccurredAt time.Time `json:"occurred_at"`
}
type Chain struct {
	ID         string `json:"chain_id"`
	PatientRef string `json:"patient_ref"`
	Events     int64  `json:"events"`
}

func (s *Service) transact(ctx context.Context, token []byte, mutate bool, fn func(context.Context, pgx.Tx, auth.User) error) error {
	return apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if actor.Role != "admin" {
			return apiutil.ErrForbidden
		}
		if mutate {
			// Serializes account changes, including competing disables. Recheck identity
			// after waiting so a disabled administrator cannot mutate with a stale session.
			if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(73194729)`); err != nil {
				return err
			}
			current, err := (auth.Store{DB: tx}).UserBySession(ctx, token)
			if errors.Is(err, pgx.ErrNoRows) {
				return apiutil.ErrUnauthenticated
			}
			if err != nil {
				return err
			}
			if current.Role != "admin" {
				return apiutil.ErrForbidden
			}
			actor = current
		}
		return fn(ctx, tx, actor)
	})
}
func record(ctx context.Context, tx pgx.Tx, actor auth.User, action, id string) error {
	_, err := tx.Exec(ctx, `INSERT INTO admin_activity(actor_id,action,resource_id) VALUES($1,$2,$3)`, actor.ID, action, id)
	return err
}

const accountSelect = `SELECT u.id::text,u.email,u.full_name,u.role,m.organization_id::text,p.patient_ref,u.disabled_at FROM users u LEFT JOIN organization_memberships m ON m.user_id=u.id LEFT JOIN patients p ON p.user_id=u.id `

func scanAccount(row interface{ Scan(...any) error }) (Account, error) {
	var a Account
	err := row.Scan(&a.ID, &a.Email, &a.FullName, &a.Role, &a.OrganizationID, &a.PatientRef, &a.DisabledAt)
	return a, err
}

func (s *Service) Users(ctx context.Context, token []byte, limit, offset int) ([]Account, error) {
	result := make([]Account, 0)
	err := s.transact(ctx, token, false, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		rows, err := tx.Query(ctx, accountSelect+`ORDER BY u.created_at DESC,u.id DESC LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			a, err := scanAccount(rows)
			if err != nil {
				return err
			}
			result = append(result, a)
		}
		return rows.Err()
	})
	return result, err
}
func organizationType(role string) string {
	switch role {
	case "doctor":
		return "clinic"
	case "lab":
		return "laboratory"
	case "pharmacy":
		return "pharmacy"
	}
	return ""
}
func ValidCreate(input CreateAccount) bool {
	if !apiutil.Text(input.FullName, 1, 200) || len(input.Password) < 12 || len(input.Password) > 1024 || len(input.Email) > 254 || strings.ContainsAny(input.Email, " \t\r\n\x00") || strings.Count(input.Email, "@") != 1 {
		return false
	}
	parts := strings.Split(input.Email, "@")
	if parts[0] == "" || parts[1] == "" {
		return false
	}
	switch input.Role {
	case "patient", "admin":
		return input.OrganizationID == ""
	case "doctor", "lab", "pharmacy":
		return apiutil.UUID(input.OrganizationID)
	default:
		return false
	}
}
func (s *Service) CreateUser(ctx context.Context, token []byte, input CreateAccount) (Account, error) {
	var result Account
	if !ValidCreate(input) {
		return result, apiutil.ErrInvalid
	}
	err := s.transact(ctx, token, true, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if input.OrganizationID != "" {
			var kind string
			err := tx.QueryRow(ctx, `SELECT type FROM organizations WHERE id=$1 FOR SHARE`, input.OrganizationID).Scan(&kind)
			if errors.Is(err, pgx.ErrNoRows) || err == nil && kind != organizationType(input.Role) {
				return apiutil.ErrInvalid
			}
			if err != nil {
				return err
			}
		}
		hash, err := auth.HashPassword(input.Password)
		if err != nil {
			return err
		}
		var id string
		if err := tx.QueryRow(ctx, `INSERT INTO users(email,password_hash,full_name,role) VALUES($1,$2,$3,$4) RETURNING id::text`, strings.ToLower(input.Email), hash, input.FullName, input.Role).Scan(&id); err != nil {
			return err
		}
		if input.OrganizationID != "" {
			if _, err := tx.Exec(ctx, `INSERT INTO organization_memberships(user_id,organization_id,staff_role) VALUES($1,$2,$3)`, id, input.OrganizationID, input.Role); err != nil {
				return err
			}
		}
		if input.Role == "patient" {
			raw := make([]byte, 12)
			if _, err := rand.Read(raw); err != nil {
				return err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO patients(user_id,patient_ref) VALUES($1,$2)`, id, "HYG-"+strings.ToUpper(hex.EncodeToString(raw))); err != nil {
				return err
			}
		}
		result, err = scanAccount(tx.QueryRow(ctx, accountSelect+`WHERE u.id=$1`, id))
		if err != nil {
			return err
		}
		return record(ctx, tx, actor, "user.create", id)
	})
	return result, err
}
func (s *Service) UpdateUser(ctx context.Context, token []byte, id string, input UpdateAccount) (Account, error) {
	var result Account
	err := s.transact(ctx, token, true, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if !apiutil.UUID(id) || (input.FullName == nil && input.Disabled == nil) || input.FullName != nil && !apiutil.Text(*input.FullName, 1, 200) {
			return apiutil.ErrInvalid
		}
		var exists string
		err := tx.QueryRow(ctx, `SELECT id::text FROM users WHERE id=$1 FOR UPDATE`, id).Scan(&exists)
		if errors.Is(err, pgx.ErrNoRows) {
			return apiutil.ErrForbidden
		}
		if err != nil {
			return err
		}
		if input.Disabled != nil && *input.Disabled && id == actor.ID {
			return apiutil.ErrConflict
		}
		if input.FullName != nil {
			if _, err := tx.Exec(ctx, `UPDATE users SET full_name=$2 WHERE id=$1`, id, *input.FullName); err != nil {
				return err
			}
		}
		if input.Disabled != nil {
			if _, err := tx.Exec(ctx, `UPDATE users SET disabled_at=CASE WHEN $2 THEN COALESCE(disabled_at,clock_timestamp()) ELSE NULL END WHERE id=$1`, id, *input.Disabled); err != nil {
				return err
			}
			if *input.Disabled {
				if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, id); err != nil {
					return err
				}
			}
		}
		result, err = scanAccount(tx.QueryRow(ctx, accountSelect+`WHERE u.id=$1`, id))
		if err != nil {
			return err
		}
		return record(ctx, tx, actor, "user.update", id)
	})
	return result, err
}
func (s *Service) Organizations(ctx context.Context, token []byte, limit, offset int) ([]auth.Organization, error) {
	result := make([]auth.Organization, 0)
	err := s.transact(ctx, token, false, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		rows, err := tx.Query(ctx, `SELECT id::text,name,type FROM organizations ORDER BY name,id LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var o auth.Organization
			if err := rows.Scan(&o.ID, &o.Name, &o.Type); err != nil {
				return err
			}
			result = append(result, o)
		}
		return rows.Err()
	})
	return result, err
}
func (s *Service) SaveOrganization(ctx context.Context, token []byte, id, name, kind string) (auth.Organization, error) {
	var result auth.Organization
	err := s.transact(ctx, token, true, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if !apiutil.Text(name, 1, 200) {
			return apiutil.ErrInvalid
		}
		action := "organization.create"
		if id == "" {
			if kind != "clinic" && kind != "laboratory" && kind != "pharmacy" {
				return apiutil.ErrInvalid
			}
			if err := tx.QueryRow(ctx, `INSERT INTO organizations(name,type) VALUES($1,$2) RETURNING id::text,name,type`, name, kind).Scan(&result.ID, &result.Name, &result.Type); err != nil {
				return err
			}
		} else {
			if !apiutil.UUID(id) {
				return apiutil.ErrInvalid
			}
			action = "organization.rename"
			err := tx.QueryRow(ctx, `UPDATE organizations SET name=$2 WHERE id=$1 RETURNING id::text,name,type`, id, name).Scan(&result.ID, &result.Name, &result.Type)
			if errors.Is(err, pgx.ErrNoRows) {
				return apiutil.ErrForbidden
			}
			if err != nil {
				return err
			}
		}
		return record(ctx, tx, actor, action, result.ID)
	})
	return result, err
}
func (s *Service) Chains(ctx context.Context, token []byte, limit, offset int) ([]Chain, error) {
	result := make([]Chain, 0)
	err := s.transact(ctx, token, false, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		rows, err := tx.Query(ctx, `SELECT p.id::text,p.patient_ref,COALESCE(h.last_seq,0) FROM patients p LEFT JOIN audit_chain_heads h ON h.chain_id=p.id ORDER BY p.patient_ref,p.id LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var c Chain
			if err := rows.Scan(&c.ID, &c.PatientRef, &c.Events); err != nil {
				return err
			}
			result = append(result, c)
		}
		return rows.Err()
	})
	return result, err
}
func (s *Service) Activity(ctx context.Context, token []byte, limit, offset int) ([]Activity, error) {
	result := make([]Activity, 0)
	err := s.transact(ctx, token, false, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		rows, err := tx.Query(ctx, `SELECT id,actor_id::text,action,resource_id::text,occurred_at FROM admin_activity ORDER BY id DESC LIMIT $1 OFFSET $2`, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var a Activity
			if err := rows.Scan(&a.ID, &a.ActorID, &a.Action, &a.ResourceID, &a.OccurredAt); err != nil {
				return err
			}
			result = append(result, a)
		}
		return rows.Err()
	})
	return result, err
}
