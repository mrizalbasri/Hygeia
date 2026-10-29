package policy

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"time"
)

type EmergencyGrant struct {
	ID        string
	ExpiresAt time.Time
}

// FindEmergencyGrant binds access to the initiating doctor and current clinic.
// Hold the shared lock through the clinical read and audit commit. Check time
// after the query, since acquiring a row lock can wait beyond grant expiry.
func FindEmergencyGrant(ctx context.Context, tx pgx.Tx, actor auth.User, patientUserID string, now func() time.Time) (EmergencyGrant, error) {
	var g EmergencyGrant
	if actor.Role != "doctor" || actor.Organization == nil || actor.Organization.Type != "clinic" {
		return g, nil
	}
	err := tx.QueryRow(ctx, `SELECT g.id::text,g.expires_at FROM break_glass_sessions g JOIN patients p ON p.id=g.patient_id
 WHERE g.doctor_user_id=$1 AND g.organization_id=$2 AND p.user_id=$3 AND g.starts_at<=$4 AND g.expires_at>$4
 ORDER BY g.expires_at DESC LIMIT 1 FOR SHARE OF g`, actor.ID, actor.Organization.ID, patientUserID, now()).Scan(&g.ID, &g.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return EmergencyGrant{}, nil
	}
	if err == nil && !g.ExpiresAt.After(now()) {
		return EmergencyGrant{}, nil
	}
	return g, err
}
