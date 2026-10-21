package carepass

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	qrcode "github.com/skip2/go-qrcode"
	"net/url"
	"strings"
	"time"
)

type Service struct {
	DB           apiutil.DB
	Audit        *audit.Writer
	ShareBaseURL string
	Now          func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// ShareBase accepts an origin, with HTTPS required in production. Limiting the
// origin also bounds QR size and prevents credentials or arbitrary redirects.
func ShareBase(value string, production bool) (string, error) {
	if value == "" && !production {
		return "http://localhost:5173", nil
	}
	u, err := url.Parse(value)
	if err != nil || len(value) > 512 || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && u.Scheme != "http") || (production && u.Scheme != "https") {
		return "", errors.New("PUBLIC_APP_URL must be an origin; HTTPS is required in production")
	}
	return strings.TrimSuffix(u.String(), "/"), nil
}

type Pass struct {
	ID                       string     `json:"id"`
	ResourceType             string     `json:"resource_type"`
	ResourceID               string     `json:"resource_id"`
	CreatedAt                time.Time  `json:"created_at"`
	ExpiresAt                time.Time  `json:"expires_at"`
	RevokedAt                *time.Time `json:"revoked_at,omitempty"`
	GranteeID                *string    `json:"grantee_user_id,omitempty"`
	GranteeOrgID             *string    `json:"grantee_org_id,omitempty"`
	RedeemedAt               *time.Time `json:"redeemed_at,omitempty"`
	patientID, patientUserID string
}
type Created struct {
	Pass     Pass   `json:"carepass"`
	Token    string `json:"token"`
	ShareURL string `json:"share_url"`
	QR       string `json:"qr_png_data_uri"`
}

const selection = `SELECT c.id::text,c.resource_type,c.resource_id::text,c.created_at,c.expires_at,c.revoked_at,c.grantee_user_id::text,c.grantee_org_id::text,c.redeemed_at,p.id::text,p.user_id::text FROM carepasses c JOIN patients p ON p.id=c.patient_id `

func scan(row interface{ Scan(...any) error }) (Pass, error) {
	var c Pass
	err := row.Scan(&c.ID, &c.ResourceType, &c.ResourceID, &c.CreatedAt, &c.ExpiresAt, &c.RevokedAt, &c.GranteeID, &c.GranteeOrgID, &c.RedeemedAt, &c.patientID, &c.patientUserID)
	return c, err
}
func (s *Service) record(ctx context.Context, tx pgx.Tx, a auth.User, c Pass, action, outcome string) error {
	return s.Audit.Append(ctx, tx, a, audit.Event{ChainID: c.patientID, Action: action, ResourceType: "carepass", ResourceID: c.ID, Outcome: outcome, Basis: "carepass", CarePassID: c.ID, CarePassExpiresAt: &c.ExpiresAt})
}
func (s *Service) Create(ctx context.Context, token []byte, kind, id string, ttl int) (Created, error) {
	var result Created
	if !apiutil.UUID(id) || (kind != "lab_result" && kind != "prescription") || (ttl != 15 && ttl != 60) {
		return result, apiutil.ErrInvalid
	}
	base, err := ShareBase(s.ShareBaseURL, false)
	if err != nil {
		return result, err
	}
	err = apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, a auth.User) error {
		if a.Role != "patient" {
			return apiutil.ErrForbidden
		}
		var patient, owner string
		query := `SELECT p.id::text,p.user_id::text FROM prescriptions r JOIN care_episodes e ON e.id=r.episode_id JOIN patients p ON p.id=e.patient_id WHERE r.id=$1 FOR SHARE OF e`
		if kind == "lab_result" {
			query = `SELECT p.id::text,p.user_id::text FROM lab_results l JOIN lab_requests r ON r.id=l.lab_request_id JOIN care_episodes e ON e.id=r.episode_id JOIN patients p ON p.id=e.patient_id WHERE l.id=$1 AND r.status='completed' FOR SHARE OF e`
		}
		err := tx.QueryRow(ctx, query, id).Scan(&patient, &owner)
		if errors.Is(err, pgx.ErrNoRows) {
			return apiutil.ErrForbidden
		}
		if err != nil {
			return err
		}
		if owner != a.ID {
			return apiutil.ErrForbidden
		}
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			return err
		}
		secret := base64.RawURLEncoding.EncodeToString(raw)
		hash := sha256.Sum256([]byte(secret))
		shareURL := base + "/carepass#token=" + secret
		png, err := qrcode.Encode(shareURL, qrcode.Medium, 384)
		if err != nil {
			return err
		}
		now := s.now().UTC().Truncate(time.Microsecond)
		var passID string
		if err := tx.QueryRow(ctx, `INSERT INTO carepasses(patient_id,resource_type,resource_id,token_hash,created_at,expires_at) VALUES($1,$2,$3,$4,$5,$6) RETURNING id::text`, patient, kind, id, hash[:], now, now.Add(time.Duration(ttl)*time.Minute)).Scan(&passID); err != nil {
			return err
		}
		c, err := scan(tx.QueryRow(ctx, selection+`WHERE c.id=$1`, passID))
		if err != nil {
			return err
		}
		result = Created{c, secret, shareURL, "data:image/png;base64," + base64.StdEncoding.EncodeToString(png)}
		return s.record(ctx, tx, a, c, "carepass.create", "allow")
	})
	if err != nil {
		return Created{}, err
	}
	return result, nil
}
func (s *Service) Redeem(ctx context.Context, token []byte, secret string) (Pass, error) {
	var result Pass
	raw, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil || len(raw) != 32 || base64.RawURLEncoding.EncodeToString(raw) != secret {
		return result, apiutil.ErrInvalid
	}
	hash := sha256.Sum256([]byte(secret))
	err = apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, a auth.User) error {
		if a.Role != "doctor" || a.Organization == nil || a.Organization.Type != "clinic" {
			return apiutil.ErrForbidden
		}
		c, err := scan(tx.QueryRow(ctx, selection+`WHERE c.token_hash=$1 FOR UPDATE OF c`, hash[:]))
		if errors.Is(err, pgx.ErrNoRows) {
			return apiutil.ErrForbidden
		}
		if err != nil {
			return err
		}
		deny := func() error {
			if err := s.record(ctx, tx, a, c, "carepass.redeem", "deny"); err != nil {
				return err
			}
			return apiutil.ErrForbidden
		}
		if c.RevokedAt != nil || !c.ExpiresAt.After(s.now()) || s.now().Before(c.CreatedAt) {
			return deny()
		}
		if c.GranteeID != nil {
			if *c.GranteeID != a.ID || *c.GranteeOrgID != a.Organization.ID {
				return deny()
			}
			result = c
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE carepasses SET grantee_user_id=$2,grantee_org_id=$3,redeemed_at=$4 WHERE id=$1`, c.ID, a.ID, a.Organization.ID, s.now()); err != nil {
			return err
		}
		result, err = scan(tx.QueryRow(ctx, selection+`WHERE c.id=$1`, c.ID))
		if err != nil {
			return err
		}
		return s.record(ctx, tx, a, result, "carepass.redeem", "allow")
	})
	return result, err
}
func (s *Service) List(ctx context.Context, token []byte, limit, offset int) ([]Pass, error) {
	result := make([]Pass, 0)
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, a auth.User) error {
		if a.Role != "patient" {
			return apiutil.ErrForbidden
		}
		rows, err := tx.Query(ctx, selection+`WHERE p.user_id=$1 ORDER BY c.created_at DESC,c.id LIMIT $2 OFFSET $3`, a.ID, limit, offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			c, err := scan(rows)
			if err != nil {
				return err
			}
			result = append(result, c)
		}
		return rows.Err()
	})
	return result, err
}
func (s *Service) Revoke(ctx context.Context, token []byte, id string) (Pass, error) {
	var result Pass
	if !apiutil.UUID(id) {
		return result, apiutil.ErrInvalid
	}
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, a auth.User) error {
		if a.Role != "patient" {
			return apiutil.ErrForbidden
		}
		c, err := scan(tx.QueryRow(ctx, selection+`WHERE c.id=$1 AND p.user_id=$2 FOR UPDATE OF c`, id, a.ID))
		if errors.Is(err, pgx.ErrNoRows) {
			return apiutil.ErrForbidden
		}
		if err != nil {
			return err
		}
		if c.RevokedAt != nil {
			result = c
			return nil
		}
		if _, err := tx.Exec(ctx, `UPDATE carepasses SET revoked_at=$2 WHERE id=$1`, id, s.now()); err != nil {
			return err
		}
		result, err = scan(tx.QueryRow(ctx, selection+`WHERE c.id=$1`, id))
		if err != nil {
			return err
		}
		return s.record(ctx, tx, a, result, "carepass.revoke", "allow")
	})
	return result, err
}
