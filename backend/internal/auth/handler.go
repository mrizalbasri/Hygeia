package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/csrf"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/time/rate"
)

const sessionCookie = "hygeia_session"
const sessionTTL = 8 * time.Hour

type loginLimit struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type Handler struct {
	// Explicit frontend hosts permitted to submit CSRF-protected API requests.
	CSRFTrustedOrigins []string
	store              Store
	secure             bool
	dummyHash          string
	mu                 sync.Mutex
	limits             map[string]*loginLimit
	trustedProxies     []netip.Prefix
	external           *ClerkGate
}

func (h *Handler) UseClerk(gate *ClerkGate) { h.external = gate }

func NewHandler(store Store, secure bool, trustedProxies ...netip.Prefix) (*Handler, error) {
	dummy, err := HashPassword("hygeia-dummy-password-never-a-user")
	if err != nil {
		return nil, err
	}
	return &Handler{store: store, secure: secure, dummyHash: dummy, limits: make(map[string]*loginLimit), trustedProxies: append([]netip.Prefix(nil), trustedProxies...)}, nil
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/auth/config", func(w http.ResponseWriter, r *http.Request) {
		if h.external != nil {
			writeJSON(w, 200, map[string]string{"provider": "clerk", "publishable_key": h.external.PublishableKey})
			return
		}
		writeJSON(w, 200, map[string]string{"provider": "local"})
	})
	mux.HandleFunc("GET /api/v1/auth/status", func(w http.ResponseWriter, r *http.Request) {
		u, err := h.store.UserBySession(r.Context(), SessionHash(r))
		if err != nil {
			if h.external != nil {
				h.external.failure(w, err)
			} else {
				writeError(w, 401, "unauthenticated", "Silakan masuk.")
			}
			return
		}
		required := false
		if external, ok := r.Context().Value(clerkContextKey{}).(*clerkRequest); ok {
			required = external.requiresMFA
		}
		writeJSON(w, 200, map[string]any{"user": u, "requires_mfa": required})
	})
	mux.HandleFunc("GET /api/v1/auth/csrf", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"csrf_token": csrf.Token(r)})
	})
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/register", h.RegisterPatient)
	mux.HandleFunc("POST /api/v1/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/v1/me", h.Me)
}

func Protect(next http.Handler, key []byte, secure bool, trustedOrigins ...string) http.Handler {
	protected := csrf.Protect(key,
		csrf.CookieName("hygeia_csrf"), csrf.Path("/"), csrf.Secure(secure),
		csrf.HttpOnly(true), csrf.SameSite(csrf.SameSiteLaxMode),
		csrf.TrustedOrigins(trustedOrigins),
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeError(w, http.StatusForbidden, "csrf_invalid", "Permintaan tidak valid. Muat ulang halaman dan coba lagi.")
		})),
	)(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if !secure {
			r = csrf.PlaintextHTTPRequest(r)
		}
		protected.ServeHTTP(w, r)
	})
}

func (h *Handler) allowLogin(remoteAddr string) bool {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}
	now := time.Now()
	h.mu.Lock()
	defer h.mu.Unlock()
	for addr, limit := range h.limits {
		if now.Sub(limit.lastSeen) > 10*time.Minute {
			delete(h.limits, addr)
		}
	}
	limit := h.limits[ip]
	if limit == nil {
		if len(h.limits) >= 10000 {
			return false
		}
		limit = &loginLimit{limiter: rate.NewLimiter(rate.Every(12*time.Second), 5)}
		h.limits[ip] = limit
	}
	limit.lastSeen = now
	return limit.limiter.Allow()
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.allowLogin(h.clientIP(r)) {
		w.Header().Set("Retry-After", "12")
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Terlalu banyak percobaan. Tunggu sebentar lalu coba lagi.")
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Gunakan application/json.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Isi email dan password dengan benar.")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_input", "Isi email dan password dengan benar.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if len(email) > 254 || len(email) < 3 || input.Password == "" || len(input.Password) > 1024 {
		writeError(w, http.StatusBadRequest, "invalid_input", "Isi email dan password dengan benar.")
		return
	}
	id, hash, err := h.store.Credentials(r.Context(), email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusServiceUnavailable, "unavailable", "Layanan sedang tidak tersedia.")
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		hash = h.dummyHash
	}
	valid := VerifyPassword(input.Password, hash)
	if !valid || id == "" {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Email atau password tidak sesuai.")
		return
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		writeError(w, 500, "internal_error", "Tidak dapat membuat sesi.")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	tokenHash := sha256.Sum256([]byte(token))
	expires := time.Now().Add(sessionTTL)
	if err := h.store.CreateSession(r.Context(), tokenHash[:], id, expires); err != nil {
		writeError(w, 503, "unavailable", "Tidak dapat membuat sesi.")
		return
	}
	user, err := h.store.UserBySession(r.Context(), tokenHash[:])
	if err != nil {
		_ = h.store.DeleteSession(r.Context(), tokenHash[:])
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, 401, "invalid_credentials", "Email atau password tidak sesuai.")
		} else {
			writeError(w, 503, "unavailable", "Layanan sedang tidak tersedia.")
		}
		return
	}
	// Re-authentication replaces the browser's previous session rather than reusing it.
	if old, err := r.Cookie(sessionCookie); err == nil {
		oldHash := sha256.Sum256([]byte(old.Value))
		if err := h.store.DeleteSession(r.Context(), oldHash[:]); err != nil {
			_ = h.store.DeleteSession(r.Context(), tokenHash[:])
			writeError(w, 503, "unavailable", "Tidak dapat mengganti sesi.")
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, MaxAge: int(sessionTTL.Seconds()), Expires: expires})
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (h *Handler) RegisterPatient(w http.ResponseWriter, r *http.Request) {
	if !h.allowLogin(h.clientIP(r)) {
		w.Header().Set("Retry-After", "12")
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Terlalu banyak percobaan. Tunggu sebentar lalu coba lagi.")
		return
	}
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "invalid_content_type", "Gunakan application/json.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	var input struct {
		Email     string `json:"email"`
		Password  string `json:"password"`
		FullName  string `json:"full_name"`
		BirthYear int    `json:"birth_year"`
		Sex       string `json:"sex"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Isi semua kolom pendaftaran dengan benar.")
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid_input", "Isi semua kolom pendaftaran dengan benar.")
		return
	}
	email := strings.ToLower(strings.TrimSpace(input.Email))
	name := strings.TrimSpace(input.FullName)
	address, emailErr := mail.ParseAddress(email)
	if emailErr != nil || address.Address != email || len(email) > 254 || len(name) < 1 || len(name) > 200 {
		writeError(w, http.StatusBadRequest, "invalid_input", "Email atau nama tidak valid.")
		return
	}
	if input.Sex != "female" && input.Sex != "male" && input.Sex != "other" && input.Sex != "unknown" {
		writeError(w, http.StatusBadRequest, "invalid_input", "Jenis kelamin harus female, male, other, atau unknown.")
		return
	}
	if input.BirthYear < 1900 || input.BirthYear > time.Now().Year() {
		writeError(w, http.StatusBadRequest, "invalid_input", "Tahun lahir tidak valid.")
		return
	}
	hash, err := HashPassword(input.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input", "Password harus 12 sampai 1024 karakter.")
		return
	}
	// A database uniqueness constraint also protects the random patient reference.
	refBytes := make([]byte, 4)
	if _, err := rand.Read(refBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Tidak dapat membuat referensi pasien.")
		return
	}
	patientRef := fmt.Sprintf("HYG-%04X-%04X",
		uint16(refBytes[0])<<8|uint16(refBytes[1]),
		uint16(refBytes[2])<<8|uint16(refBytes[3]))

	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Tidak dapat membuat sesi.")
		return
	}
	token := base64.RawURLEncoding.EncodeToString(random)
	tokenHash := sha256.Sum256([]byte(token))
	expires := time.Now().Add(sessionTTL)
	user, err := h.store.RegisterPatient(r.Context(), email, hash, name, input.BirthYear, input.Sex, patientRef, tokenHash[:], expires)
	if err != nil {
		var dbErr *pgconn.PgError
		if errors.As(err, &dbErr) && dbErr.Code == "23505" && dbErr.ConstraintName == "users_email_key" {
			writeError(w, http.StatusConflict, "email_taken", "Email sudah terdaftar.")
			return
		}
		writeError(w, http.StatusServiceUnavailable, "unavailable", "Layanan sedang tidak tersedia.")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, MaxAge: int(sessionTTL.Seconds()), Expires: expires})
	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil || len(cookie.Value) != 43 {
		writeError(w, 401, "unauthenticated", "Silakan masuk terlebih dahulu.")
		return
	}
	hash := sha256.Sum256([]byte(cookie.Value))
	user, err := h.store.UserBySession(r.Context(), hash[:])
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 401, "unauthenticated", "Sesi sudah berakhir. Silakan masuk kembali.")
		return
	}
	if err != nil {
		writeError(w, 503, "unavailable", "Layanan sedang tidak tersedia.")
		return
	}
	writeJSON(w, 200, map[string]any{"user": user})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if h.external != nil {
		if current, ok := r.Context().Value(clerkContextKey{}).(*clerkRequest); ok {
			if err := h.external.Provider.Revoke(r.Context(), current.sessionID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
				h.external.failure(w, err)
				return
			}
		}
	}
	if cookie, err := r.Cookie(sessionCookie); err == nil {
		hash := sha256.Sum256([]byte(cookie.Value))
		if err := h.store.DeleteSession(r.Context(), hash[:]); err != nil {
			writeError(w, 503, "unavailable", "Tidak dapat mengakhiri sesi.")
			return
		}
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: h.secure, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	w.WriteHeader(http.StatusNoContent)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": w.Header().Get("X-Request-ID")}})
}
