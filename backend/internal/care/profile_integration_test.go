package care

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/mrizalbasri/Hygeia/backend/db"
	"github.com/mrizalbasri/Hygeia/backend/internal/patientprofile"
)

const profileJSON = `{"full_name":"  Pasien Profil Sintetis  ","birth_date":"2000-02-29","sex":"female","phone":"+62 812-3456-7890","address":" Jalan Sintetis 1 ","emergency_contact_name":"Kontak Sintetis","emergency_contact_phone":"08123456780"}`

func getProfile(t *testing.T, f *fixture, actor string) patientprofile.Profile {
	t.Helper()
	r := f.request(actor, "GET", "/api/v1/me/profile", "", false)
	expectStatus(t, r, 200)
	var body struct {
		Profile patientprofile.Profile `json:"profile"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Profile
}

func TestPatientProfileIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `UPDATE patients SET birth_year=1998 WHERE user_id=$1`, f.ids["patient"]); err != nil {
		t.Fatal(err)
	}
	initial := getProfile(t, f, "patient")
	if initial.BirthDate != nil || initial.BirthYear == nil || *initial.BirthYear != 1998 ||
		initial.PatientRef != "TEST-patient" || initial.Sex != "unknown" {
		t.Fatalf("legacy profile changed: %+v", initial)
	}
	other := getProfile(t, f, "other")
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/me/profile", profileJSON, true), 200)
	saved := getProfile(t, f, "patient")
	if saved.FullName != "Pasien Profil Sintetis" || saved.BirthDate == nil || *saved.BirthDate != "2000-02-29" ||
		saved.BirthYear == nil || *saved.BirthYear != 2000 || saved.Address != "Jalan Sintetis 1" ||
		saved.Phone != "+62 812-3456-7890" || saved.EmergencyContactPhone != "08123456780" ||
		saved.Email != initial.Email || saved.PatientRef != initial.PatientRef {
		t.Fatalf("profile not persisted: %+v", saved)
	}
	if got := getProfile(t, f, "other"); got != other {
		t.Fatal("update changed another patient")
	}
	me := f.request("patient", "GET", "/api/v1/me", "", false)
	expectStatus(t, me, 200)
	if !strings.Contains(me.Body.String(), `"full_name":"Pasien Profil Sintetis"`) {
		t.Fatal("identity did not use updated name")
	}
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE chain_id=$1 AND action='patient.profile_update' AND basis='patient_self'`, f.patientID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("missing profile audit: count=%d err=%v", count, err)
	}
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	verification, err := f.service.Audit.Verify(ctx, tx, f.patientID)
	if err != nil || !verification.Intact {
		t.Fatalf("audit chain invalid: %+v %v", verification, err)
	}
	_ = tx.Rollback(ctx)

	// Removing optional fields must not invent or erase the known birth year.
	clear := `{"full_name":"Pasien Sintetis","birth_date":"","sex":"unknown","phone":"","address":"","emergency_contact_name":"","emergency_contact_phone":""}`
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/me/profile", clear, true), 200)
	cleared := getProfile(t, f, "patient")
	if cleared.BirthDate != nil || cleared.BirthYear == nil || *cleared.BirthYear != 2000 ||
		cleared.Phone != "" || cleared.Address != "" || cleared.EmergencyContactName != "" || cleared.EmergencyContactPhone != "" {
		t.Fatalf("optional fields not cleared: %+v", cleared)
	}
}

func TestPatientProfileAccessIntegration(t *testing.T) {
	f := integrationFixture(t)
	path := "/api/v1/me/profile"
	expectStatus(t, f.request("", "GET", path, "", false), 401)
	expectStatus(t, f.request("", "PATCH", path, profileJSON, true), 401)
	expectStatus(t, f.request("patient", "PATCH", path, profileJSON, false), 403)
	for _, actor := range []string{"doctor", "lab", "pharmacy", "admin"} {
		t.Run(actor, func(t *testing.T) {
			expectStatus(t, f.request(actor, "GET", path, "", false), 403)
			expectStatus(t, f.request(actor, "PATCH", path, profileJSON, true), 403)
		})
	}
	for _, query := range []string{"?patient_ref=TEST-other", "?user_id=" + f.ids["other"]} {
		expectStatus(t, f.request("patient", "GET", path+query, "", false), 400)
		expectStatus(t, f.request("patient", "PATCH", path+query, profileJSON, true), 400)
	}
	for _, field := range []string{"id", "user_id", "patient_id", "patient_ref", "email", "role", "birth_year", "disabled_at"} {
		payload := strings.TrimSuffix(profileJSON, "}") + `,"` + field + `":"spoof"}`
		expectStatus(t, f.request("patient", "PATCH", path, payload, true), 400)
	}
	for _, payload := range []string{
		`{}`, `null`, profileJSON + `{}`, strings.Replace(profileJSON, "2000-02-29", "2999-01-01", 1),
		strings.Replace(profileJSON, "2000-02-29", "2001-02-29", 1),
		strings.Replace(profileJSON, "+62 812-3456-7890", "not a phone", 1),
		strings.Replace(profileJSON, "Kontak Sintetis", "", 1),
	} {
		expectStatus(t, f.request("patient", "PATCH", path, payload, true), 400)
	}
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `UPDATE users SET disabled_at=now() WHERE id=$1`, f.ids["patient"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("patient", "GET", path, "", false), 401)
	expectStatus(t, f.request("patient", "PATCH", path, profileJSON, true), 401)
	if _, err := f.pool.Exec(ctx, `UPDATE sessions SET expires_at=now()-interval '1 second',created_at=now()-interval '1 hour' WHERE user_id=$1`, f.ids["other"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("other", "GET", path, "", false), 401)
	expectStatus(t, f.request("other", "PATCH", path, profileJSON, true), 401)
}

func TestPatientProfileAuditFailureRollsBackIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	initial := getProfile(t, f, "patient")
	_, err := f.pool.Exec(ctx, `CREATE FUNCTION fail_profile_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test audit failure'; END; $$;
		CREATE TRIGGER fail_profile_audit BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION fail_profile_audit()`)
	if err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/me/profile", profileJSON, true), 503)
	if got := getProfile(t, f, "patient"); got != initial {
		t.Fatalf("audit failure did not roll back: %+v", got)
	}
}

func TestPatientProfileMigrationIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `UPDATE patients SET birth_year=1998 WHERE user_id=$1`, f.ids["patient"]); err != nil {
		t.Fatal(err)
	}
	for _, direction := range []string{"down", "up"} {
		sql, err := db.Migrations.ReadFile("migrations/000011_patient_profile." + direction + ".sql")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.pool.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	profile := getProfile(t, f, "patient")
	if profile.BirthDate != nil || profile.BirthYear == nil || *profile.BirthYear != 1998 {
		t.Fatalf("migration fabricated a birth date: %+v", profile)
	}
	// SQL-shaped input remains data, not executable SQL.
	payload := strings.Replace(profileJSON, "  Pasien Profil Sintetis  ", "Robert'); DROP TABLE users;--", 1)
	expectStatus(t, f.request("patient", "PATCH", "/api/v1/me/profile", payload, true), 200)
	if got := getProfile(t, f, "patient"); got.FullName != "Robert'); DROP TABLE users;--" {
		t.Fatal("parameterized value was not preserved")
	}
}

func TestPatientProfileDisabledWhileWaitingIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	var holder int
	if err := tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&holder); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET disabled_at=now() WHERE id=$1`, f.ids["patient"]); err != nil {
		t.Fatal(err)
	}
	done := make(chan int, 1)
	go func() {
		done <- f.request("patient", "PATCH", "/api/v1/me/profile", profileJSON, true).Code
	}()
	deadline := time.Now().Add(3 * time.Second)
	for {
		var waiting bool
		if err := f.pool.QueryRow(ctx, `SELECT EXISTS(
			SELECT 1 FROM pg_stat_activity WHERE wait_event_type='Lock'
			AND $1=ANY(pg_blocking_pids(pid)) AND query LIKE '%SELECT id FROM users WHERE id=%'
		)`, holder).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("profile update never waited for the account lock")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if status := <-done; status != 401 {
		t.Fatalf("disabled account updated profile after lock wait: %d", status)
	}
	var name string
	if err := f.pool.QueryRow(ctx, `SELECT full_name FROM users WHERE id=$1`, f.ids["patient"]).Scan(&name); err != nil || name != "Synthetic patient" {
		t.Fatalf("disabled account changed name: %s %v", name, err)
	}
}
