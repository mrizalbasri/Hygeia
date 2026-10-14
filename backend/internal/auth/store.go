package auth

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type User struct {
	ID           string        `json:"id"`
	Email        string        `json:"email"`
	FullName     string        `json:"full_name"`
	Role         string        `json:"role"`
	PatientRef   *string       `json:"patient_ref,omitempty"`
	Organization *Organization `json:"organization"`
}

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type DB interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Store struct{ DB DB }

func (s Store) Credentials(ctx context.Context, email string) (string, string, error) {
	var id, hash string
	err := s.DB.QueryRow(ctx, `SELECT id::text, password_hash FROM users WHERE email=$1 AND disabled_at IS NULL`, email).Scan(&id, &hash)
	return id, hash, err
}

func (s Store) CreateSession(ctx context.Context, tokenHash []byte, userID string, expires time.Time) error {
	_, err := s.DB.Exec(ctx, `INSERT INTO sessions(token_hash, user_id, expires_at) VALUES ($1,$2,$3)`, tokenHash, userID, expires)
	return err
}

func (s Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.DB.Exec(ctx, `DELETE FROM sessions WHERE token_hash=$1`, tokenHash)
	return err
}

func (s Store) UserBySession(ctx context.Context, tokenHash []byte) (User, error) {
	var user User
	var orgID, orgName, orgType *string
	err := s.DB.QueryRow(ctx, `
SELECT u.id::text, u.email, u.full_name, u.role, o.id::text, o.name, o.type, p.patient_ref
FROM sessions s JOIN users u ON u.id=s.user_id
LEFT JOIN patients p ON p.user_id=u.id AND u.role='patient'
LEFT JOIN organization_memberships m ON m.user_id=u.id AND m.staff_role=u.role
LEFT JOIN organizations o ON o.id=m.organization_id
WHERE s.token_hash=$1 AND s.expires_at>now() AND u.disabled_at IS NULL`, tokenHash).Scan(
		&user.ID, &user.Email, &user.FullName, &user.Role, &orgID, &orgName, &orgType, &user.PatientRef,
	)
	if err != nil {
		return User{}, err
	}
	if orgID != nil {
		user.Organization = &Organization{ID: *orgID, Name: *orgName, Type: *orgType}
	}
	if (user.Role == "doctor" && (user.Organization == nil || user.Organization.Type != "clinic")) ||
		(user.Role == "lab" && (user.Organization == nil || user.Organization.Type != "laboratory")) ||
		(user.Role == "pharmacy" && (user.Organization == nil || user.Organization.Type != "pharmacy")) {
		return User{}, pgx.ErrNoRows
	}
	return user, nil
}
