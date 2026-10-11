package consent

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

type Handler struct {
	Service *Service
	Logger  *slog.Logger
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var refPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/consents", h.create)
	mux.HandleFunc("GET /api/v1/consents", h.list)
	mux.HandleFunc("GET /api/v1/consents/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/consents/{id}", h.decide)
}
func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
func apiError(w http.ResponseWriter, status int, code, message string) {
	jsonResponse(w, status, map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": w.Header().Get("X-Request-ID")}})
}
func invalid(w http.ResponseWriter) { apiError(w, 400, "invalid_input", "Isi permintaan tidak valid.") }
func (h *Handler) respondError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrUnauthenticated):
		apiError(w, 401, "unauthenticated", "Silakan masuk terlebih dahulu.")
	case errors.Is(err, ErrForbidden):
		apiError(w, 403, "forbidden", "Akses tidak diizinkan.")
	case errors.Is(err, ErrConflict):
		apiError(w, 409, "invalid_state", "Status izin tidak mengizinkan tindakan ini.")
	default:
		if h.Logger != nil {
			h.Logger.Error("consent operation failed", "request_id", w.Header().Get("X-Request-ID"), "method", r.Method)
		}
		apiError(w, 503, "unavailable", "Layanan sedang tidak tersedia.")
	}
}
func decode(w http.ResponseWriter, r *http.Request, input any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		apiError(w, 415, "invalid_content_type", "Gunakan application/json.")
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(input); err != nil {
		invalid(w)
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		invalid(w)
		return false
	}
	return true
}
func resourceID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		invalid(w)
		return "", false
	}
	return strings.ToLower(id), true
}
func pagination(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	limit, offset := 20, 0
	var err error
	if r.URL.Query().Has("limit") {
		limit, err = strconv.Atoi(r.URL.Query().Get("limit"))
		if err != nil || limit < 1 || limit > 100 {
			invalid(w)
			return 0, 0, false
		}
	}
	if r.URL.Query().Has("offset") {
		offset, err = strconv.Atoi(r.URL.Query().Get("offset"))
		if err != nil || offset < 0 || offset > 10000 {
			invalid(w)
			return 0, 0, false
		}
	}
	return limit, offset, true
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if !decode(w, r, &input) {
		return
	}
	input.Purpose = strings.TrimSpace(input.Purpose)
	if input.ResourceType != "lab_result" || !uuidPattern.MatchString(input.ResourceID) || !refPattern.MatchString(input.PatientRef) || input.Purpose == "" || !utf8.ValidString(input.Purpose) || strings.ContainsRune(input.Purpose, 0) || utf8.RuneCountInString(input.Purpose) > 1000 {
		invalid(w)
		return
	}
	input.ResourceID = strings.ToLower(input.ResourceID)
	c, err := h.Service.Create(r.Context(), auth.SessionHash(r), input)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/consents/"+c.ID)
	jsonResponse(w, 201, map[string]any{"consent": c})
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	c, err := h.Service.Get(r.Context(), auth.SessionHash(r), id)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"consent": c})
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	status := r.URL.Query().Get("status")
	switch status {
	case "", "pending", "active", "denied", "expired", "revoked":
	default:
		invalid(w)
		return
	}
	consents, err := h.Service.List(r.Context(), auth.SessionHash(r), status, limit, offset)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"consents": consents, "limit": limit, "offset": offset})
}
func (h *Handler) decide(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	var input struct {
		Action string `json:"action"`
		Hours  int    `json:"duration_hours"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !(input.Action == "approve" && (input.Hours == 1 || input.Hours == 24 || input.Hours == 168) || (input.Action == "deny" || input.Action == "revoke") && input.Hours == 0) {
		invalid(w)
		return
	}
	c, err := h.Service.Decide(r.Context(), auth.SessionHash(r), id, input.Action, input.Hours)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"consent": c})
}
