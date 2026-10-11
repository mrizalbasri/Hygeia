package care

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/mrizalbasri/Hygeia/backend/internal/notification"
)

func notificationPage(t *testing.T, f *fixture, actor, path string) notification.Page {
	t.Helper()
	r := f.request(actor, "GET", path, "", false)
	expectStatus(t, r, 200)
	var page notification.Page
	if err := json.Unmarshal(r.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	return page
}
func TestNotificationsIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	path := "/api/v1/notifications"
	expectStatus(t, f.request("", "GET", path, "", false), 401)
	e := activeLabEpisode(t, f)
	p := newPrescription(t, f, e)
	patient := notificationPage(t, f, "patient", path)
	if len(patient.Notifications) != 2 {
		t.Fatalf("patient notifications: %+v", patient)
	}
	if patient.Notifications[0].Action != "prescription.create" {
		t.Fatal("missing prescription approval request")
	}
	if len(notificationPage(t, f, "pharmacy", path).Notifications) != 0 {
		t.Fatal("pharmacy notified before patient approval")
	}
	approvePrescription(t, f, p.ID)
	pharmacy := notificationPage(t, f, "pharmacy", path)
	if len(pharmacy.Notifications) != 1 || pharmacy.Notifications[0].ResourceID != p.ID {
		t.Fatal("assignment notification missing")
	}
	if len(notificationPage(t, f, "pharmacyOther", path).Notifications) != 0 || len(notificationPage(t, f, "other", path).Notifications) != 0 {
		t.Fatal("notification crossed user/org boundary")
	}
	id := patient.Notifications[0].ID
	expectStatus(t, f.request("other", "PATCH", fmt.Sprintf("%s/%d", path, id), `{"read":true}`, true), 403)
	expectStatus(t, f.request("patient", "PATCH", fmt.Sprintf("%s/%d", path, id), `{"read":true}`, false), 403)
	expectStatus(t, f.request("patient", "PATCH", fmt.Sprintf("%s/%d", path, id), `{"read":true}`, true), 200)
	first := notificationPage(t, f, "patient", path)
	expectStatus(t, f.request("patient", "PATCH", fmt.Sprintf("%s/%d", path, id), `{"read":true}`, true), 200)
	second := notificationPage(t, f, "patient", path)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("mark read not idempotent")
	}
	unread := notificationPage(t, f, "patient", path+"?unread=true")
	if len(unread.Notifications) != 1 {
		t.Fatal("wrong unread filter")
	}
	expectStatus(t, f.request("patient", "GET", path+"/unread-count", "", false), 200)
	page := notificationPage(t, f, "patient", path+"?limit=1")
	if page.NextBeforeID == nil {
		t.Fatal("cursor missing")
	}
	tail := notificationPage(t, f, "patient", fmt.Sprintf("%s?limit=1&before_id=%d", path, *page.NextBeforeID))
	if len(tail.Notifications) != 1 || tail.NextBeforeID != nil || tail.Notifications[0].ID == page.Notifications[0].ID {
		t.Fatal("bad pagination")
	}
	var before, after int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("patient", "GET", "/api/v1/prescriptions/"+p.ID, "", false), 200)
	expectStatus(t, f.request("other", "GET", "/api/v1/prescriptions/"+p.ID, "", false), 403)
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications`).Scan(&after); err != nil || before != after {
		t.Fatal("read or deny created notification")
	}
	raw := f.request("patient", "GET", path, "", false).Body.String()
	if strings.Contains(raw, "SYNTHETIC-MEDICATION") || strings.Contains(raw, "dosage") || strings.Contains(raw, "source_chain") {
		t.Fatal("notification leaked clinical data")
	}
	for _, query := range []string{"user_id=other", "limit=101", "before_id=0", "before_id=9223372036854775808", "unread=1", "limit=1&limit=2"} {
		expectStatus(t, f.request("patient", "GET", path+"?"+query, "", false), 400)
	}
}
func TestNotificationRollbackIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	e := activeLabEpisode(t, f)
	var events int64
	if err := f.pool.QueryRow(ctx, `SELECT last_seq FROM audit_chain_heads WHERE chain_id=$1`, f.patientID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `CREATE FUNCTION fail_notification() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'synthetic notification failure'; END $$; CREATE TRIGGER fail_notification BEFORE INSERT ON notifications FOR EACH ROW EXECUTE FUNCTION fail_notification()`); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("doctor", "POST", "/api/v1/prescriptions", `{"episode_id":"`+e.ID+`","items":[`+prescriptionItemJSON+`]}`, true), 503)
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM prescriptions`).Scan(&count); err != nil || count != 0 {
		t.Fatal("prescription survived notification failure")
	}
	var after int64
	if err := f.pool.QueryRow(ctx, `SELECT last_seq FROM audit_chain_heads WHERE chain_id=$1`, f.patientID).Scan(&after); err != nil || after != events {
		t.Fatal("audit survived notification failure")
	}
}
