package policy

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

type ConsentGrant struct {
	ID             string
	GranteeID      string
	OrganizationID string
	PatientUserID  string
	ResourceID     string
	Status         string
	ExpiresAt      time.Time
}

func CanUseLabResultConsent(actor auth.User, g ConsentGrant, patientUserID, resultID string, now time.Time) bool {
	return actor.Role == "doctor" && actor.Organization != nil && actor.Organization.Type == "clinic" &&
		actor.ID == g.GranteeID && actor.Organization.ID == g.OrganizationID && g.PatientUserID == patientUserID &&
		g.ResourceID == resultID && g.Status == "active" && g.ExpiresAt.After(now)
}

// The shared row lock serializes a read with consent revocation. The caller must
// hold it until the clinical read and audit commit, and recheck the clock after waiting.
func FindLabResultConsent(ctx context.Context, tx pgx.Tx, actor auth.User, patientUserID, resultID string, now time.Time) (ConsentGrant, error) {
	var grant ConsentGrant
	if actor.Role != "doctor" || actor.Organization == nil || actor.Organization.Type != "clinic" {
		return grant, nil
	}
	err := tx.QueryRow(ctx, `SELECT c.id::text,c.grantee_user_id::text,c.grantee_org_id::text,p.user_id::text,c.resource_id::text,c.status,c.expires_at
        FROM consents c JOIN patients p ON p.id=c.patient_id
        WHERE c.grantee_user_id=$1 AND c.grantee_org_id=$2 AND p.user_id=$3 AND c.resource_id=$4
        AND c.resource_type='lab_result' AND c.allowed_action='view' AND c.status='active' AND c.expires_at>$5 FOR SHARE OF c`,
		actor.ID, actor.Organization.ID, patientUserID, resultID, now).Scan(&grant.ID, &grant.GranteeID, &grant.OrganizationID, &grant.PatientUserID, &grant.ResourceID, &grant.Status, &grant.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ConsentGrant{}, nil
	}
	return grant, err
}
