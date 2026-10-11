package care

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/timeline"
)

func timelineResponse(t *testing.T, r *httptest.ResponseRecorder) timeline.Page {
	t.Helper()
	expectStatus(t, r, 200)
	var page timeline.Page
	if err := json.Unmarshal(r.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if page.Events == nil {
		t.Fatal("timeline events must be an array")
	}
	return page
}

func TestPatientTimelineJourneyIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	path := "/api/v1/patient/timeline"
	initial := timelineResponse(t, f.request("patient", "GET", path, "", false))
	if len(initial.Events) != 0 || initial.NextBeforeSeq != nil || initial.Limit != 20 {
		t.Fatal("incorrect empty timeline")
	}
	e := activeLabEpisode(t, f)
	expected := []string{"episode.invite", "episode.accept"}
	lab := newLabRequest(t, f, e)
	approveLab(t, f, lab.ID)
	processLab(t, f, lab.ID)
	result := submitLab(t, f, lab.ID)
	expected = append(expected, "lab_request.create", "lab_request.approve", "lab_request.accepted", "lab_request.in_progress", "lab_result.submit")
	p := newPrescription(t, f, e)
	approvePrescription(t, f, p.ID)
	for _, status := range []string{"accepted", "preparing", "ready", "completed"} {
		expectStatus(t, f.request("pharmacy", "PATCH", "/api/v1/prescriptions/"+p.ID+"/status", `{"status":"`+status+`"}`, true), 200)
	}
	expected = append(expected, "prescription.create", "prescription.approve", "prescription.accepted", "prescription.preparing", "prescription.ready", "prescription.completed")
	c := requestConsent(t, f, result.ID)
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/consents/"+c.ID, `{"action":"approve","duration_hours":24}`, true), 200)
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/consents/"+c.ID, `{"action":"revoke"}`, true), 200)
	expected = append(expected, "consent.request", "consent.approve", "consent.revoke")
	c = requestConsent(t, f, result.ID)
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/consents/"+c.ID, `{"action":"deny"}`, true), 200)
	expected = append(expected, "consent.request", "consent.deny")
	rejectedLab := newLabRequest(t, f, e)
	expectStatus(t, f.request("patient", "POST", "/api/v1/lab-requests/"+rejectedLab.ID+"/approval", `{"decision":"reject"}`, true), 200)
	rejectedPrescription := newPrescription(t, f, e)
	expectStatus(t, f.request("patient", "POST", "/api/v1/prescriptions/"+rejectedPrescription.ID+"/approval", `{"decision":"reject"}`, true), 200)
	expected = append(expected, "lab_request.create", "lab_request.reject", "prescription.create", "prescription.reject")
	newLabRequest(t, f, e)
	newPrescription(t, f, e)
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/episodes/"+e.ID, `{"action":"end"}`, true), 200)
	expected = append(expected, "lab_request.create", "prescription.create", "lab_request.cancel", "prescription.cancel", "episode.end")
	invite, err := f.service.Invite(ctx, f.tokens["doctor"], "TEST-patient")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.service.Transition(ctx, f.tokens["patient"], invite.ID, "reject"); err != nil {
		t.Fatal(err)
	}
	expected = append(expected, "episode.invite", "episode.reject")
	// Reads, denied attempts and failed state changes must stay in access history.
	expectStatus(t, f.request("patient", "GET", "/api/v1/lab-results/"+result.ID, "", false), 200)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/lab-results/"+result.ID, "", false), 403)
	expectStatus(t, f.request("patient", "GET", "/api/v1/prescriptions", "", false), 200)
	expectStatus(t, f.request("pharmacy", "PATCH", "/api/v1/prescriptions/"+p.ID+"/status", `{"status":"completed"}`, true), 409)
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/episodes/"+invite.ID, `{"action":"accept"}`, true), 409)
	// Names come from the historical event, never today's mutable user profile.
	if _, err := f.pool.Exec(ctx, `UPDATE users SET full_name='Renamed current doctor' WHERE id=$1`, f.ids["doctor"]); err != nil {
		t.Fatal(err)
	}
	r := f.request("patient", "GET", path+"?limit=100", "", false)
	page := timelineResponse(t, r)
	if len(page.Events) != len(expected) || page.NextBeforeSeq != nil {
		t.Fatalf("want %d milestones got %d", len(expected), len(page.Events))
	}
	for i, event := range page.Events {
		if event.Action != expected[len(expected)-1-i] {
			t.Fatalf("event %d: want %s got %s", i, expected[len(expected)-1-i], event.Action)
		}
		if event.Seq <= 0 || event.ResourceID == "" || event.ResourceType == "" || event.ActorName == "" || event.OccurredAt.IsZero() || event.Message == "" {
			t.Fatal("incomplete milestone")
		}
		if i > 0 && event.Seq >= page.Events[i-1].Seq {
			t.Fatal("timeline not in descending chain order")
		}
	}
	for _, secret := range []string{"SYNTHETIC-RESULT-42", "SYNTHETIC-MEDICATION-ONLY", "demo-dose", "Second opinion", "Renamed current doctor", "prev_hash", "actor_id", "patient_id"} {
		if strings.Contains(r.Body.String(), secret) {
			t.Fatalf("timeline exposed %s", secret)
		}
	}
	if !strings.Contains(r.Body.String(), "Synthetic doctor") || !strings.Contains(r.Body.String(), "menyelesaikan resep") {
		t.Fatal("missing historical name or message")
	}
	for _, kind := range []string{"episode", "laboratory", "prescription", "consent"} {
		filtered := timelineResponse(t, f.request("patient", "GET", path+"?kind="+kind+"&limit=100", "", false))
		want := make([]timeline.Event, 0)
		for _, event := range page.Events {
			if event.Kind == kind {
				want = append(want, event)
			}
		}
		if !reflect.DeepEqual(filtered.Events, want) || filtered.NextBeforeSeq != nil {
			t.Fatalf("incorrect %s filter", kind)
		}
	}
	var before, after int64
	if err := f.pool.QueryRow(ctx, `SELECT last_seq FROM audit_chain_heads WHERE chain_id=$1`, f.patientID).Scan(&before); err != nil {
		t.Fatal(err)
	}
	timelineResponse(t, f.request("patient", "GET", path, "", false))
	if err := f.pool.QueryRow(ctx, `SELECT last_seq FROM audit_chain_heads WHERE chain_id=$1`, f.patientID).Scan(&after); err != nil || after != before {
		t.Fatal("refresh grew audit chain")
	}
	verification, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verification.Intact {
		t.Fatal("timeline changed audit integrity")
	}
}

func TestPatientTimelineIsolationIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	newPrescription(t, f, e)
	path := "/api/v1/patient/timeline"
	expectStatus(t, f.request("", "GET", path, "", false), 401)
	for _, actor := range []string{"doctor", "colleague", "lab", "pharmacy", "admin"} {
		r := f.request(actor, "GET", path, "", false)
		expectStatus(t, r, 403)
		if strings.Contains(r.Body.String(), e.ID) {
			t.Fatal("staff denial leaked timeline")
		}
	}
	other := timelineResponse(t, f.request("other", "GET", path, "", false))
	if len(other.Events) != 0 {
		t.Fatal("other patient can see first patient's journey")
	}
	otherEpisode, err := f.service.Invite(ctx, f.tokens["doctor"], "TEST-other")
	if err != nil {
		t.Fatal(err)
	}
	other = timelineResponse(t, f.request("other", "GET", path, "", false))
	if len(other.Events) != 1 || other.Events[0].ResourceID != otherEpisode.ID {
		t.Fatal("other patient's own invitation missing")
	}
	for _, selector := range []string{"patient_id=" + f.patientID, "chain=" + f.patientID, "patient_ref=TEST-patient"} {
		expectStatus(t, f.request("other", "GET", path+"?"+selector, "", false), 400)
	}
	first := timelineResponse(t, f.request("patient", "GET", path, "", false))
	for _, event := range first.Events {
		if event.ResourceID == otherEpisode.ID {
			t.Fatal("patient sees another patient's milestone")
		}
	}
	if _, err := f.pool.Exec(ctx, `UPDATE sessions SET created_at=clock_timestamp()-interval '2 hours',expires_at=clock_timestamp()-interval '1 second' WHERE user_id=$1`, f.ids["patient"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("patient", "GET", path, "", false), 401)
}

func TestPatientTimelineCursorIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	for i := 0; i < 5; i++ {
		newPrescription(t, f, e)
	}
	path := "/api/v1/patient/timeline?kind=prescription&limit=2"
	original := timelineResponse(t, f.request("patient", "GET", "/api/v1/patient/timeline?kind=prescription&limit=100", "", false))
	page := timelineResponse(t, f.request("patient", "GET", path, "", false))
	if len(page.Events) != 2 || page.NextBeforeSeq == nil || *page.NextBeforeSeq != page.Events[1].Seq {
		t.Fatal("incorrect first cursor")
	}
	combined := append([]timeline.Event{}, page.Events...)
	// A concurrent new milestone must not shift continuation pages.
	newPrescription(t, f, e)
	for page.NextBeforeSeq != nil {
		page = timelineResponse(t, f.request("patient", "GET", fmt.Sprintf("%s&before_seq=%d", path, *page.NextBeforeSeq), "", false))
		combined = append(combined, page.Events...)
	}
	if !reflect.DeepEqual(combined, original.Events) {
		t.Fatal("cursor duplicated/skipped existing milestones after insertion")
	}
	empty := timelineResponse(t, f.request("patient", "GET", path+"&before_seq=1", "", false))
	if len(empty.Events) != 0 || empty.NextBeforeSeq != nil {
		t.Fatal("invalid exhausted page")
	}
	exact := timelineResponse(t, f.request("patient", "GET", "/api/v1/patient/timeline?kind=prescription&limit=6", "", false))
	if len(exact.Events) != 6 || exact.NextBeforeSeq != nil {
		t.Fatal("exact-sized final page must not emit cursor")
	}
	// Reading a page is metadata-only and does not require an audit write.
	if _, err := f.pool.Exec(ctx, `CREATE FUNCTION fail_timeline_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic audit unavailable'; END $$;
        CREATE TRIGGER fail_timeline_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION fail_timeline_audit()`); err != nil {
		t.Fatal(err)
	}
	failure := f.request("doctor", "POST", "/api/v1/prescriptions", `{"episode_id":"`+e.ID+`","items":[`+prescriptionItemJSON+`]}`, true)
	expectStatus(t, failure, 503)
	unchanged := timelineResponse(t, f.request("patient", "GET", "/api/v1/patient/timeline?kind=prescription&limit=100", "", false))
	if !reflect.DeepEqual(unchanged.Events, exact.Events) {
		t.Fatal("rolled-back prescription appeared in timeline")
	}
}

func TestPatientTimelineAllowlistAndFailureIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	p := newPrescription(t, f, e)
	path := "/api/v1/patient/timeline"
	before := timelineResponse(t, f.request("patient", "GET", path, "", false))
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	actor, err := (auth.Store{DB: tx}).UserBySession(ctx, f.tokens["doctor"])
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range []audit.Event{
		{Action: "future.business_action", ResourceType: "prescription", ResourceID: p.ID, Outcome: "allow"},
		{Action: "prescription.create", ResourceType: "lab_result", ResourceID: p.ID, Outcome: "allow"},
		{Action: "prescription.view", ResourceType: "prescription", ResourceID: p.ID, Outcome: "allow"},
		{Action: "episode.invite", ResourceType: "care_episode", ResourceID: e.ID, Outcome: "deny"},
	} {
		event.ChainID = f.patientID
		event.Basis = "synthetic_test"
		if err := f.service.Audit.Append(ctx, tx, actor, event); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	after := timelineResponse(t, f.request("patient", "GET", path, "", false))
	if !reflect.DeepEqual(before, after) {
		t.Fatal("unknown, mismatched, read, or denied event entered timeline")
	}
	if _, err := f.pool.Exec(ctx, `ALTER TABLE audit_events RENAME TO timeline_unavailable`); err != nil {
		t.Fatal(err)
	}
	r := f.request("patient", "GET", path, "", false)
	expectStatus(t, r, 503)
	if strings.Contains(r.Body.String(), p.ID) || strings.Contains(r.Body.String(), "Synthetic doctor") || strings.Contains(r.Body.String(), "audit_events") {
		t.Fatal("failed timeline query leaked event data or SQL")
	}
}
