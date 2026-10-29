package auth

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/go-jose/go-jose/v3"
	josejwt "github.com/go-jose/go-jose/v3/jwt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mrizalbasri/Hygeia/backend/db"
)

type fakeClerk struct {
	mu       sync.Mutex
	identity ClerkIdentity
	err      error
	revoked  bool
}

func (p *fakeClerk) Inspect(_ context.Context, sid, uid string) (ClerkIdentity, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return ClerkIdentity{}, p.err
	}
	if p.revoked || sid != p.identity.SessionID || uid != p.identity.UserID {
		return ClerkIdentity{}, pgx.ErrNoRows
	}
	return p.identity, nil
}
func (p *fakeClerk) Revoke(context.Context, string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.revoked = true
	return nil
}

func clerkFixture(t *testing.T) (*pgxpool.Pool, *ClerkGate, *fakeClerk, *rsa.PrivateKey) {
	t.Helper()
	connection := os.Getenv("TEST_DATABASE_URL")
	if connection == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, connection)
	if err != nil {
		t.Fatal("connect test database")
	}
	random := make([]byte, 8)
	rand.Read(random)
	schema := "clerk_test_" + hex.EncodeToString(random)
	quoted := pgx.Identifier{schema}.Sanitize()
	if _, err = pool.Exec(ctx, "CREATE SCHEMA "+quoted); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, err := pool.Exec(context.Background(), "DROP SCHEMA "+quoted+" CASCADE")
		if err != nil {
			t.Error(err)
		}
		pool.Close()
	})
	config, _ := pgxpool.ParseConfig(connection)
	// Set session state explicitly: remote proxies may ignore startup search_path.
	// Verify isolation before any migration or fixture writes can reach public.
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		if _, err := conn.Exec(ctx, "SELECT set_config('search_path', $1, false)", quoted); err != nil {
			return err
		}
		var current string
		if err := conn.QueryRow(ctx, "SELECT current_schema()").Scan(&current); err != nil {
			return err
		}
		if current != schema {
			return errors.New("test connection did not select the isolated schema")
		}
		return nil
	}
	scoped, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(scoped.Close)
	files, _ := fs.ReadDir(db.Migrations, "migrations")
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".up.sql") {
			sql, _ := db.Migrations.ReadFile("migrations/" + file.Name())
			if _, err := scoped.Exec(ctx, string(sql)); err != nil {
				t.Fatal(err)
			}
		}
	}
	secret := bytes.Repeat([]byte{7}, 32)
	gate, err := NewClerkGate(Store{DB: scoped}, "pk_test_"+base64.RawStdEncoding.EncodeToString([]byte("synthetic.clerk.accounts.dev$")), "sk_test_synthetic", []string{"http://localhost:5173"}, secret)
	if err != nil {
		t.Fatal(err)
	}
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	gate.verify = clerkhttp.WithHeaderAuthorization(clerkhttp.AuthorizationJWTExtractor(clerkToken), clerkhttp.JSONWebKey(string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: encoded}))), clerkhttp.AuthorizedParty(func(origin string) bool { return origin == "http://localhost:5173" }), clerkhttp.CustomClaimsConstructor(func(context.Context) any { return &clerkFactors{} }), clerkhttp.AuthorizationFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { writeError(w, 401, "unauthenticated", "Invalid session") })))
	provider := &fakeClerk{identity: ClerkIdentity{UserID: "user_Synthetic123", SessionID: "sess_Synthetic123", Email: "synthetic@clerk.test", Name: "Synthetic Clerk", ExpiresAt: time.Now().Add(time.Hour)}}
	gate.Provider = provider
	return scoped, gate, provider, key
}
func signClerk(t *testing.T, key *rsa.PrivateKey, overrides map[string]any) string {
	t.Helper()
	claims := map[string]any{"iss": "https://synthetic.clerk.accounts.dev", "sub": "user_Synthetic123", "sid": "sess_Synthetic123", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "nbf": time.Now().Add(-time.Second).Unix(), "azp": "http://localhost:5173", "fva": []int{0, -1}}
	for key, value := range overrides {
		if value == nil {
			delete(claims, key)
		} else {
			claims[key] = value
		}
	}
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "synthetic"))
	if err != nil {
		t.Fatal(err)
	}
	token, err := josejwt.Signed(signer).Claims(claims).CompactSerialize()
	if err != nil {
		t.Fatal(err)
	}
	return token
}
func TestClerkJWTAndIdentityIntegration(t *testing.T) {
	pool, gate, provider, key := clerkFixture(t)
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, err := gate.Store.UserBySession(r.Context(), SessionHash(r))
		if err != nil {
			t.Error(err)
			w.WriteHeader(503)
			return
		}
		writeJSON(w, 200, map[string]any{"user": u})
	})
	api := gate.Wrap(next)
	request := func(token string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "http://localhost/api/v1/me", nil)
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		w := httptest.NewRecorder()
		api.ServeHTTP(w, r)
		return w
	}
	token := signClerk(t, key, nil)
	w := request(token)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"role":"patient"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, bad := range []map[string]any{{"iss": "https://other.clerk.test"}, {"azp": "https://evil.test"}, {"azp": nil}, {"exp": time.Now().Add(-time.Hour).Unix()}, {"sid": "sess_Other123"}} {
		if w := request(signClerk(t, key, bad)); w.Code != 401 {
			t.Fatalf("bad claims accepted: %v %d", bad, w.Code)
		}
	}
	if w := request(signClerk(t, key, map[string]any{"iss": "https://other.clerk.accounts.dev"})); w.Code != 401 {
		t.Fatal("another Clerk issuer accepted")
	}
	cookieRequest := httptest.NewRequest("GET", "http://localhost/api/v1/me", nil)
	cookieRequest.AddCookie(&http.Cookie{Name: "__session", Value: token})
	cookieRequest.AddCookie(&http.Cookie{Name: sessionCookie, Value: strings.Repeat("B", 43)})
	w = httptest.NewRecorder()
	api.ServeHTTP(w, cookieRequest)
	if w.Code != 200 {
		t.Fatal("Clerk cookie or stripping of legacy cookie failed", w.Code)
	}
	if w := request(token + "tampered"); w.Code != 401 {
		t.Fatal("tampered JWT accepted")
	}
	// Browser metadata cannot assign a staff role or organization.
	w = request(signClerk(t, key, map[string]any{"public_metadata": map[string]string{"role": "admin"}, "org_role": "org:admin"}))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"role":"patient"`) {
		t.Fatal("metadata escalated role")
	}
	legacy := httptest.NewRequest("GET", "http://localhost/api/v1/me", nil)
	legacy.AddCookie(&http.Cookie{Name: sessionCookie, Value: strings.Repeat("A", 43)})
	w = httptest.NewRecorder()
	api.ServeHTTP(w, legacy)
	if w.Code != 401 {
		t.Fatal("local cookie bypassed Clerk")
	}
	provider.err = errors.New("offline")
	if w := request(token); w.Code != 503 {
		t.Fatal("provider outage allowed access", w.Code)
	}
	provider.err = nil
	provider.revoked = true
	if w := request(token); w.Code != 401 {
		t.Fatal("revoked session accepted")
	}
	provider.revoked = false
	_, err := pool.Exec(context.Background(), `UPDATE users SET disabled_at=now() WHERE clerk_user_id=$1`, provider.identity.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if w := request(token); w.Code != 401 {
		t.Fatal("disabled local user accepted")
	}
}
func TestClerkStaffMFAAndSSERevalidationIntegration(t *testing.T) {
	pool, gate, provider, key := clerkFixture(t)
	ctx := context.Background()
	if _, _, err := gate.resolve(ctx, provider.identity); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE users SET role='admin' WHERE clerk_user_id=$1`, provider.identity.UserID); err != nil {
		t.Fatal(err)
	}
	h, _ := NewHandler(Store{DB: pool}, false)
	h.UseClerk(gate)
	mux := http.NewServeMux()
	h.Register(mux)
	mux.HandleFunc("GET /api/v1/long-read", func(w http.ResponseWriter, r *http.Request) {
		if _, err := gate.Store.UserBySession(r.Context(), SessionHash(r)); err != nil {
			t.Fatal(err)
		}
		provider.mu.Lock()
		provider.revoked = true
		provider.mu.Unlock()
		time.Sleep(1100 * time.Millisecond)
		if _, err := gate.Store.UserBySession(r.Context(), SessionHash(r)); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatal("long-lived stream retained revoked access", err)
		}
		w.WriteHeader(204)
	})
	api := gate.Wrap(Protect(mux, bytes.Repeat([]byte{5}, 32), false))
	request := func(path string, fva any) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "http://localhost"+path, nil)
		r.Header.Set("Authorization", "Bearer "+signClerk(t, key, map[string]any{"fva": fva}))
		w := httptest.NewRecorder()
		api.ServeHTTP(w, r)
		return w
	}
	if w := request("/api/v1/me", []int{0, -1}); w.Code != 403 {
		t.Fatal("staff without MFA accepted", w.Code)
	}
	if w := request("/api/v1/auth/status", []int{0, -1}); w.Code != 200 || !strings.Contains(w.Body.String(), `"requires_mfa":true`) {
		t.Fatal("MFA setup blocked", w.Code, w.Body.String())
	}
	provider.identity.MFAEnabled = true
	if w := request("/api/v1/me", nil); w.Code != 403 {
		t.Fatal("missing fva accepted for staff")
	}
	if w := request("/api/v1/me", []int{0, 0}); w.Code != 200 {
		t.Fatal("valid MFA rejected", w.Code, w.Body.String())
	}
	if w := request("/api/v1/long-read", []int{0, 0}); w.Code != 204 {
		t.Fatal("stream failed")
	}
}
func TestClerkNeverAutoLinksEmailIntegration(t *testing.T) {
	pool, gate, provider, _ := clerkFixture(t)
	_, err := pool.Exec(context.Background(), `INSERT INTO users(email,password_hash,full_name,role) VALUES($1,'unused','Existing administrator','admin')`, provider.identity.Email)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := gate.resolve(context.Background(), provider.identity); !errors.Is(err, ErrClerkIdentityConflict) {
		t.Fatal("email claimed existing account", err)
	}
	var linked *string
	if err := pool.QueryRow(context.Background(), `SELECT clerk_user_id FROM users WHERE email=$1`, provider.identity.Email).Scan(&linked); err != nil || linked != nil {
		t.Fatal("existing account altered")
	}
}
func TestClerkConfiguration(t *testing.T) {
	if _, err := NewClerkGate(Store{}, "bad", "bad", nil, nil); err == nil {
		t.Fatal("bad config accepted")
	}
}
