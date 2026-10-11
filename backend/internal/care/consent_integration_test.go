package care

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mrizalbasri/Hygeia/backend/internal/consent"
	"github.com/mrizalbasri/Hygeia/backend/internal/laboratory"
)

func completedConsentResult(t *testing.T, f *fixture) (Episode, laboratory.Request, laboratory.Receipt) {
	t.Helper()
	e := activeLabEpisode(t, f)
	r := newLabRequest(t, f, e)
	approveLab(t, f, r.ID)
	processLab(t, f, r.ID)
	return e, r, submitLab(t, f, r.ID)
}

// A read that started before a revoke/expiry must recheck after its row lock wait.
func TestConsentReadAfterLockWaitIntegration(t *testing.T) {
	for _, scenario := range []string{"revoked", "expired"} {
		t.Run(scenario, func(t *testing.T) {
			f := integrationFixture(t)
			ctx := context.Background()
			_, _, r := completedConsentResult(t, f)
			var clock atomic.Int64
			clock.Store(time.Now().UTC().UnixMicro())
			f.consentService.Now = func() time.Time { return time.UnixMicro(clock.Load()) }
			f.labService.Now = f.consentService.Now
			c := requestConsent(t, f, r.ID)
			granted, err := f.consentService.Decide(ctx, f.tokens["patient"], c.ID, "approve", 1)
			if err != nil {
				t.Fatal(err)
			}
			tx, err := f.pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			// Hold the same row lock used by a patient decision, without committing yet.
			if _, err := tx.Exec(ctx, `SELECT id FROM consents WHERE id=$1 FOR UPDATE`, c.ID); err != nil {
				t.Fatal(err)
			}
			if scenario == "revoked" {
				if _, err := tx.Exec(ctx, `UPDATE consents SET status='revoked',revoked_at=clock_timestamp() WHERE id=$1`, c.ID); err != nil {
					t.Fatal(err)
				}
			}
			done := make(chan int, 1)
			go func() { done <- f.request("colleague", "GET", "/api/v1/lab-results/"+r.ID, "", false).Code }()
			deadline := time.Now().Add(2 * time.Second)
			waiting := false
			for time.Now().Before(deadline) {
				if err := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FOR SHARE OF c%')`).Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !waiting {
				t.Fatal("read never waited for consent lock")
			}
			if scenario == "expired" {
				clock.Store(granted.ExpiresAt.UnixMicro())
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case code := <-done:
				if code != 403 {
					t.Fatalf("read after %s returned %d", scenario, code)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("read stuck after lock release")
			}
		})
	}
}

func TestConsentAcrossClinicsAndEpisodeBasisIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	_, _, r := completedConsentResult(t, f)
	var org string
	if err := f.pool.QueryRow(ctx, `INSERT INTO organizations(type,name) VALUES('clinic','Second clinic') RETURNING id::text`).Scan(&org); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE organization_memberships SET organization_id=$2 WHERE user_id=$1`, f.ids["colleague"], org); err != nil {
		t.Fatal(err)
	}
	c := requestConsent(t, f, r.ID)
	if c.OrganizationID != org {
		t.Fatal("request not bound to second clinic")
	}
	path := "/api/v1/consents/" + c.ID
	resultPath := "/api/v1/lab-results/" + r.ID
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"approve","duration_hours":168}`, true), 200)
	expectStatus(t, f.request("colleague", "GET", resultPath, "", false), 200)
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"revoke"}`, true), 200)
	expectStatus(t, f.request("colleague", "GET", resultPath, "", false), 403)
	// Revoking C does not revoke the inviting doctor's independent active episode A.
	expectStatus(t, f.request("doctor", "GET", resultPath, "", false), 200)
	if _, err := f.pool.Exec(ctx, `UPDATE sessions SET created_at=now()-interval '2 hours',expires_at=now()-interval '1 second' WHERE user_id=$1`, f.ids["colleague"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("colleague", "GET", "/api/v1/consents", "", false), 401)
}
func consentBody(resultID string) string {
	b, _ := json.Marshal(consent.CreateInput{PatientRef: "TEST-patient", ResourceType: "lab_result", ResourceID: resultID, Purpose: "Second opinion using synthetic lab data"})
	return string(b)
}
func requestConsent(t *testing.T, f *fixture, resultID string) consent.Consent {
	t.Helper()
	r := f.request("colleague", "POST", "/api/v1/consents", consentBody(resultID), true)
	expectStatus(t, r, 201)
	if strings.Contains(r.Body.String(), "test_name") || strings.Contains(r.Body.String(), "SYNTHETIC-RESULT-42") {
		t.Fatal("consent request leaked clinical data")
	}
	var body struct {
		Consent consent.Consent `json:"consent"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Consent.Status != "pending" || body.Consent.GranteeID != f.ids["colleague"] || body.Consent.ResourceID != resultID {
		t.Fatal("invalid consent request")
	}
	return body.Consent
}
func TestConsentLifecycleIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e, request, result := completedConsentResult(t, f)
	resultPath := "/api/v1/lab-results/" + result.ID
	expectStatus(t, f.request("colleague", "GET", resultPath, "", false), 403)
	expectStatus(t, f.request("", "GET", "/api/v1/consents", "", false), 401)
	expectStatus(t, f.request("colleague", "POST", "/api/v1/consents", consentBody(result.ID), false), 403)
	expectStatus(t, f.request("colleague", "POST", "/api/v1/consents", strings.Replace(consentBody(result.ID), "lab_result", "all_results", 1), true), 400)
	expectStatus(t, f.request("colleague", "POST", "/api/v1/consents", strings.Replace(consentBody(result.ID), "TEST-patient", "TEST-other", 1), true), 403)
	c := requestConsent(t, f, result.ID)
	path := "/api/v1/consents/" + c.ID
	expectStatus(t, f.request("colleague", "POST", "/api/v1/consents", consentBody(result.ID), true), 409)
	patient := f.request("patient", "GET", path, "", false)
	expectStatus(t, patient, 200)
	if !strings.Contains(patient.Body.String(), "Synthetic lab test") || !strings.Contains(patient.Body.String(), "Second opinion") {
		t.Fatal("patient cannot inspect requested scope")
	}
	for _, actor := range []string{"other", "doctor", "lab", "labOther", "pharmacy", "admin"} {
		expectStatus(t, f.request(actor, "GET", path, "", false), 403)
		expectStatus(t, f.request(actor, "PATCH", path, `{"action":"approve","duration_hours":24}`, true), 403)
	}
	expectStatus(t, f.request("colleague", "PATCH", path, `{"action":"approve","duration_hours":24}`, true), 403)
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"approve","duration_hours":2}`, true), 400)
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"deny"}`, true), 200)
	expectStatus(t, f.request("colleague", "GET", resultPath, "", false), 403)
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"approve","duration_hours":24}`, true), 409)
	c = requestConsent(t, f, result.ID)
	path = "/api/v1/consents/" + c.ID
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"approve","duration_hours":24}`, true), 200)
	expectStatus(t, f.request("colleague", "GET", resultPath, "", false), 200)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/lab-requests/"+request.ID, "", false), 403)
	otherRequest := newLabRequest(t, f, e)
	approveLab(t, f, otherRequest.ID)
	processLab(t, f, otherRequest.ID)
	otherResult := submitLab(t, f, otherRequest.ID)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/lab-results/"+otherResult.ID, "", false), 403)
	for _, actor := range []string{"other", "lab", "pharmacy", "admin"} {
		expectStatus(t, f.request(actor, "GET", resultPath, "", false), 403)
	}
	for _, actor := range []string{"lab", "pharmacy", "admin"} {
		expectStatus(t, f.request(actor, "GET", "/api/v1/consents", "", false), 403)
	}
	expectStatus(t, f.request("patient", "GET", "/api/v1/consents?status=active&limit=1&offset=0", "", false), 200)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/consents?status=active", "", false), 200)
	expectStatus(t, f.request("patient", "GET", "/api/v1/consents?limit=101", "", false), 400)
	expectStatus(t, f.request("patient", "GET", "/api/v1/consents?status=invalid", "", false), 400)
	var newOrg string
	if err := f.pool.QueryRow(ctx, `INSERT INTO organizations(type,name) VALUES('clinic','Other clinic') RETURNING id::text`).Scan(&newOrg); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE organization_memberships SET organization_id=$2 WHERE user_id=$1`, f.ids["colleague"], newOrg); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("colleague", "GET", resultPath, "", false), 403)
	expectStatus(t, f.request("colleague", "GET", path, "", false), 403)
	if _, err := f.pool.Exec(ctx, `UPDATE organization_memberships SET organization_id=$2 WHERE user_id=$1`, f.ids["colleague"], f.ids["colleagueOrg"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/episodes/"+e.ID, `{"action":"end"}`, true), 200)
	expectStatus(t, f.request("doctor", "GET", resultPath, "", false), 403)
	expectStatus(t, f.request("colleague", "GET", resultPath, "", false), 200)
	summaries, err := f.service.AccessSummary(ctx, f.tokens["patient"], 100, 0)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, s := range summaries {
		if s.Action == "lab_result.view" && s.Basis == "consent" {
			found = true
			if s.ConsentID != c.ID || s.ConsentExpiresAt == "" {
				t.Fatal("missing consent audit scope")
			}
		}
	}
	if !found {
		t.Fatal("missing consent read audit")
	}
	var leaked bool
	if err := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_events WHERE context::text LIKE '%SYNTHETIC-RESULT-42%')`).Scan(&leaked); err != nil {
		t.Fatal(err)
	}
	if leaked {
		t.Fatal("clinical value leaked into audit")
	}
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"revoke"}`, true), 200)
	expectStatus(t, f.request("colleague", "GET", resultPath, "", false), 403)
	expectStatus(t, f.request("patient", "GET", resultPath, "", false), 200)
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"revoke"}`, true), 409)
	requestConsent(t, f, result.ID)
	verify, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatalf("consent damaged chain: %v", err)
	}
}

func TestConsentExpiryIntegration(t *testing.T) {
	f := integrationFixture(t)
	_, _, r := completedConsentResult(t, f)
	now := time.Now().UTC().Truncate(time.Microsecond)
	f.consentService.Now = func() time.Time { return now }
	f.labService.Now = f.consentService.Now
	for _, hours := range []int{1, 24, 168} {
		c := requestConsent(t, f, r.ID)
		path := "/api/v1/consents/" + c.ID
		granted, err := f.consentService.Decide(context.Background(), f.tokens["patient"], c.ID, "approve", hours)
		if err != nil || granted.ExpiresAt == nil || !granted.ExpiresAt.Equal(now.Add(time.Duration(hours)*time.Hour)) {
			t.Fatalf("duration %d: %v", hours, err)
		}
		now = granted.ExpiresAt.Add(-time.Microsecond)
		expectStatus(t, f.request("colleague", "GET", "/api/v1/lab-results/"+r.ID, "", false), 200)
		now = *granted.ExpiresAt
		expectStatus(t, f.request("colleague", "GET", "/api/v1/lab-results/"+r.ID, "", false), 403)
		expired := f.request("patient", "GET", path, "", false)
		expectStatus(t, expired, 200)
		if !strings.Contains(expired.Body.String(), `"status":"expired"`) {
			t.Fatal("expiry not derived on read")
		}
		listed := f.request("patient", "GET", "/api/v1/consents?status=expired", "", false)
		expectStatus(t, listed, 200)
		if !strings.Contains(listed.Body.String(), c.ID) {
			t.Fatal("expired filter missed derived state")
		}
		expectStatus(t, f.request("patient", "PATCH", path, `{"action":"revoke"}`, true), 409)
	}
}

func TestConsentConcurrentDecisionsIntegration(t *testing.T) {
	f := integrationFixture(t)
	_, _, r := completedConsentResult(t, f)
	c := requestConsent(t, f, r.ID)
	var wg sync.WaitGroup
	codes := make(chan int, 2)
	start := make(chan struct{})
	for _, body := range []string{`{"action":"approve","duration_hours":1}`, `{"action":"deny"}`} {
		wg.Add(1)
		go func(body string) {
			defer wg.Done()
			<-start
			codes <- f.request("patient", "PATCH", "/api/v1/consents/"+c.ID, body, true).Code
		}(body)
	}
	close(start)
	wg.Wait()
	close(codes)
	counts := map[int]int{}
	for code := range codes {
		counts[code]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatalf("concurrent decisions: %v", counts)
	}
	verify, err := f.service.Verify(context.Background(), f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatal("concurrent decisions damaged audit")
	}
}

func TestConsentAuditFailureIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	_, _, r := completedConsentResult(t, f)
	c := requestConsent(t, f, r.ID)
	fail := func() {
		t.Helper()
		if _, err := f.pool.Exec(ctx, `CREATE TRIGGER fail_consent_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION reject_audit_mutation()`); err != nil {
			t.Fatal(err)
		}
	}
	restore := func() {
		t.Helper()
		if _, err := f.pool.Exec(ctx, `DROP TRIGGER fail_consent_audit ON audit_events`); err != nil {
			t.Fatal(err)
		}
	}
	path := "/api/v1/consents/" + c.ID
	fail()
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"approve","duration_hours":1}`, true), 503)
	restore()
	got, err := f.consentService.Get(ctx, f.tokens["patient"], c.ID)
	if err != nil || got.Status != "pending" {
		t.Fatal("approval survived failed audit")
	}
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"approve","duration_hours":1}`, true), 200)
	fail()
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"revoke"}`, true), 503)
	hidden := f.request("colleague", "GET", "/api/v1/lab-results/"+r.ID, "", false)
	expectStatus(t, hidden, 503)
	if strings.Contains(hidden.Body.String(), "SYNTHETIC-RESULT-42") {
		t.Fatal("result exposed when audit unavailable")
	}
	restore()
	got, err = f.consentService.Get(ctx, f.tokens["patient"], c.ID)
	if err != nil || got.Status != "active" {
		t.Fatal("revocation survived failed audit")
	}
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"revoke"}`, true), 200)
	fail()
	expectStatus(t, f.request("colleague", "POST", "/api/v1/consents", consentBody(r.ID), true), 503)
	restore()
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM consents WHERE status='pending'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("request survived failed audit")
	}
	verify, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatal("failed consent transaction damaged chain")
	}
}
