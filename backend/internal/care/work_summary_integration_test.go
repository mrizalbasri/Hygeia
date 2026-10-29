package care

import (
	"context"
	"encoding/json"
	"testing"
)

func TestWorkSummaryIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	path := "/api/v1/patient/work-summary"
	assertSummary := func(actor string, want WorkSummary) {
		t.Helper()
		response := f.request(actor, "GET", path, "", false)
		expectStatus(t, response, 200)
		var body struct {
			Summary WorkSummary `json:"summary"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body.Summary != want {
			t.Fatalf("%s summary: got %+v, want %+v", actor, body.Summary, want)
		}
	}
	assertSummary("patient", WorkSummary{})
	episode, _, result := completedConsentResult(t, f)
	grant := requestConsent(t, f, result.ID)
	// A second doctor's invitation remains pending alongside the active episode.
	if _, err := f.service.Invite(ctx, f.tokens["colleague"], "TEST-patient"); err != nil {
		t.Fatal(err)
	}
	// More than a full list page: counts must not depend on a limit or newest six.
	for i := 0; i < 25; i++ {
		newLabRequest(t, f, episode)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO prescriptions(episode_id) SELECT $1 FROM generate_series(1,23)`, episode.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `INSERT INTO prescriptions(episode_id,pharmacy_org_id,status,approved_at)
		VALUES($1,$2,'ready',now())`, episode.ID, f.ids["pharmacyOrg"]); err != nil {
		t.Fatal(err)
	}
	processing := newLabRequest(t, f, episode)
	approveLab(t, f, processing.ID)
	want := WorkSummary{
		ActiveEpisodes: 1, LabsInProgress: 1, PrescriptionsReady: 1,
		Pending: PendingWork{Episodes: 1, LabRequests: 25, Prescriptions: 23, Consents: 1},
	}
	assertSummary("patient", want)
	assertSummary("other", WorkSummary{})
	// Active grants, including expired grants not yet materialized as expired,
	// must never be counted as pending decisions.
	if _, err := f.consentService.Decide(ctx, f.tokens["patient"], grant.ID, "approve", 1); err != nil {
		t.Fatal(err)
	}
	want.Pending.Consents = 0
	assertSummary("patient", want)
	if _, err := f.pool.Exec(ctx, `UPDATE consents SET decided_at=now()-interval '2 hours',
		expires_at=now()-interval '1 hour' WHERE id=$1`, grant.ID); err != nil {
		t.Fatal(err)
	}
	assertSummary("patient", want)
	for _, actor := range []string{"doctor", "lab", "pharmacy", "admin"} {
		expectStatus(t, f.request(actor, "GET", path, "", false), 403)
	}
	expectStatus(t, f.request("", "GET", path, "", false), 401)
	expectStatus(t, f.request("patient", "GET", path+"?patient_id="+f.patientID, "", false), 400)
	if _, err := f.pool.Exec(ctx, `UPDATE sessions SET created_at=now()-interval '2 hours',
		expires_at=now()-interval '1 hour' WHERE user_id=$1`, f.ids["patient"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("patient", "GET", path, "", false), 401)
}
