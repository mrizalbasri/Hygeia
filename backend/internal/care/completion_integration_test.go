package care

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/laboratory"
)

func TestAuditRotationMixedChainIntegration(t *testing.T) {
	f := integrationFixture(t)
	e := activeLabEpisode(t, f)
	ctx := context.Background()
	actor, err := (auth.Store{DB: f.pool}).UserBySession(ctx, f.tokens["doctor"])
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := audit.NewWriter(bytes.Repeat([]byte{9}, 32), bytes.Repeat([]byte{3}, 32))
	if err != nil {
		t.Fatal(err)
	}
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err := rotated.Append(ctx, tx, actor, audit.Event{ChainID: f.patientID, Action: "episode.view", ResourceType: "care_episode", ResourceID: e.ID, Outcome: "allow", Basis: "episode"}); err != nil {
		t.Fatal(err)
	}
	verification, err := rotated.Verify(ctx, tx, f.patientID)
	if err != nil || !verification.Intact || verification.Events < 3 {
		t.Fatalf("mixed chain verification: %+v %v", verification, err)
	}
	withoutHistory, _ := audit.NewWriter(bytes.Repeat([]byte{9}, 32))
	verification, err = withoutHistory.Verify(ctx, tx, f.patientID)
	if err != nil || verification.Intact {
		t.Fatalf("missing historical key did not detect mismatch: %+v %v", verification, err)
	}
}

const registrationJSON = `{"email":" NEW@HYGEIA.TEST ","password":"Synthetic-password-2026","full_name":"Synthetic Patient","birth_year":2000,"sex":"unknown"}`

func TestPatientRegistrationIntegration(t *testing.T) {
	f := integrationFixture(t)
	expectStatus(t, f.request("", "POST", "/api/v1/auth/register", registrationJSON, false), 403)
	r := f.request("", "POST", "/api/v1/auth/register", registrationJSON, true)
	expectStatus(t, r, 201)
	var body struct {
		User auth.User `json:"user"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.User.Role != "patient" || body.User.Email != "new@hygeia.test" || body.User.PatientRef == nil {
		t.Fatal("invalid registered identity")
	}
	var session *http.Cookie
	for _, c := range r.Result().Cookies() {
		if c.Name == "hygeia_session" {
			session = c
		}
	}
	if session == nil || !session.HttpOnly || session.MaxAge <= 0 {
		t.Fatal("missing authenticated cookie")
	}
	req := httptest.NewRequest("GET", "http://localhost/api/v1/me", nil)
	req.AddCookie(session)
	me := httptest.NewRecorder()
	f.api.ServeHTTP(me, req)
	expectStatus(t, me, 200)
	expectStatus(t, f.request("", "POST", "/api/v1/auth/register", registrationJSON, true), 409)
	var count int
	if err := f.pool.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE email='new@hygeia.test'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("duplicate registration: count=%d err=%v", count, err)
	}
}

func TestPatientRegistrationSessionFailureRollsBackIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	_, err := f.pool.Exec(ctx, `CREATE FUNCTION fail_session() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test session failure'; END; $$;
	CREATE TRIGGER fail_session BEFORE INSERT ON sessions FOR EACH ROW EXECUTE FUNCTION fail_session()`)
	if err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("", "POST", "/api/v1/auth/register", registrationJSON, true), 503)
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email='new@hygeia.test'`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("registration not rolled back: count=%d err=%v", count, err)
	}
}

func TestSessionCleanupIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	expired := sha256.Sum256([]byte("expired-session"))
	if _, err := f.pool.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,created_at,expires_at) VALUES($1,$2,now()-interval '2 hours',now()-interval '1 hour')`, expired[:], f.ids["patient"]); err != nil {
		t.Fatal(err)
	}
	store := auth.Store{DB: f.pool}
	if n, err := store.DeleteExpired(ctx); err != nil || n != 1 {
		t.Fatalf("cleanup: count=%d err=%v", n, err)
	}
	if _, err := store.UserBySession(ctx, f.tokens["patient"]); err != nil {
		t.Fatal("cleanup removed active session", err)
	}
}

func TestLabCorrectionsIntegration(t *testing.T) {
	f := integrationFixture(t)
	e := activeLabEpisode(t, f)
	request := newLabRequest(t, f, e)
	approveLab(t, f, request.ID)
	processLab(t, f, request.ID)
	result := submitLab(t, f, request.ID)
	path := "/api/v1/lab-results/" + result.ID
	payload := `{"reason":"Synthetic instrument correction","value":"CORRECTED-DEMO","unit":"demo","reference_range":"demo-only","flag":"normal","note":"Synthetic amendment"}`
	expectStatus(t, f.request("lab", "POST", path+"/corrections", payload, false), 403)
	for _, actor := range []string{"patient", "doctor", "labOther", "admin"} {
		expectStatus(t, f.request(actor, "POST", path+"/corrections", payload, true), 403)
	}
	expectStatus(t, f.request("lab", "POST", path+"/corrections", payload, true), 201)
	expectStatus(t, f.request("labColleague", "POST", path+"/corrections", payload, true), 201)
	r := f.request("patient", "GET", path, "", false)
	expectStatus(t, r, 200)
	var body struct {
		Result laboratory.Result `json:"result"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Result.Value != "SYNTHETIC-RESULT-42" || len(body.Result.Corrections) != 2 || body.Result.Corrections[0].Value != "CORRECTED-DEMO" {
		t.Fatal("original or corrections lost")
	}
	for _, sql := range []string{`UPDATE lab_result_corrections SET value='changed'`, `DELETE FROM lab_result_corrections`, `TRUNCATE lab_result_corrections`} {
		if _, err := f.pool.Exec(context.Background(), sql); err == nil {
			t.Fatal("correction mutation allowed", sql)
		}
	}
	expectStatus(t, f.request("patient", "GET", "/api/v1/audit/verify", "", false), 200)
	var leaked bool
	if err := f.pool.QueryRow(context.Background(), `SELECT EXISTS(SELECT 1 FROM audit_events WHERE context::text LIKE '%CORRECTED-DEMO%')`).Scan(&leaked); err != nil || leaked {
		t.Fatalf("audit payload leak: %v", err)
	}
}
