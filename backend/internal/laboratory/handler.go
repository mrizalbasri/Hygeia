package laboratory

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
var testCodePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,63}$`)

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/laboratories", h.laboratories)
	mux.HandleFunc("POST /api/v1/lab-requests", h.create)
	mux.HandleFunc("GET /api/v1/lab-requests", h.list)
	mux.HandleFunc("GET /api/v1/lab-requests/{id}", h.get)
	mux.HandleFunc("POST /api/v1/lab-requests/{id}/approval", h.approval)
	mux.HandleFunc("PATCH /api/v1/lab-requests/{id}/status", h.status)
	mux.HandleFunc("POST /api/v1/lab-requests/{id}/result", h.submit)
	mux.HandleFunc("GET /api/v1/lab-results/{id}", h.result)
	mux.HandleFunc("POST /api/v1/lab-results/{id}/corrections", h.correct)
	mux.HandleFunc("GET /api/v1/lab-receipts", h.receipts)
	mux.HandleFunc("POST /api/v1/lab-requests/{id}/cancel", h.cancel)
}

func (h *Handler) receipts(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	receipts, err := h.Service.Receipts(r.Context(), auth.SessionHash(r), limit, offset)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"receipts": receipts, "limit": limit, "offset": offset})
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	var input struct {
		Reason string `json:"reason"`
	}
	if !decode(w, r, &input) {
		return
	}
	input.Reason = strings.TrimSpace(input.Reason)
	if !validText(input.Reason, 5, 500) {
		invalid(w)
		return
	}
	if err := h.Service.Cancel(r.Context(), auth.SessionHash(r), id, input.Reason); err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]string{"status": "cancelled"})
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
		apiError(w, 409, "invalid_state", "Status permintaan lab tidak mengizinkan tindakan ini.")
	default:
		if h.Logger != nil {
			h.Logger.Error("laboratory operation failed", "request_id", w.Header().Get("X-Request-ID"), "method", r.Method)
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
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16384))
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
	case "", "pending_patient_approval", "sent", "accepted", "in_progress", "completed", "cancelled":
		return true
	default:
		return false
	}
}
func validResult(input ResultInput) bool {
	if !validText(input.Value, 1, 500) || !validText(input.Unit, 0, 100) || !validText(input.ReferenceRange, 0, 200) || !validText(input.Note, 0, 2000) {
		return false
	}
	switch input.Flag {
	case "normal", "high", "low", "abnormal", "unknown":
		return true
	default:
		return false
	}
}

func (h *Handler) laboratories(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	labs, err := h.Service.Laboratories(r.Context(), auth.SessionHash(r), limit, offset)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"laboratories": labs, "limit": limit, "offset": offset})
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input CreateInput
	if !decode(w, r, &input) {
		return
	}
	input.TestName = strings.TrimSpace(input.TestName)
	if !uuidPattern.MatchString(input.EpisodeID) || !uuidPattern.MatchString(input.LabOrgID) || !testCodePattern.MatchString(input.TestCode) || !validText(input.TestName, 1, 200) {
		invalid(w)
		return
	}
	input.EpisodeID = strings.ToLower(input.EpisodeID)
	input.LabOrgID = strings.ToLower(input.LabOrgID)
	request, err := h.Service.Create(r.Context(), auth.SessionHash(r), input)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/lab-requests/"+request.ID)
	jsonResponse(w, 201, map[string]any{"lab_request": request})
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	request, err := h.Service.Get(r.Context(), auth.SessionHash(r), id)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"lab_request": request})
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
	requests, err := h.Service.List(r.Context(), auth.SessionHash(r), status, limit, offset)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"lab_requests": requests, "limit": limit, "offset": offset})
}
func (h *Handler) approval(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	var input struct {
		Decision string `json:"decision"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Decision != "approve" && input.Decision != "reject" {
		invalid(w)
		return
	}
	request, err := h.Service.Approve(r.Context(), auth.SessionHash(r), id, input.Decision)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"lab_request": request})
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
	if input.Status != "accepted" && input.Status != "in_progress" {
		invalid(w)
		return
	}
	request, err := h.Service.UpdateStatus(r.Context(), auth.SessionHash(r), id, input.Status)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"lab_request": request})
}
func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	var input ResultInput
	if !decode(w, r, &input) {
		return
	}
	if !validResult(input) {
		invalid(w)
		return
	}
	receipt, err := h.Service.SubmitResult(r.Context(), auth.SessionHash(r), id, input)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/lab-results/"+receipt.ID)
	jsonResponse(w, 201, map[string]any{"result": receipt})
}
func (h *Handler) result(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	result, err := h.Service.GetResult(r.Context(), auth.SessionHash(r), id)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"result": result})
}

func (h *Handler) correct(w http.ResponseWriter, r *http.Request) {
	id, ok := resourceID(w, r)
	if !ok {
		return
	}
	var input CorrectionInput
	if !decode(w, r, &input) {
		return
	}
	input.Reason = strings.TrimSpace(input.Reason)
	if length := utf8.RuneCountInString(input.Reason); length < 5 || length > 500 {
		apiError(w, 400, "invalid_input", "Alasan koreksi harus antara 5 dan 500 karakter.")
		return
	}
	if !validResult(ResultInput{Value: input.Value, Unit: input.Unit, ReferenceRange: input.ReferenceRange, Flag: input.Flag, Note: input.Note}) {
		invalid(w)
		return
	}
	corr, err := h.Service.CorrectResult(r.Context(), auth.SessionHash(r), id, input)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 201, map[string]any{"correction": corr})
}
