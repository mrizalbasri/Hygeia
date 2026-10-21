package auth

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mrizalbasri/Hygeia/backend/db"
)

func TestPasswords(t *testing.T) {
	hash, err := HashPassword("Synthetic-password-2026")
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyPassword("Synthetic-password-2026", hash) {
		t.Fatal("correct password rejected")
	}
	if VerifyPassword("incorrect", hash) {
		t.Fatal("incorrect password accepted")
	}
	if VerifyPassword("anything", "$argon2id$v=19$m=999999999,t=3,p=2$invalid$hash") {
		t.Fatal("unbounded parameters accepted")
	}
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("short seed password accepted")
	}
}

func TestCSRFAndLoginLimit(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /mutate", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	protected := Protect(mux, bytes.Repeat([]byte{1}, 32), false)
	response := httptest.NewRecorder()
	protected.ServeHTTP(response, httptest.NewRequest("POST", "http://localhost/mutate", nil))
	if response.Code != 403 {
		t.Fatalf("missing CSRF status = %d", response.Code)
	}
	h := &Handler{limits: make(map[string]*loginLimit)}
	for i := 0; i < 5; i++ {
		if !h.allowLogin("127.0.0.1:1000") {
			t.Fatal("initial burst unexpectedly denied")
		}
	}
	if h.allowLogin("127.0.0.1:1001") {
		t.Fatal("changing source port bypassed IP rate limit")
	}
	if !h.allowLogin("127.0.0.2:1001") {
		t.Fatal("different IP shares login quota")
	}
}

// Uses an isolated transaction/schema and rolls back all test data.
// Set TEST_DATABASE_URL explicitly; production/demo tables are never modified.
func TestAuthenticationIntegration(t *testing.T) {
	connection := os.Getenv("TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connection)
	if err != nil {
		t.Fatal("connect test database")
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	schema := "auth_test_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
	if _, err := tx.Exec(ctx, "CREATE SCHEMA "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+pgx.Identifier{schema}.Sanitize()); err != nil {
		t.Fatal(err)
	}
	migration, err := db.Migrations.ReadFile("migrations/000001_identity.up.sql")
	if err != nil {
		t.Fatal(err)
	}
	// The test already owns the outer transaction; strip migration transaction markers.
	sql := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(string(migration)), "BEGIN;"), "COMMIT;")
	if _, err := tx.Exec(ctx, sql); err != nil {
		t.Fatal(err)
	}
	password := "Synthetic-password-2026"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	var userID string
	if err := tx.QueryRow(ctx, `INSERT INTO users(email,password_hash,full_name,role) VALUES('test@hygeia.test',$1,'Synthetic Patient','patient') RETURNING id::text`, hash).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO patients(user_id,patient_ref) VALUES($1,'SYNTHETIC-SELF-REF')`, userID); err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(Store{DB: tx}, false)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler.Register(mux)
	api := Protect(mux, bytes.Repeat([]byte{2}, 32), false)

	csrfResponse := httptest.NewRecorder()
	api.ServeHTTP(csrfResponse, httptest.NewRequest("GET", "http://localhost/api/v1/auth/csrf", nil))
	var csrfBody struct {
		Token string `json:"csrf_token"`
	}
	if err := json.Unmarshal(csrfResponse.Body.Bytes(), &csrfBody); err != nil {
		t.Fatal(err)
	}
	csrfCookies := csrfResponse.Result().Cookies()
	request := func(method, path, body, token string, cookies []*http.Cookie, origin string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)
		if token != "" {
			r.Header.Set("X-CSRF-Token", token)
		}
		for _, c := range cookies {
			r.AddCookie(c)
		}
		w := httptest.NewRecorder()
		api.ServeHTTP(w, r)
		return w
	}
	credentials := `{"email":"TEST@HYGEIA.TEST","password":"` + password + `"}`
	if got := request("POST", "/api/v1/auth/login", credentials, "", csrfCookies, "http://localhost"); got.Code != 403 {
		t.Fatalf("CSRF bypass: %d", got.Code)
	}
	if got := request("POST", "/api/v1/auth/login", credentials, csrfBody.Token, csrfCookies, "https://evil.example"); got.Code != 403 {
		t.Fatalf("cross-origin accepted: %d", got.Code)
	}
	wrong := request("POST", "/api/v1/auth/login", `{"email":"test@hygeia.test","password":"incorrect"}`, csrfBody.Token, csrfCookies, "http://localhost")
	unknown := request("POST", "/api/v1/auth/login", `{"email":"unknown@hygeia.test","password":"incorrect"}`, csrfBody.Token, csrfCookies, "http://localhost")
	if wrong.Code != 401 || unknown.Code != 401 || wrong.Body.String() != unknown.Body.String() {
		t.Fatal("credential failure leaks account existence")
	}
	login := request("POST", "/api/v1/auth/login", credentials, csrfBody.Token, csrfCookies, "http://localhost")
	if login.Code != 200 {
		t.Fatalf("login: %d %s", login.Code, login.Body.String())
	}
	var session *http.Cookie
	for _, cookie := range login.Result().Cookies() {
		if cookie.Name == sessionCookie {
			session = cookie
		}
	}
	if session == nil || !session.HttpOnly || session.SameSite != http.SameSiteLaxMode || session.MaxAge != int(sessionTTL.Seconds()) {
		t.Fatal("invalid session cookie")
	}
	sessionHash := sha256.Sum256([]byte(session.Value))
	var stored []byte
	if err := tx.QueryRow(ctx, `SELECT token_hash FROM sessions WHERE user_id=$1`, userID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, sessionHash[:]) || bytes.Equal(stored, []byte(session.Value)) {
		t.Fatal("raw session stored in database")
	}
	cookies := append(append([]*http.Cookie{}, csrfCookies...), session)
	if got := request("GET", "/api/v1/me", "", "", cookies, ""); got.Code != 200 || strings.Contains(got.Body.String(), "password_hash") {
		t.Fatalf("me: %d", got.Code)
	}
	var identity struct {
		User User `json:"user"`
	}
	if err := json.Unmarshal(request("GET", "/api/v1/me", "", "", cookies, "").Body.Bytes(), &identity); err != nil || identity.User.PatientRef == nil || *identity.User.PatientRef != "SYNTHETIC-SELF-REF" {
		t.Fatalf("patient cannot read own reference: %v", err)
	}
	if got := request("GET", "/api/v1/me", "", "", nil, ""); got.Code != 401 {
		t.Fatal("unauthenticated access allowed")
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET disabled_at=now() WHERE id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	if got := request("GET", "/api/v1/me", "", "", cookies, ""); got.Code != 401 {
		t.Fatal("disabled account retained access")
	}
	if _, err := tx.Exec(ctx, `UPDATE users SET disabled_at=NULL WHERE id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET created_at=now()-interval '2 days',expires_at=now()-interval '1 day' WHERE user_id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	if got := request("GET", "/api/v1/me", "", "", cookies, ""); got.Code != 401 {
		t.Fatal("expired session accepted")
	}
	if _, err := tx.Exec(ctx, `UPDATE sessions SET expires_at=now()+interval '1 hour' WHERE user_id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	if got := request("POST", "/api/v1/auth/logout", "{}", csrfBody.Token, cookies, "http://localhost"); got.Code != 204 {
		t.Fatalf("logout: %d", got.Code)
	}
	if got := request("GET", "/api/v1/me", "", "", cookies, ""); got.Code != 401 {
		t.Fatal("revoked session accepted")
	}

	// Organization identity is read from the DB, never accepted from request input.
	var doctorID, orgID string
	if err := tx.QueryRow(ctx, `INSERT INTO organizations(name,type) VALUES('Synthetic Clinic','clinic') RETURNING id::text`).Scan(&orgID); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `INSERT INTO users(email,password_hash,full_name,role) VALUES('doctor@hygeia.test',$1,'Synthetic Doctor','doctor') RETURNING id::text`, hash).Scan(&doctorID); err != nil {
		t.Fatal(err)
	}
	store := Store{DB: tx}
	key := sha256.Sum256([]byte("synthetic-doctor-token"))
	if err := store.CreateSession(ctx, key[:], doctorID, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.UserBySession(ctx, key[:]); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatal("staff without organization accepted")
	}
	if _, err := tx.Exec(ctx, `INSERT INTO organization_memberships(user_id,organization_id,staff_role) VALUES($1,$2,'doctor')`, doctorID, orgID); err != nil {
		t.Fatal(err)
	}
	doctor, err := store.UserBySession(ctx, key[:])
	if err != nil || doctor.Role != "doctor" || doctor.Organization == nil || doctor.Organization.ID != orgID {
		t.Fatal("incorrect organization identity")
	}
}
