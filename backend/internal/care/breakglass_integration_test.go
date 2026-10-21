package care

import (
	"context"
	"encoding/json"
	"github.com/mrizalbasri/Hygeia/backend/internal/breakglass"
	"strings"
	"testing"
	"time"
)

const emergencyInput = `{"patient_ref":"TEST-patient","reason_code":"unconscious","reason_text":"Synthetic unconscious patient"}`

func TestBreakGlassBoundedReadAndReview(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	r := newLabRequest(t, f, e)
	approveLab(t, f, r.ID)
	processLab(t, f, r.ID)
	result := submitLab(t, f, r.ID)
	rx := newPrescription(t, f, e)
	pending := newPrescription(t, f, e)
	approvePrescription(t, f, rx.ID)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/lab-results/"+result.ID, "", false), 403)
	for _, actor := range []string{"patient", "lab", "pharmacy", "admin"} {
		expectStatus(t, f.request(actor, "POST", "/api/v1/break-glass", emergencyInput, true), 403)
	}
	expectStatus(t, f.request("colleague", "POST", "/api/v1/break-glass", emergencyInput, false), 403)
	response := f.request("colleague", "POST", "/api/v1/break-glass", emergencyInput, true)
	expectStatus(t, response, 201)
	var body struct {
		Session breakglass.Session `json:"session"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	g := body.Session
	expectStatus(t, f.request("colleague", "GET", "/api/v1/prescriptions/"+rx.ID, "", false), 200)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/prescriptions/"+pending.ID, "", false), 403)
	if g.ExpiresAt.Sub(g.StartsAt) != time.Hour {
		t.Fatal("grant not bounded")
	}
	expectStatus(t, f.request("colleague", "POST", "/api/v1/break-glass", emergencyInput, true), 409)
	for _, actor := range []string{"patient", "colleague", "admin"} {
		path := "/api/v1/break-glass"
		if actor == "admin" {
			path = "/api/v1/admin/break-glass"
		}
		expectStatus(t, f.request(actor, "GET", path, "", false), 200)
	}
	resources := f.request("colleague", "GET", "/api/v1/break-glass/"+g.ID+"/resources", "", false)
	expectStatus(t, resources, 200)
	if !strings.Contains(resources.Body.String(), result.ID) || strings.Contains(resources.Body.String(), "SYNTHETIC-RESULT") {
		t.Fatal("resource directory incorrect")
	}
	expectStatus(t, f.request("doctor", "GET", "/api/v1/break-glass/"+g.ID+"/resources", "", false), 403)
	emergencyResult := f.request("colleague", "GET", "/api/v1/lab-results/"+result.ID, "", false)
	expectStatus(t, emergencyResult, 200)
	if strings.Contains(emergencyResult.Body.String(), "Synthetic test, not medical data") {
		t.Fatal("emergency access leaked clinical note")
	}
	if !strings.Contains(emergencyResult.Body.String(), "SYNTHETIC-RESULT-42") {
		t.Fatal("emergency scope omitted structured result")
	}
	note := `{"decision":"appropriate","note":"Synthetic emergency review"}`
	expectStatus(t, f.request("colleague", "PATCH", "/api/v1/admin/break-glass/"+g.ID, note, true), 403)
	expectStatus(t, f.request("admin", "PATCH", "/api/v1/admin/break-glass/"+g.ID, note, true), 200)
	expectStatus(t, f.request("admin", "PATCH", "/api/v1/admin/break-glass/"+g.ID, note, true), 409)
	var basis string
	if err := f.pool.QueryRow(ctx, `SELECT basis FROM audit_events WHERE action='lab_result.view' AND actor_id=$1 AND outcome='allow' ORDER BY seq DESC LIMIT 1`, f.ids["colleague"]).Scan(&basis); err != nil || basis != "break_glass" {
		t.Fatalf("missing audited basis %s %v", basis, err)
	}
	f.labService.Now = func() time.Time { return g.ExpiresAt }
	f.prescriptionService.Now = f.labService.Now
	f.breakGlassService.Now = f.labService.Now
	expectStatus(t, f.request("colleague", "GET", "/api/v1/lab-results/"+result.ID, "", false), 403)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/prescriptions/"+rx.ID, "", false), 403)
	expectStatus(t, f.request("colleague", "GET", "/api/v1/break-glass/"+g.ID+"/resources", "", false), 403)
	page := notificationPage(t, f, "patient", "/api/v1/notifications")
	found := false
	for _, n := range page.Notifications {
		if n.Action == "break_glass.start" {
			found = true
		}
	}
	if !found {
		t.Fatal("patient not notified")
	}
	v, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !v.Intact {
		t.Fatalf("audit invalid %v", err)
	}
}
func TestBreakGlassNotificationRollback(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `CREATE FUNCTION fail_emergency_notification() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic'; END $$; CREATE TRIGGER fail_emergency_notification BEFORE INSERT ON notifications FOR EACH ROW EXECUTE FUNCTION fail_emergency_notification()`); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("colleague", "POST", "/api/v1/break-glass", emergencyInput, true), 503)
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM break_glass_sessions`).Scan(&count); err != nil || count != 0 {
		t.Fatal("grant survived rollback")
	}
}
