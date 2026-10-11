package policy

import (
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"testing"
	"time"
)

func TestLabResultConsentScope(t *testing.T) {
	now := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	actor := auth.User{ID: "doctor", Role: "doctor", Organization: &auth.Organization{ID: "clinic", Type: "clinic"}}
	grant := ConsentGrant{ID: "consent", GranteeID: "doctor", OrganizationID: "clinic", PatientUserID: "patient", ResourceID: "result", Status: "active", ExpiresAt: now.Add(time.Hour)}
	if !CanUseLabResultConsent(actor, grant, "patient", "result", now) {
		t.Fatal("valid exact grant denied")
	}
	tests := []struct {
		name   string
		change func(*auth.User, *ConsentGrant)
	}{
		{"other doctor", func(a *auth.User, g *ConsentGrant) { a.ID = "another" }},
		{"other clinic", func(a *auth.User, g *ConsentGrant) { a.Organization = &auth.Organization{ID: "other", Type: "clinic"} }},
		{"no membership", func(a *auth.User, g *ConsentGrant) { a.Organization = nil }},
		{"lab", func(a *auth.User, g *ConsentGrant) { a.Role = "lab" }},
		{"non clinic", func(a *auth.User, g *ConsentGrant) {
			a.Organization = &auth.Organization{ID: "clinic", Type: "laboratory"}
		}},
		{"other patient", func(a *auth.User, g *ConsentGrant) { g.PatientUserID = "other" }},
		{"other result", func(a *auth.User, g *ConsentGrant) { g.ResourceID = "other" }},
		{"pending", func(a *auth.User, g *ConsentGrant) { g.Status = "pending" }},
		{"revoked", func(a *auth.User, g *ConsentGrant) { g.Status = "revoked" }},
		{"exact expiry", func(a *auth.User, g *ConsentGrant) { g.ExpiresAt = now }},
		{"past expiry", func(a *auth.User, g *ConsentGrant) { g.ExpiresAt = now.Add(-time.Second) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a, g := actor, grant
			tc.change(&a, &g)
			if CanUseLabResultConsent(a, g, "patient", "result", now) {
				t.Fatal("scope escaped")
			}
		})
	}
}
