package care

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mrizalbasri/Hygeia/backend/db"
	adminapi "github.com/mrizalbasri/Hygeia/backend/internal/admin"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/consent"
	"github.com/mrizalbasri/Hygeia/backend/internal/httpapi"
	"github.com/mrizalbasri/Hygeia/backend/internal/laboratory"
	"github.com/mrizalbasri/Hygeia/backend/internal/notification"
	"github.com/mrizalbasri/Hygeia/backend/internal/prescription"
)

type fixture struct {
	service             *Service
	labService          *laboratory.Service
	consentService      *consent.Service
	prescriptionService *prescription.Service
	pool                *pgxpool.Pool
	api                 http.Handler
	tokens              map[string][]byte
	cookies             map[string]*http.Cookie
	ids                 map[string]string
	patientID           string
	csrf                string
	csrfCookie          *http.Cookie
}

// Uses a unique, committed schema to exercise independent concurrent transactions.
// Only this generated schema is removed; existing application tables are untouched.
func integrationFixture(t *testing.T) *fixture {
	t.Helper()
	connection := os.Getenv("TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, connection)
	if err != nil {
		t.Fatal("connect test database")
	}
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		t.Fatal(err)
	}
	schema := "care_test_" + hex.EncodeToString(random)
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE"); err != nil {
			t.Errorf("remove isolated test schema: %v", err)
		}
		admin.Close()
	})
	config, err := pgxpool.ParseConfig(connection)
	if err != nil {
		t.Fatal("parse test database")
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	config.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	for _, file := range []string{"000001_identity.up.sql", "000002_care_audit.up.sql", "000003_laboratory.up.sql", "000004_consents.up.sql", "000005_prescriptions.up.sql", "000006_admin.up.sql", "000007_notifications.up.sql"} {
		sql, err := db.Migrations.ReadFile("migrations/" + file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			t.Fatal(err)
		}
	}
	writer, _ := audit.NewWriter(bytes.Repeat([]byte{3}, 32))
	f := &fixture{pool: pool, service: &Service{DB: pool, Audit: writer}, tokens: map[string][]byte{}, cookies: map[string]*http.Cookie{}, ids: map[string]string{}}
	f.labService = &laboratory.Service{DB: pool, Audit: writer}
	f.consentService = &consent.Service{DB: pool, Audit: writer}
	f.prescriptionService = &prescription.Service{DB: pool, Audit: writer}
	var clinic, lab, pharmacy, otherLab, otherPharmacy string
	for _, org := range []struct {
		kind string
		id   *string
	}{{"clinic", &clinic}, {"laboratory", &lab}, {"pharmacy", &pharmacy}} {
		if err := pool.QueryRow(ctx, `INSERT INTO organizations(type,name) VALUES($1,$1) RETURNING id::text`, org.kind).Scan(org.id); err != nil {
			t.Fatal(err)
		}
	}
	if err := pool.QueryRow(ctx, `INSERT INTO organizations(type,name) VALUES('laboratory','Other lab') RETURNING id::text`).Scan(&otherLab); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO organizations(type,name) VALUES('pharmacy','Other pharmacy') RETURNING id::text`).Scan(&otherPharmacy); err != nil {
		t.Fatal(err)
	}
	for _, entry := range []struct{ name, role, org string }{
		{"patient", "patient", ""}, {"other", "patient", ""},
		{"doctor", "doctor", clinic}, {"colleague", "doctor", clinic},
		{"lab", "lab", lab}, {"pharmacy", "pharmacy", pharmacy}, {"admin", "admin", ""},
		{"labOther", "lab", otherLab}, {"labColleague", "lab", lab},
		{"pharmacyOther", "pharmacy", otherPharmacy}, {"pharmacyColleague", "pharmacy", pharmacy},
	} {
		var id string
		if err := pool.QueryRow(ctx, `INSERT INTO users(email,password_hash,full_name,role) VALUES($1,'unused-test-hash',$2,$3) RETURNING id::text`, strings.ToLower(entry.name)+"@care.test", "Synthetic "+entry.name, entry.role).Scan(&id); err != nil {
			t.Fatal(err)
		}
		f.ids[entry.name] = id
		if entry.org != "" {
			f.ids[entry.name+"Org"] = entry.org
		}
		if entry.org != "" {
			if _, err := pool.Exec(ctx, `INSERT INTO organization_memberships(user_id,organization_id,staff_role) VALUES($1,$2,$3)`, id, entry.org, entry.role); err != nil {
				t.Fatal(err)
			}
		}
		if entry.role == "patient" {
			var patient string
			if err := pool.QueryRow(ctx, `INSERT INTO patients(user_id,patient_ref) VALUES($1,$2) RETURNING id::text`, id, "TEST-"+entry.name).Scan(&patient); err != nil {
				t.Fatal(err)
			}
			if entry.name == "patient" {
				f.patientID = patient
			}
		}
		raw := base64.RawURLEncoding.EncodeToString(bytes.Repeat([]byte{byte(len(f.ids))}, 32))
		hash := sha256.Sum256([]byte(raw))
		f.tokens[entry.name] = hash[:]
		f.cookies[entry.name] = &http.Cookie{Name: "hygeia_session", Value: raw}
		if err := (auth.Store{DB: pool}).CreateSession(ctx, hash[:], id, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	login, err := auth.NewHandler(auth.Store{DB: pool}, false)
	if err != nil {
		t.Fatal(err)
	}
	handler := &Handler{Service: f.service}
	labHandler := &laboratory.Handler{Service: f.labService}
	consentHandler := &consent.Handler{Service: f.consentService}
	prescriptionHandler := &prescription.Handler{Service: f.prescriptionService}
	adminHandler := &adminapi.Handler{Service: &adminapi.Service{DB: pool}}
	notificationHandler := &notification.Handler{Service: &notification.Service{DB: pool}}
	f.api = httpapi.NewAuthenticatedHandler(slog.New(slog.NewTextHandler(io.Discard, nil)), pool.Ping, login, bytes.Repeat([]byte{4}, 32), false, handler.Register, labHandler.Register, consentHandler.Register, prescriptionHandler.Register, adminHandler.Register, notificationHandler.Register)
	csrf := httptest.NewRecorder()
	f.api.ServeHTTP(csrf, httptest.NewRequest("GET", "http://localhost/api/v1/auth/csrf", nil))
	var body struct {
		Token string `json:"csrf_token"`
	}
	if err := json.Unmarshal(csrf.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	f.csrf = body.Token
	for _, cookie := range csrf.Result().Cookies() {
		if cookie.Name == "hygeia_csrf" {
			f.csrfCookie = cookie
		}
	}
	if f.csrfCookie == nil {
		t.Fatal("missing CSRF cookie")
	}
	return f
}

func (f *fixture) request(actor, method, path, body string, csrf bool) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://localhost")
	if cookie := f.cookies[actor]; cookie != nil {
		r.AddCookie(cookie)
	}
	if csrf {
		r.AddCookie(f.csrfCookie)
		r.Header.Set("X-CSRF-Token", f.csrf)
	}
	result := httptest.NewRecorder()
	f.api.ServeHTTP(result, r)
	return result
}

func expectStatus(t *testing.T, r *httptest.ResponseRecorder, status int) {
	t.Helper()
	if r.Code != status {
		t.Fatalf("want status %d; got %d: %s", status, r.Code, r.Body.String())
	}
	if r.Header().Get("X-Request-ID") == "" || r.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing request ID or cache policy")
	}
}

func responseEpisode(t *testing.T, r *httptest.ResponseRecorder) Episode {
	t.Helper()
	var body struct {
		Episode Episode `json:"episode"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Episode
}

func TestCareIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	expectStatus(t, f.request("", "GET", "/api/v1/episodes", "", false), 401)
	expectStatus(t, f.request("doctor", "POST", "/api/v1/episodes", `{"patient_ref":"TEST-patient"}`, false), 403)
	expectStatus(t, f.request("patient", "POST", "/api/v1/episodes", `{"patient_ref":"TEST-patient"}`, true), 403)
	expectStatus(t, f.request("doctor", "POST", "/api/v1/episodes", `{"patient_ref":"TEST-patient","doctor_id":"spoof"}`, true), 400)
	expectStatus(t, f.request("doctor", "POST", "/api/v1/episodes", `{"patient_ref":"test-patient"}`, true), 403)
	expectStatus(t, f.request("doctor", "GET", "/api/v1/episodes?limit=101", "", false), 400)
	expectStatus(t, f.request("doctor", "GET", "/api/v1/episodes/not-a-uuid", "", false), 400)
	invite := f.request("doctor", "POST", "/api/v1/episodes", `{"patient_ref":"TEST-patient"}`, true)
	expectStatus(t, invite, 201)
	episode := responseEpisode(t, invite)
	if episode.Status != "pending" || episode.DoctorID != f.ids["doctor"] || episode.PatientID != f.patientID || episode.AcceptedAt != nil {
		t.Fatal("invalid pending episode")
	}
	if invite.Header().Get("Location") != "/api/v1/episodes/"+episode.ID {
		t.Fatal("missing Location")
	}
	path := "/api/v1/episodes/" + episode.ID
	expectStatus(t, f.request("doctor", "POST", "/api/v1/episodes", `{"patient_ref":"TEST-patient"}`, true), 409)
	expectStatus(t, f.request("doctor", "PATCH", path, `{"action":"accept"}`, true), 403)
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"end"}`, true), 409)

	denied := f.request("colleague", "GET", path, "", false)
	expectStatus(t, denied, 403)
	var deniedBody map[string]any
	if err := json.Unmarshal(denied.Body.Bytes(), &deniedBody); err != nil {
		t.Fatal(err)
	}
	if len(deniedBody) != 1 || deniedBody["error"] == nil || strings.Contains(denied.Body.String(), episode.PatientRef) || strings.Contains(denied.Body.String(), episode.ID) {
		t.Fatal("denied response leaked resource")
	}
	for _, actor := range []string{"other", "lab", "pharmacy", "admin"} {
		expectStatus(t, f.request(actor, "GET", path, "", false), 403)
		expectStatus(t, f.request(actor, "PATCH", path, `{"action":"accept"}`, true), 403)
	}
	unknown := f.request("colleague", "GET", "/api/v1/episodes/00000000-0000-0000-0000-000000000001", "", false)
	expectStatus(t, unknown, 403)
	// Error messages for unknown and inaccessible UUIDs must match (IDs differ).
	var unknownBody struct {
		Error struct{ Code, Message string }
	}
	if err := json.Unmarshal(unknown.Body.Bytes(), &unknownBody); err != nil {
		t.Fatal(err)
	}
	if unknownBody.Error.Code != "forbidden" || unknownBody.Error.Message != "Akses tidak diizinkan." {
		t.Fatal("resource existence leaked")
	}
	expectStatus(t, f.request("patient", "GET", path, "", false), 200)
	ownList := f.request("patient", "GET", "/api/v1/episodes?status=pending", "", false)
	expectStatus(t, ownList, 200)
	if !strings.Contains(ownList.Body.String(), episode.ID) {
		t.Fatal("patient cannot list invitation")
	}
	for _, actor := range []string{"other", "colleague"} {
		r := f.request(actor, "GET", "/api/v1/episodes", "", false)
		expectStatus(t, r, 200)
		if !strings.Contains(r.Body.String(), `"episodes":[]`) {
			t.Fatal("list leaked another person's episode")
		}
	}
	expectStatus(t, f.request("admin", "GET", "/api/v1/episodes", "", false), 403)
	accepted := f.request("patient", "PATCH", path, `{"action":"accept"}`, true)
	expectStatus(t, accepted, 200)
	active := responseEpisode(t, accepted)
	if active.Status != "active" || active.AcceptedAt == nil || active.ClosedAt != nil {
		t.Fatal("invalid accepted timestamps")
	}
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"accept"}`, true), 409)
	ended := f.request("doctor", "PATCH", path, `{"action":"end"}`, true)
	expectStatus(t, ended, 200)
	closed := responseEpisode(t, ended)
	if closed.Status != "ended" || closed.ClosedAt == nil {
		t.Fatal("invalid ended episode")
	}
	expectStatus(t, f.request("patient", "PATCH", path, `{"action":"end"}`, true), 409)
	// History remains available; clinical access is tested independently in policy.
	expectStatus(t, f.request("doctor", "GET", path, "", false), 200)
	reinvite := f.request("doctor", "POST", "/api/v1/episodes", `{"patient_ref":"TEST-patient"}`, true)
	expectStatus(t, reinvite, 201)
	second := responseEpisode(t, reinvite)
	rejected := f.request("patient", "PATCH", "/api/v1/episodes/"+second.ID, `{"action":"reject"}`, true)
	expectStatus(t, rejected, 200)
	rejectEpisode := responseEpisode(t, rejected)
	if rejectEpisode.Status != "rejected" || rejectEpisode.AcceptedAt != nil || rejectEpisode.ClosedAt == nil {
		t.Fatal("invalid rejected episode")
	}
	// Rejections commit audit even though the caller receives 403 or 409.
	var denies int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE chain_id=$1 AND outcome='deny'`, f.patientID).Scan(&denies); err != nil || denies < 8 {
		t.Fatalf("denied events lost: %d %v", denies, err)
	}
	summary := f.request("patient", "GET", "/api/v1/patient/access-summary?limit=100", "", false)
	expectStatus(t, summary, 200)
	if !strings.Contains(summary.Body.String(), "Synthetic colleague") || !strings.Contains(summary.Body.String(), "ditolak") {
		t.Fatal("patient access summary incomplete")
	}
	expectStatus(t, f.request("doctor", "GET", "/api/v1/patient/access-summary", "", false), 403)
	expectStatus(t, f.request("other", "GET", "/api/v1/audit/verify?chain="+f.patientID, "", false), 403)
	expectStatus(t, f.request("doctor", "GET", "/api/v1/audit/verify?chain="+f.patientID, "", false), 403)
	for _, actor := range []string{"patient", "admin"} {
		verify := f.request(actor, "GET", "/api/v1/audit/verify?chain="+f.patientID, "", false)
		expectStatus(t, verify, 200)
		if !strings.Contains(verify.Body.String(), `"intact":true`) || strings.Contains(verify.Body.String(), "actor_name") {
			t.Fatal("invalid verification or admin sees event details")
		}
	}
	// Disabled and expired sessions are checked for domain endpoints too.
	if _, err := f.pool.Exec(ctx, `UPDATE users SET disabled_at=now() WHERE id=$1`, f.ids["doctor"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("doctor", "GET", path, "", false), 401)
	if _, err := f.pool.Exec(ctx, `UPDATE users SET disabled_at=NULL WHERE id=$1`, f.ids["doctor"]); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE sessions SET created_at=now()-interval '2 hours',expires_at=now()-interval '1 hour' WHERE user_id=$1`, f.ids["doctor"]); err != nil {
		t.Fatal(err)
	}
	expectStatus(t, f.request("doctor", "GET", path, "", false), 401)
}

func TestCareConcurrencyAndAuditFailureIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	// Concurrent invites must produce one pending relationship.
	var group sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := f.service.Invite(ctx, f.tokens["doctor"], "TEST-patient")
			results <- err
		}()
	}
	group.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("invite race: %d successes %d conflicts", successes, conflicts)
	}
	list, err := f.service.List(ctx, f.tokens["patient"], "pending", 20, 0)
	if err != nil || len(list) != 1 {
		t.Fatal("duplicate invitation")
	}
	e := list[0]
	// Race on a state transition: exactly one decision wins.
	results = make(chan error, 2)
	for _, action := range []string{"accept", "reject"} {
		group.Add(1)
		go func(action string) {
			defer group.Done()
			_, err := f.service.Transition(ctx, f.tokens["patient"], e.ID, action)
			results <- err
		}(action)
	}
	group.Wait()
	close(results)
	successes, conflicts = 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatal("concurrent decisions both succeeded")
	}
	// Many independent readers append to the same patient's chain.
	errorsCh := make(chan error, 16)
	for i := 0; i < 16; i++ {
		group.Add(1)
		go func() { defer group.Done(); _, err := f.service.Get(ctx, f.tokens["patient"], e.ID); errorsCh <- err }()
	}
	group.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	verification, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verification.Intact || verification.Events != 21 {
		t.Fatalf("concurrent audit: %+v %v", verification, err)
	}

	// Force audit failure in this test schema; no response data or business write
	// may survive it. The production append-only trigger remains untouched.
	pending, err := f.service.Invite(ctx, f.tokens["colleague"], "TEST-other")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `CREATE FUNCTION fail_audit_insert() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test audit unavailable'; END $$;
        CREATE TRIGGER fail_audit_insert BEFORE INSERT ON audit_events FOR EACH ROW EXECUTE FUNCTION fail_audit_insert()`); err != nil {
		t.Fatal(err)
	}
	r := f.request("doctor", "POST", "/api/v1/episodes", `{"patient_ref":"TEST-other"}`, true)
	expectStatus(t, r, 503)
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM care_episodes e JOIN patients p ON p.id=e.patient_id WHERE p.patient_ref='TEST-other' AND e.doctor_id=$1`, f.ids["doctor"]).Scan(&count); err != nil || count != 0 {
		t.Fatal("business mutation survived audit failure")
	}
	expectStatus(t, f.request("other", "PATCH", "/api/v1/episodes/"+pending.ID, `{"action":"accept"}`, true), 503)
	var status string
	if err := f.pool.QueryRow(ctx, `SELECT status FROM care_episodes WHERE id=$1`, pending.ID).Scan(&status); err != nil || status != "pending" {
		t.Fatal("status change survived audit failure")
	}
	expectStatus(t, f.request("patient", "GET", "/api/v1/episodes/"+e.ID, "", false), 503)
	if _, err := f.pool.Exec(ctx, `DROP TRIGGER fail_audit_insert ON audit_events; DROP FUNCTION fail_audit_insert()`); err != nil {
		t.Fatal(err)
	}
	// Trigger rejects ordinary tampering.
	if _, err := f.pool.Exec(ctx, `UPDATE audit_events SET basis='tampered' WHERE chain_id=$1 AND seq=1`, f.patientID); err == nil {
		t.Fatal("audit update accepted")
	}
	if _, err := f.pool.Exec(ctx, `DELETE FROM audit_events WHERE chain_id=$1 AND seq=1`, f.patientID); err == nil {
		t.Fatal("audit delete accepted")
	}
	if _, err := f.pool.Exec(ctx, `TRUNCATE audit_events`); err == nil {
		t.Fatal("audit truncate accepted")
	}
	// Simulate a privileged DB owner bypassing the guard, only in this schema.
	if _, err := f.pool.Exec(ctx, `ALTER TABLE audit_events DISABLE TRIGGER audit_events_immutable`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE audit_events SET basis='tampered' WHERE chain_id=$1 AND seq=2`, f.patientID); err != nil {
		t.Fatal(err)
	}
	verification, err = f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || verification.Intact || verification.FirstMismatchSeq == nil || *verification.FirstMismatchSeq != 2 {
		t.Fatalf("tampering not detected: %+v %v", verification, err)
	}
	// Restore payload, then remove the tail: verification must compare the head.
	if _, err := f.pool.Exec(ctx, `UPDATE audit_events SET basis='invalid_state' WHERE chain_id=$1 AND seq=2`, f.patientID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE audit_events SET context=jsonb_set(context,'{tampered}', '123') WHERE chain_id=$1 AND seq=3`, f.patientID); err != nil {
		t.Fatal(err)
	}
	verification, err = f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || verification.Intact || verification.FirstMismatchSeq == nil || *verification.FirstMismatchSeq != 3 {
		t.Fatalf("numeric context tampering not detected: %+v %v", verification, err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE audit_events SET context=context-'tampered' WHERE chain_id=$1 AND seq=3`, f.patientID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `DELETE FROM audit_events WHERE chain_id=$1 AND seq=21`, f.patientID); err != nil {
		t.Fatal(err)
	}
	verification, err = f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || verification.Intact || verification.FirstMismatchSeq == nil || *verification.FirstMismatchSeq != 21 {
		t.Fatalf("tail deletion not detected: %+v %v", verification, err)
	}
}

func TestHandlerInputValidation(t *testing.T) {
	h := &Handler{}
	mux := http.NewServeMux()
	h.Register(mux)
	for _, tc := range []struct {
		method, path, body, media string
		status                    int
	}{
		{"POST", "/api/v1/episodes", `{}`, "text/plain", 415},
		{"POST", "/api/v1/episodes", `{"patient_ref":"TEST"} {}`, "application/json", 400},
		{"POST", "/api/v1/episodes", `{"patient_ref":"` + strings.Repeat("A", 5000) + `"}`, "application/json", 400},
		{"POST", "/api/v1/episodes", `{"patient_ref":"%"}`, "application/json", 400},
		{"GET", "/api/v1/episodes?status=secret", "", "", 400},
		{"GET", "/api/v1/episodes?offset=-1", "", "", 400},
		{"GET", "/api/v1/episodes?limit=0", "", "", 400},
		{"GET", "/api/v1/audit/verify?chain=invalid", "", "", 400},
		{"PATCH", "/api/v1/episodes/00000000-0000-0000-0000-000000000001", `{"action":"delete"}`, "application/json", 400},
	} {
		t.Run(fmt.Sprintf("%s_%s_%d", tc.method, tc.path, tc.status), func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			r.Header.Set("Content-Type", tc.media)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("status %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
