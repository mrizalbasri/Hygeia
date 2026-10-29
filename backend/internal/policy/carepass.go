package policy

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"time"
)

type CarePassGrant struct {
	ID        string
	ExpiresAt time.Time
}

func FindCarePassGrant(ctx context.Context, tx pgx.Tx, a auth.User, patientUserID, kind, id string, now func() time.Time) (CarePassGrant, error) {
	var g CarePassGrant
	if a.Role != "doctor" || a.Organization == nil || a.Organization.Type != "clinic" {
		return g, nil
	}
	err := tx.QueryRow(ctx, `SELECT c.id::text,c.expires_at FROM carepasses c JOIN patients p ON p.id=c.patient_id WHERE c.grantee_user_id=$1 AND c.grantee_org_id=$2 AND p.user_id=$3 AND c.resource_type=$4 AND c.resource_id=$5 AND c.revoked_at IS NULL AND c.created_at<=$6 AND c.expires_at>$6 ORDER BY c.expires_at DESC LIMIT 1 FOR SHARE OF c`, a.ID, a.Organization.ID, patientUserID, kind, id, now()).Scan(&g.ID, &g.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return CarePassGrant{}, nil
	}
	if err == nil && !g.ExpiresAt.After(now()) {
		return CarePassGrant{}, nil
	}
	return g, err
}
