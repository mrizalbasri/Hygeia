package care

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mrizalbasri/Hygeia/backend/internal/prescription"
)

const prescriptionItemJSON = `{"medication_name":"SYNTHETIC-MEDICATION-ONLY","dosage":"demo-dose","instructions":"Synthetic example, not medical advice","quantity":2}`

func newPrescription(t *testing.T, f *fixture, e Episode) prescription.Prescription {
	t.Helper()
	r := f.request("doctor", "POST", "/api/v1/prescriptions", `{"episode_id":"`+e.ID+`","items":[`+prescriptionItemJSON+`]}`, true)
	expectStatus(t, r, 201)
	p := prescriptionResponse(t, r)
	if r.Header().Get("Location") != "/api/v1/prescriptions/"+p.ID {
		t.Fatal("missing prescription location")
	}
	return p
}

func prescriptionResponse(t *testing.T, r *httptest.ResponseRecorder) prescription.Prescription {
	t.Helper()
	var body struct {
		Prescription prescription.Prescription `json:"prescription"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Prescription
}

func approvePrescription(t *testing.T, f *fixture, id string) {
	t.Helper()
	expectStatus(t, f.request("patient", "POST", "/api/v1/prescriptions/"+id+"/approval", `{"decision":"approve","pharmacy_org_id":"`+f.ids["pharmacyOrg"]+`"}`, true), 200)
}

func assertPharmacyProjection(t *testing.T, r *httptest.ResponseRecorder) {
	t.Helper()
	var body struct {
		Prescription map[string]any `json:"prescription"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Prescription) != 4 {
		t.Fatalf("extra pharmacy fields: %s", r.Body.String())
	}
	for _, key := range []string{"id", "patient_ref", "items", "status"} {
		if body.Prescription[key] == nil {
			t.Fatalf("missing %s", key)
		}
	}
}

func TestPrescriptionLifecycleIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	expectStatus(t, f.request("", "GET", "/api/v1/prescriptions", "", false), 401)
	directory := f.request("patient", "GET", "/api/v1/pharmacies?limit=1&offset=1", "", false)
	expectStatus(t, directory, 200)
	var providers struct {
		Pharmacies []map[string]any `json:"pharmacies"`
	}
	if err := json.Unmarshal(directory.Body.Bytes(), &providers); err != nil || len(providers.Pharmacies) != 1 || len(providers.Pharmacies[0]) != 2 {
		t.Fatal("incorrect pharmacy directory/projection")
	}
	for _, actor := range []string{"doctor", "pharmacy", "lab", "admin"} {
		expectStatus(t, f.request(actor, "GET", "/api/v1/pharmacies", "", false), 403)
	}
	e := activeLabEpisode(t, f)
	create := `{"episode_id":"` + e.ID + `","items":[` + prescriptionItemJSON + `]}`
	expectStatus(t, f.request("doctor", "POST", "/api/v1/prescriptions", create, false), 403)
	for _, actor := range []string{"patient", "other", "colleague", "lab", "pharmacy", "admin"} {
		expectStatus(t, f.request(actor, "POST", "/api/v1/prescriptions", create, true), 403)
	}
	p := newPrescription(t, f, e)
	if p.Status != "pending_patient_approval" || p.PharmacyOrgID != nil || p.ApprovedAt != nil || len(p.Items) != 1 || p.Items[0].Quantity != 2 {
		t.Fatal("invalid pending prescription")
	}
	path := "/api/v1/prescriptions/" + p.ID
	for _, actor := range []string{"other", "colleague", "lab", "pharmacy", "pharmacyOther", "admin"} {
		r := f.request(actor, "GET", path, "", false)
		expectStatus(t, r, 403)
		if strings.Contains(r.Body.String(), "SYNTHETIC") {
			t.Fatal("denied response leaked medication")
		}
		expectStatus(t, f.request(actor, "POST", path+"/approval", `{"decision":"reject"}`, true), 403)
	}
	for _, actor := range []string{"other", "colleague", "pharmacy", "pharmacyOther"} {
		r := f.request(actor, "GET", "/api/v1/prescriptions", "", false)
		expectStatus(t, r, 200)
		if !strings.Contains(r.Body.String(), `"prescriptions":[]`) {
			t.Fatal("pending or unrelated prescription leaked in list")
		}
	}
	for _, actor := range []string{"patient", "doctor"} {
		expectStatus(t, f.request(actor, "GET", path, "", false), 200)
	}
	expectStatus(t, f.request("patient", "POST", path+"/approval", `{"decision":"approve","pharmacy_org_id":"`+f.ids["labOrg"]+`"}`, true), 403)
	expectStatus(t, f.request("patient", "POST", path+"/approval", `{"decision":"approve","pharmacy_org_id":"00000000-0000-0000-0000-000000000001"}`, true), 403)
	approvePrescription(t, f, p.ID)
	expectStatus(t, f.request("patient", "POST", path+"/approval", `{"decision":"approve","pharmacy_org_id":"`+f.ids["pharmacyOtherOrg"]+`"}`, true), 409)
	expectStatus(t, f.request("patient", "POST", path+"/approval", `{"decision":"reject"}`, true), 409)
	for _, actor := range []string{"pharmacy", "pharmacyColleague"} {
		r := f.request(actor, "GET", path, "", false)
		expectStatus(t, r, 200)
		assertPharmacyProjection(t, r)
	}
	expectStatus(t, f.request("pharmacyOther", "GET", path, "", false), 403)
	for _, actor := range []string{"patient", "doctor", "colleague", "lab", "pharmacyOther", "admin"} {
		expectStatus(t, f.request(actor, "PATCH", path+"/status", `{"status":"accepted"}`, true), 403)
	}
	expectStatus(t, f.request("pharmacy", "PATCH", path+"/status", `{"status":"completed"}`, true), 409)
	queue := f.request("pharmacy", "GET", "/api/v1/prescriptions?status=sent", "", false)
	expectStatus(t, queue, 200)
	if !strings.Contains(queue.Body.String(), p.ID) || strings.Contains(queue.Body.String(), "episode_id") {
		t.Fatal("invalid queue projection")
	}
	for _, status := range []string{"accepted", "preparing", "ready", "completed"} {
		r := f.request("pharmacyColleague", "PATCH", path+"/status", `{"status":"`+status+`"}`, true)
		expectStatus(t, r, 200)
		assertPharmacyProjection(t, r)
		if prescriptionResponse(t, r).Status != status {
			t.Fatal("wrong next status")
		}
		expectStatus(t, f.request("pharmacy", "PATCH", path+"/status", `{"status":"`+status+`"}`, true), 409)
	}
	expectStatus(t, f.request("pharmacy", "GET", path, "", false), 403)
	queue = f.request("pharmacy", "GET", "/api/v1/prescriptions?status=completed", "", false)
	expectStatus(t, queue, 200)
	if !strings.Contains(queue.Body.String(), `"prescriptions":[]`) {
		t.Fatal("completed prescription still exposed to pharmacy")
	}
	finished := f.request("patient", "GET", path, "", false)
	expectStatus(t, finished, 200)
	final := prescriptionResponse(t, finished)
	if final.CompletedAt == nil || final.ApprovedAt == nil || final.PharmacyOrgID == nil || *final.PharmacyOrgID != f.ids["pharmacyOrg"] {
		t.Fatal("missing final metadata")
	}
	var leaked int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE context::text LIKE '%SYNTHETIC-MEDICATION%'`).Scan(&leaked); err != nil || leaked != 0 {
		t.Fatal("medication leaked into audit")
	}
	summary := f.request("patient", "GET", "/api/v1/patient/access-summary?limit=100", "", false)
	expectStatus(t, summary, 200)
	if !strings.Contains(summary.Body.String(), "menyelesaikan resep") || !strings.Contains(summary.Body.String(), "approved_pharmacy_assignment") {
		t.Fatal("missing prescription summary")
	}
	verify, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatal("prescription audit chain broken")
	}
	// Medication content is immutable even through direct database updates.
	if _, err := f.pool.Exec(ctx, `UPDATE prescription_items SET dosage='changed' WHERE prescription_id=$1`, p.ID); err == nil {
		t.Fatal("issued item update accepted")
	}
}

func TestPrescriptionRejectionAndEpisodeEndIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	pending := newPrescription(t, f, e)
	rejected := newPrescription(t, f, e)
	sent := newPrescription(t, f, e)
	approvePrescription(t, f, sent.ID)
	expectStatus(t, f.request("patient", "POST", "/api/v1/prescriptions/"+rejected.ID+"/approval", `{"decision":"reject"}`, true), 200)
	expectStatus(t, f.request("doctor", "PATCH", "/api/v1/episodes/"+e.ID, `{"action":"end"}`, true), 200)
	for _, tc := range []struct{ id, reason string }{{pending.ID, "episode_ended"}, {rejected.ID, "patient_rejected"}} {
		r := f.request("patient", "GET", "/api/v1/prescriptions/"+tc.id, "", false)
		expectStatus(t, r, 200)
		p := prescriptionResponse(t, r)
		if p.Status != "cancelled" || p.CancelledAt == nil || p.PharmacyOrgID != nil || p.CancellationReason == nil || *p.CancellationReason != tc.reason {
			t.Fatal("invalid cancellation")
		}
		expectStatus(t, f.request("patient", "POST", "/api/v1/prescriptions/"+tc.id+"/approval", `{"decision":"approve","pharmacy_org_id":"`+f.ids["pharmacyOrg"]+`"}`, true), 409)
	}
	for _, p := range []prescription.Prescription{pending, rejected, sent} {
		expectStatus(t, f.request("doctor", "GET", "/api/v1/prescriptions/"+p.ID, "", false), 403)
	}
	r := f.request("doctor", "GET", "/api/v1/prescriptions", "", false)
	expectStatus(t, r, 200)
	if !strings.Contains(r.Body.String(), `"prescriptions":[]`) {
		t.Fatal("doctor list retained ended episode prescriptions")
	}
	expectStatus(t, f.request("doctor", "POST", "/api/v1/prescriptions", `{"episode_id":"`+e.ID+`","items":[`+prescriptionItemJSON+`]}`, true), 403)
	for _, status := range []string{"accepted", "preparing", "ready", "completed"} {
		expectStatus(t, f.request("pharmacy", "PATCH", "/api/v1/prescriptions/"+sent.ID+"/status", `{"status":"`+status+`"}`, true), 200)
	}
	verify, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatal("audit after episode end broken")
	}
}

func TestPrescriptionConcurrencyIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		e := activeLabEpisode(t, f)
		p := newPrescription(t, f, e)
		start := make(chan struct{})
		approval, end := make(chan error, 1), make(chan error, 1)
		go func() {
			<-start
			_, err := f.prescriptionService.Approve(ctx, f.tokens["patient"], p.ID, prescription.ApprovalInput{Decision: "approve", PharmacyOrgID: f.ids["pharmacyOrg"]})
			approval <- err
		}()
		go func() { <-start; _, err := f.service.Transition(ctx, f.tokens["patient"], e.ID, "end"); end <- err }()
		close(start)
		decision, ending := <-approval, <-end
		if ending != nil {
			t.Fatal(ending)
		}
		if decision != nil && !errors.Is(decision, prescription.ErrConflict) {
			t.Fatal(decision)
		}
		var status string
		var approved *time.Time
		if err := f.pool.QueryRow(ctx, `SELECT status,approved_at FROM prescriptions WHERE id=$1`, p.ID).Scan(&status, &approved); err != nil {
			t.Fatal(err)
		}
		if decision == nil && (status != "sent" || approved == nil) || decision != nil && (status != "cancelled" || approved != nil) {
			t.Fatal("approval/end produced contradictory state")
		}
	}
	e := activeLabEpisode(t, f)
	p := newPrescription(t, f, e)
	approvePrescription(t, f, p.ID)
	start := make(chan struct{})
	errorsCh := make(chan error, 2)
	var group sync.WaitGroup
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			_, err := f.prescriptionService.UpdateStatus(ctx, f.tokens["pharmacy"], p.ID, "accepted")
			errorsCh <- err
		}()
	}
	close(start)
	group.Wait()
	close(errorsCh)
	successes, conflicts := 0, 0
	for err := range errorsCh {
		if err == nil {
			successes++
		} else if errors.Is(err, prescription.ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatal("duplicate transition succeeded")
	}
	verify, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatal("concurrent audit broken")
	}
}

func TestPrescriptionAuditFailureIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	pending := newPrescription(t, f, e)
	sent := newPrescription(t, f, e)
	approvePrescription(t, f, sent.ID)
	var originalCount int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM prescriptions`).Scan(&originalCount); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `CREATE FUNCTION fail_prescription_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic audit failure'; END $$;
        CREATE TRIGGER fail_prescription_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION fail_prescription_audit()`); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ actor, method, path, body string }{
		{"doctor", "POST", "/api/v1/prescriptions", `{"episode_id":"` + e.ID + `","items":[` + prescriptionItemJSON + `]}`},
		{"patient", "POST", "/api/v1/prescriptions/" + pending.ID + "/approval", `{"decision":"approve","pharmacy_org_id":"` + f.ids["pharmacyOrg"] + `"}`},
		{"patient", "POST", "/api/v1/prescriptions/" + pending.ID + "/approval", `{"decision":"reject"}`},
		{"pharmacy", "PATCH", "/api/v1/prescriptions/" + sent.ID + "/status", `{"status":"accepted"}`},
		{"doctor", "PATCH", "/api/v1/episodes/" + e.ID, `{"action":"end"}`},
		{"patient", "GET", "/api/v1/prescriptions/" + sent.ID, ""},
		{"pharmacy", "GET", "/api/v1/prescriptions", ""},
		{"other", "GET", "/api/v1/prescriptions/" + sent.ID, ""},
	} {
		r := f.request(tc.actor, tc.method, tc.path, tc.body, true)
		expectStatus(t, r, 503)
		if strings.Contains(r.Body.String(), "SYNTHETIC") || strings.Contains(r.Body.String(), "prescription\"") {
			t.Fatal("audit failure leaked prescription")
		}
	}
	var count, itemCount int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM prescriptions`).Scan(&count); err != nil || count != originalCount {
		t.Fatal("create survived audit failure")
	}
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM prescription_items`).Scan(&itemCount); err != nil || itemCount != originalCount {
		t.Fatal("items survived failed create")
	}
	var episodeStatus, pendingStatus, sentStatus string
	if err := f.pool.QueryRow(ctx, `SELECT status FROM care_episodes WHERE id=$1`, e.ID).Scan(&episodeStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT status FROM prescriptions WHERE id=$1`, pending.ID).Scan(&pendingStatus); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT status FROM prescriptions WHERE id=$1`, sent.ID).Scan(&sentStatus); err != nil {
		t.Fatal(err)
	}
	if episodeStatus != "active" || pendingStatus != "pending_patient_approval" || sentStatus != "sent" {
		t.Fatal("mutation survived audit failure")
	}
	if _, err := f.pool.Exec(ctx, `DROP TRIGGER fail_prescription_audit ON audit_events; DROP FUNCTION fail_prescription_audit()`); err != nil {
		t.Fatal(err)
	}
	verify, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verify.Intact {
		t.Fatal("audit damaged by rollback")
	}
}

// Reads that were already waiting must re-evaluate access after the writer
// commits, rather than returning the clinical snapshot selected before the wait.
func TestPrescriptionReadAfterLockWaitIntegration(t *testing.T) {
	for _, scenario := range []string{"doctor_list", "doctor_detail", "pharmacy_list"} {
		t.Run(scenario, func(t *testing.T) {
			f := integrationFixture(t)
			ctx := context.Background()
			e := activeLabEpisode(t, f)
			p := newPrescription(t, f, e)
			approvePrescription(t, f, p.ID)
			if scenario == "pharmacy_list" {
				for _, status := range []string{"accepted", "preparing", "ready"} {
					if _, err := f.prescriptionService.UpdateStatus(ctx, f.tokens["pharmacy"], p.ID, status); err != nil {
						t.Fatal(err)
					}
				}
			}
			tx, err := f.pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback(ctx)
			if _, err := tx.Exec(ctx, `SELECT id FROM care_episodes WHERE id=$1 FOR UPDATE`, e.ID); err != nil {
				t.Fatal(err)
			}
			actor, path := "doctor", "/api/v1/prescriptions"
			if scenario == "doctor_detail" {
				path += "/" + p.ID
			}
			if scenario == "pharmacy_list" {
				actor = "pharmacy"
			}
			done := make(chan *httptest.ResponseRecorder, 1)
			go func() { done <- f.request(actor, "GET", path, "", false) }()
			deadline := time.Now().Add(2 * time.Second)
			waiting := false
			for time.Now().Before(deadline) {
				if err := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FOR SHARE OF e%')`).Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !waiting {
				t.Fatal("prescription read never waited for episode lock")
			}
			if scenario == "pharmacy_list" {
				_, err = tx.Exec(ctx, `UPDATE prescriptions SET status='completed',completed_at=clock_timestamp() WHERE id=$1`, p.ID)
			} else {
				_, err = tx.Exec(ctx, `UPDATE care_episodes SET status='ended',closed_at=clock_timestamp() WHERE id=$1`, e.ID)
			}
			if err != nil {
				t.Fatal(err)
			}
			if err := tx.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case r := <-done:
				if scenario == "doctor_detail" {
					expectStatus(t, r, 403)
				} else {
					expectStatus(t, r, 200)
					if !strings.Contains(r.Body.String(), `"prescriptions":[]`) {
						t.Fatal("list exposed stale prescription")
					}
				}
				if strings.Contains(r.Body.String(), "SYNTHETIC") {
					t.Fatal("read exposed revoked clinical data")
				}
			case <-time.After(3 * time.Second):
				t.Fatal("prescription read stuck after lock release")
			}
		})
	}
}

func TestPrescriptionOrderedItemsIntegration(t *testing.T) {
	f := integrationFixture(t)
	e := activeLabEpisode(t, f)
	items := make([]prescription.Item, 20)
	for i := range items {
		items[i] = prescription.Item{MedicationName: strings.Repeat("x", i+1), Dosage: " demo ", Instructions: " synthetic ", Quantity: i + 1}
	}
	body, err := json.Marshal(prescription.CreateInput{EpisodeID: e.ID, Items: items})
	if err != nil {
		t.Fatal(err)
	}
	r := f.request("doctor", "POST", "/api/v1/prescriptions", string(body), true)
	expectStatus(t, r, 201)
	p := prescriptionResponse(t, r)
	if len(p.Items) != 20 {
		t.Fatal("missing prescription items")
	}
	approvePrescription(t, f, p.ID)
	r = f.request("pharmacy", "GET", "/api/v1/prescriptions/"+p.ID, "", false)
	expectStatus(t, r, 200)
	assertPharmacyProjection(t, r)
	p = prescriptionResponse(t, r)
	for i, item := range p.Items {
		if item.MedicationName != items[i].MedicationName || item.Quantity != i+1 || item.Dosage != "demo" || item.Instructions != "synthetic" {
			t.Fatal("items lost order or text normalization")
		}
	}
}
