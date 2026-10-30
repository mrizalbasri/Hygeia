package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/clerk/clerk-sdk-go/v2/jwks"
	"github.com/clerk/clerk-sdk-go/v2/session"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var ErrClerkIdentityConflict = errors.New("existing account requires explicit identity link")

type ClerkIdentity struct {
	UserID, SessionID, Email, Name string
	ExpiresAt                      time.Time
	MFAEnabled                     bool
}
type ClerkProvider interface {
	Inspect(context.Context, string, string) (ClerkIdentity, error)
	Revoke(context.Context, string) error
}
type sdkProvider struct {
	sessions *session.Client
	users    *user.Client
}

func (p *sdkProvider) Inspect(ctx context.Context, sid, uid string) (ClerkIdentity, error) {
	s, err := p.sessions.Get(ctx, sid)
	if err != nil {
		return ClerkIdentity{}, providerError(err)
	}
	if s.UserID != uid || s.Status != "active" || s.ExpireAt <= time.Now().UnixMilli() {
		return ClerkIdentity{}, pgx.ErrNoRows
	}
	identity, err := p.LookupUser(ctx, uid)
	if err != nil {
		return ClerkIdentity{}, err
	}
	identity.SessionID = sid
	identity.ExpiresAt = time.UnixMilli(s.ExpireAt)
	return identity, nil
}

func (p *sdkProvider) LookupUser(ctx context.Context, uid string) (ClerkIdentity, error) {
	u, err := p.users.Get(ctx, uid)
	if err != nil {
		return ClerkIdentity{}, providerError(err)
	}
	if u.Banned || u.Locked || u.PrimaryEmailAddressID == nil {
		return ClerkIdentity{}, pgx.ErrNoRows
	}
	email := ""
	for _, e := range u.EmailAddresses {
		if e.ID == *u.PrimaryEmailAddressID && e.Verification != nil && e.Verification.Status == "verified" {
			email = strings.ToLower(strings.TrimSpace(e.EmailAddress))
		}
	}
	if email == "" {
		return ClerkIdentity{}, pgx.ErrNoRows
	}
	name := ""
	if u.FirstName != nil {
		name = *u.FirstName
	}
	if u.LastName != nil {
		name += " " + *u.LastName
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Pengguna Hygeia"
	}
	if utf8.RuneCountInString(name) > 200 {
		name = string([]rune(name)[:200])
	}
	return ClerkIdentity{UserID: uid, Email: email, Name: name, MFAEnabled: u.TwoFactorEnabled}, nil
}
func (p *sdkProvider) Revoke(ctx context.Context, id string) error {
	_, err := p.sessions.Revoke(ctx, &session.RevokeParams{ID: id})
	return providerError(err)
}
func providerError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *clerk.APIErrorResponse
	if errors.As(err, &apiErr) && (apiErr.HTTPStatusCode == 404 || apiErr.HTTPStatusCode == 401) {
		return pgx.ErrNoRows
	}
	return errors.New("Clerk unavailable")
}

type clerkFactors struct {
	FVA []int64 `json:"fva"`
}
type clerkRequest struct {
	check       func(context.Context) error
	sessionID   string
	requiresMFA bool
}
type clerkContextKey struct{}

type ClerkGate struct {
	Store                  Store
	Provider               ClerkProvider
	LookupUser             func(context.Context, string) (ClerkIdentity, error)
	PublishableKey, Issuer string
	key                    []byte
	verify                 func(http.Handler) http.Handler
}

func NewClerkGate(store Store, publishable, secret string, parties []string, key []byte) (*ClerkGate, error) {
	if (!strings.HasPrefix(publishable, "pk_test_") && !strings.HasPrefix(publishable, "pk_live_")) || (!strings.HasPrefix(secret, "sk_test_") && !strings.HasPrefix(secret, "sk_live_")) || len(key) != 32 {
		return nil, errors.New("valid Clerk keys and SESSION_SECRET are required")
	}
	if strings.HasPrefix(publishable, "pk_test_") != strings.HasPrefix(secret, "sk_test_") {
		return nil, errors.New("Clerk publishable and secret key environments must match")
	}
	encoded := strings.SplitN(publishable, "_", 3)
	domain, err := base64.RawStdEncoding.DecodeString(encoded[2])
	if err != nil {
		return nil, errors.New("invalid Clerk publishable key")
	}
	host := strings.TrimSuffix(string(domain), "$")
	if host == "" || strings.ContainsAny(host, "/:?#\\ \r\n") {
		return nil, errors.New("invalid Clerk domain")
	}
	allowed := map[string]bool{}
	for _, party := range parties {
		u, err := url.Parse(strings.TrimSpace(party))
		if err != nil || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") {
			return nil, errors.New("CLERK_AUTHORIZED_PARTIES must contain exact origins")
		}
		allowed[u.String()] = true
	}
	if len(allowed) == 0 {
		return nil, errors.New("CLERK_AUTHORIZED_PARTIES is required")
	}
	config := &clerk.ClientConfig{}
	config.Key = clerk.String(secret)
	config.HTTPClient = &http.Client{Timeout: 4 * time.Second}
	g := &ClerkGate{Store: store, Provider: &sdkProvider{session.NewClient(config), user.NewClient(config)}, PublishableKey: publishable, Issuer: "https://" + host, key: append([]byte(nil), key...)}
	g.LookupUser = g.Provider.(*sdkProvider).LookupUser
	g.verify = clerkhttp.WithHeaderAuthorization(
		func(p *clerkhttp.AuthorizationParams) error { p.JWKSClient = jwks.NewClient(config); return nil },
		clerkhttp.AuthorizationJWTExtractor(clerkToken),
		clerkhttp.AuthorizedParty(func(value string) bool { return allowed[value] }),
		clerkhttp.CustomClaimsConstructor(func(context.Context) any { return &clerkFactors{} }),
		clerkhttp.AuthorizationFailureHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, 401, "unauthenticated", "Sesi Clerk tidak valid. Silakan masuk kembali.")
		})),
	)
	return g, nil
}
func clerkToken(r *http.Request) string {
	if header := r.Header.Get("Authorization"); header != "" {
		if strings.HasPrefix(header, "Bearer ") {
			return strings.TrimPrefix(header, "Bearer ")
		}
		return "invalid"
	}
	if c, err := r.Cookie("__session"); err == nil {
		return c.Value
	}
	return ""
}

func (g *ClerkGate) Wrap(next http.Handler) http.Handler {
	protected := g.verify(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { g.authorize(w, r, next) }))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/health", "/api/v1/readyz", "/api/v1/auth/csrf", "/api/v1/auth/config":
			next.ServeHTTP(w, r)
			return
		case "/api/v1/auth/login", "/api/v1/auth/register":
			writeError(w, 404, "clerk_required", "Gunakan login Clerk.")
			return
		}
		protected.ServeHTTP(w, r)
	})
}
func (g *ClerkGate) authorize(w http.ResponseWriter, r *http.Request, next http.Handler) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok || claims.Subject == "" || claims.SessionID == "" || claims.Issuer != g.Issuer || claims.Expiry == nil || *claims.Expiry <= time.Now().Unix() {
		writeError(w, 401, "unauthenticated", "Silakan masuk melalui Clerk.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	identity, err := g.Provider.Inspect(ctx, claims.SessionID, claims.Subject)
	if err != nil {
		cancel()
		g.failure(w, err)
		return
	}
	uid, role, err := g.resolve(ctx, identity)
	if err != nil {
		cancel()
		g.failure(w, err)
		return
	}
	factors, _ := claims.Custom.(*clerkFactors)
	hasSecondFactor := identity.MFAEnabled && factors != nil && len(factors.FVA) == 2 && factors.FVA[1] >= 0
	needsMFA := role != "patient" && !hasSecondFactor
	accountPath := r.URL.Path == "/api/v1/auth/status" || r.URL.Path == "/api/v1/auth/logout"
	if needsMFA && !accountPath {
		cancel()
		writeError(w, 403, "mfa_required", "Aktifkan MFA dan masuk dengan faktor kedua sebelum mengakses ruang staf.")
		return
	}
	mac := hmac.New(sha256.New, g.key)
	mac.Write([]byte("hygeia-clerk-session\x00" + claims.Subject + "\x00" + claims.SessionID))
	internal := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	hash := sha256.Sum256([]byte(internal))
	_, err = g.Store.DB.Exec(ctx, `INSERT INTO sessions(token_hash,user_id,expires_at,clerk_session_id) VALUES($1,$2,$3,$4) ON CONFLICT(token_hash) DO UPDATE SET expires_at=EXCLUDED.expires_at`, hash[:], uid, identity.ExpiresAt, claims.SessionID)
	cancel()
	if err != nil {
		g.failure(w, err)
		return
	}
	// Fresh request-local validation is also used by long-lived SSE polls and
	// domain rechecks after waiting for a lock. No stale cross-request cache.
	var mu sync.Mutex
	last := time.Now()
	check := func(ctx context.Context) error {
		mu.Lock()
		defer mu.Unlock()
		if time.Since(last) < time.Second {
			return nil
		}
		current, err := g.Provider.Inspect(ctx, claims.SessionID, claims.Subject)
		if err != nil {
			return err
		}
		if role != "patient" && !accountPath && !current.MFAEnabled {
			return pgx.ErrNoRows
		}
		last = time.Now()
		return nil
	}
	request := r.Clone(context.WithValue(r.Context(), clerkContextKey{}, &clerkRequest{check, claims.SessionID, needsMFA}))
	request.Header.Del("Cookie")
	for _, c := range r.Cookies() {
		if c.Name != sessionCookie {
			request.AddCookie(c)
		}
	}
	request.AddCookie(&http.Cookie{Name: sessionCookie, Value: internal})
	next.ServeHTTP(w, request)
}
func (g *ClerkGate) failure(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		writeError(w, 401, "unauthenticated", "Sesi tidak aktif atau email belum terverifikasi.")
	case errors.Is(err, ErrClerkIdentityConflict):
		writeError(w, 409, "identity_link_required", "Akun Hygeia sudah ada. Minta pengelola menautkan identitas Clerk Anda.")
	default:
		writeError(w, 503, "unavailable", "Layanan autentikasi belum tersedia. Coba lagi.")
	}
}
func (g *ClerkGate) resolve(ctx context.Context, identity ClerkIdentity) (string, string, error) {
	tx, err := g.Store.DB.Begin(ctx)
	if err != nil {
		return "", "", err
	}
	defer tx.Rollback(ctx)
	var id, role string
	var disabled *time.Time
	err = tx.QueryRow(ctx, `SELECT id::text,role,disabled_at FROM users WHERE clerk_user_id=$1 FOR SHARE`, identity.UserID).Scan(&id, &role, &disabled)
	if err == nil {
		if disabled != nil {
			return "", "", pgx.ErrNoRows
		}
		return id, role, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", "", err
	}
	raw := make([]byte, 12)
	if _, err = rand.Read(raw); err != nil {
		return "", "", err
	}
	// A Clerk email never automatically claims an existing local account or role.
	err = tx.QueryRow(ctx, `INSERT INTO users(email,password_hash,full_name,role,clerk_user_id) VALUES($1,'!clerk',$2,'patient',$3) ON CONFLICT(clerk_user_id) DO UPDATE SET clerk_user_id=EXCLUDED.clerk_user_id RETURNING id::text,role,disabled_at`, identity.Email, identity.Name, identity.UserID).Scan(&id, &role, &disabled)
	if err != nil {
		var p *pgconn.PgError
		if errors.As(err, &p) && p.Code == "23505" {
			return "", "", ErrClerkIdentityConflict
		}
		return "", "", err
	}
	if disabled != nil {
		return "", "", pgx.ErrNoRows
	}
	if role == "patient" {
		_, err = tx.Exec(ctx, `INSERT INTO patients(user_id,patient_ref) VALUES($1,$2) ON CONFLICT(user_id) DO NOTHING`, id, "HYG-"+strings.ToUpper(hex.EncodeToString(raw)))
		if err != nil {
			return "", "", err
		}
	}
	return id, role, tx.Commit(ctx)
}
