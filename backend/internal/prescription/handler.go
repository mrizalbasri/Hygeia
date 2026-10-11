package prescription

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

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/pharmacies", h.pharmacies)
	mux.HandleFunc("POST /api/v1/prescriptions", h.create)
	mux.HandleFunc("GET /api/v1/prescriptions", h.list)
	mux.HandleFunc("GET /api/v1/prescriptions/{id}", h.get)
	mux.HandleFunc("POST /api/v1/prescriptions/{id}/approval", h.approval)
	mux.HandleFunc("PATCH /api/v1/prescriptions/{id}/status", h.status)
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
		apiError(w, 409, "invalid_state", "Status resep tidak mengizinkan tindakan ini.")
	default:
		if h.Logger != nil {
			h.Logger.Error("prescription operation failed", "request_id", w.Header().Get("X-Request-ID"), "method", r.Method)
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
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
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
func validText(value string, min, max int) bool {
	return utf8.ValidString(value) && !strings.ContainsRune(value, 0) && utf8.RuneCountInString(strings.TrimSpace(value)) >= min && utf8.RuneCountInString(value) <= max
}

func validStatus(status string) bool {
	switch status {
	case "", "pending_patient_approval", "sent", "accepted", "preparing", "ready", "completed", "cancelled":
		return true
	default:
		return false
	}
}

func validCreate(input CreateInput) bool {
	if !uuidPattern.MatchString(input.EpisodeID) || len(input.Items) < 1 || len(input.Items) > 20 {
		return false
	}
	for _, item := range input.Items {
		if !validText(item.MedicationName, 1, 200) || !validText(item.Dosage, 1, 200) || !validText(item.Instructions, 1, 1000) || item.Quantity < 1 || item.Quantity > 10000 {
			return false
		}
	}
	return true
}

func (h *Handler) pharmacies(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	providers, err := h.Service.Pharmacies(r.Context(), auth.SessionHash(r), limit, offset)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"pharmacies": providers, "limit": limit, "offset": offset})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if !decode(w, r, &input) {
		return
	}
	for i := range input.Items {
		input.Items[i].MedicationName = strings.TrimSpace(input.Items[i].MedicationName)
		input.Items[i].Dosage = strings.TrimSpace(input.Items[i].Dosage)
		input.Items[i].Instructions = strings.TrimSpace(input.Items[i].Instructions)
	}
	if !validCreate(input) {
		invalid(w)
		return
	}
	input.EpisodeID = strings.ToLower(input.EpisodeID)
	p, err := h.Service.Create(r.Context(), auth.SessionHash(r), input)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/prescriptions/"+p.ID)
	jsonResponse(w, 201, map[string]any{"prescription": p})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	p, err := h.Service.Get(r.Context(), auth.SessionHash(r), id)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"prescription": p})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	status := r.URL.Query().Get("status")
	if !validStatus(status) {
		invalid(w)
		return
	}
	p, err := h.Service.List(r.Context(), auth.SessionHash(r), status, limit, offset)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"prescriptions": p, "limit": limit, "offset": offset})
}

func (h *Handler) approval(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	var input ApprovalInput
	if !decode(w, r, &input) {
		return
	}
	if (input.Decision != "approve" && input.Decision != "reject") ||
		(input.Decision == "approve" && !uuidPattern.MatchString(input.PharmacyOrgID)) ||
		(input.Decision == "reject" && input.PharmacyOrgID != "") {
		invalid(w)
		return
	}
	input.PharmacyOrgID = strings.ToLower(input.PharmacyOrgID)
	p, err := h.Service.Approve(r.Context(), auth.SessionHash(r), id, input)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"prescription": p})
}

func (h *Handler) status(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	var input struct {
		Status string `json:"status"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Status != "accepted" && input.Status != "preparing" && input.Status != "ready" && input.Status != "completed" {
		invalid(w)
		return
	}
	p, err := h.Service.UpdateStatus(r.Context(), auth.SessionHash(r), id, input.Status)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"prescription": p})
}
