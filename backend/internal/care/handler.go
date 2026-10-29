package care

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/timeline"
)

type Handler struct {
	Service *Service
	Logger  *slog.Logger
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
var refPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/v1/episodes", h.invite)
	mux.HandleFunc("GET /api/v1/episodes", h.list)
	mux.HandleFunc("GET /api/v1/episodes/{id}", h.get)
	mux.HandleFunc("PATCH /api/v1/episodes/{id}", h.transition)
	mux.HandleFunc("GET /api/v1/patient/access-summary", h.summary)
	mux.HandleFunc("GET /api/v1/patient/work-summary", h.workSummary)
	mux.HandleFunc("GET /api/v1/patient/timeline", h.timeline)
	mux.HandleFunc("GET /api/v1/audit/verify", h.verify)
}

func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func apiError(w http.ResponseWriter, status int, code, message string) {
	jsonResponse(w, status, map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": w.Header().Get("X-Request-ID")}})
}

func (h *Handler) respondError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrUnauthenticated):
		apiError(w, 401, "unauthenticated", "Silakan masuk terlebih dahulu.")
	case errors.Is(err, ErrForbidden):
		apiError(w, 403, "forbidden", "Akses tidak diizinkan.")
	case errors.Is(err, ErrConflict):
		apiError(w, 409, "invalid_state", "Status episode tidak mengizinkan tindakan ini.")
	case errors.Is(err, timeline.ErrInvalidOptions):
		apiError(w, 400, "invalid_input", "Parameter timeline tidak valid.")
	default:
		// Avoid database errors containing query parameters or connection secrets.
		if h.Logger != nil {
			h.Logger.Error("care operation failed", "request_id", w.Header().Get("X-Request-ID"), "method", r.Method)
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
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(input); err != nil {
		apiError(w, 400, "invalid_input", "Isi permintaan tidak valid.")
		return false
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		apiError(w, 400, "invalid_input", "Isi permintaan tidak valid.")
		return false
	}
	return true
}

func episodeID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := r.PathValue("id")
	if !uuidPattern.MatchString(id) {
		apiError(w, 400, "invalid_input", "ID episode tidak valid.")
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
			apiError(w, 400, "invalid_input", "Limit harus antara 1 dan 100.")
			return 0, 0, false
		}
	}
	if r.URL.Query().Has("offset") {
		offset, err = strconv.Atoi(r.URL.Query().Get("offset"))
		if err != nil || offset < 0 || offset > 10000 {
			apiError(w, 400, "invalid_input", "Offset harus antara 0 dan 10000.")
			return 0, 0, false
		}
	}
	return limit, offset, true
}

func (h *Handler) invite(w http.ResponseWriter, r *http.Request) {
	var input struct {
		PatientRef string `json:"patient_ref"`
	}
	if !decode(w, r, &input) {
		return
	}
	if !refPattern.MatchString(input.PatientRef) {
		apiError(w, 400, "invalid_input", "Kode pasien tidak valid.")
		return
	}
	e, err := h.Service.Invite(r.Context(), auth.SessionHash(r), input.PatientRef)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	w.Header().Set("Location", "/api/v1/episodes/"+e.ID)
	jsonResponse(w, 201, map[string]any{"episode": e})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := episodeID(w, r)
	if !ok {
		return
	}
	e, err := h.Service.Get(r.Context(), auth.SessionHash(r), id)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"episode": e})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	status := r.URL.Query().Get("status")
	if status != "" && status != "pending" && status != "active" && status != "rejected" && status != "ended" {
		apiError(w, 400, "invalid_input", "Status tidak valid.")
		return
	}
	episodes, err := h.Service.List(r.Context(), auth.SessionHash(r), status, limit, offset)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"episodes": episodes, "limit": limit, "offset": offset})
}

func (h *Handler) transition(w http.ResponseWriter, r *http.Request) {
	id, ok := episodeID(w, r)
	if !ok {
		return
	}
	var input struct {
		Action string `json:"action"`
	}
	if !decode(w, r, &input) {
		return
	}
	if input.Action != "accept" && input.Action != "reject" && input.Action != "end" {
		apiError(w, 400, "invalid_input", "Action harus accept, reject, atau end.")
		return
	}
	e, err := h.Service.Transition(r.Context(), auth.SessionHash(r), id, input.Action)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"episode": e})
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	limit, offset, ok := pagination(w, r)
	if !ok {
		return
	}
	events, err := h.Service.AccessSummary(r.Context(), auth.SessionHash(r), limit, offset)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"events": events, "limit": limit, "offset": offset})
}

func timelineOptions(r *http.Request) (timeline.Options, bool) {
	options := timeline.Options{Limit: 20}
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return options, false
	}
	for key, values := range query {
		if len(values) != 1 {
			return options, false
		}
		switch key {
		case "kind":
			options.Kind = values[0]
		case "limit":
			options.Limit, err = strconv.Atoi(values[0])
			if err != nil {
				return options, false
			}
		case "before_seq":
			options.BeforeSeq, err = strconv.ParseInt(values[0], 10, 64)
			if err != nil || options.BeforeSeq <= 0 {
				return options, false
			}
		default:
			// Patient identity comes from the session, never a query selector.
			return options, false
		}
	}
	return options, options.Valid()
}

func (h *Handler) timeline(w http.ResponseWriter, r *http.Request) {
	options, ok := timelineOptions(r)
	if !ok {
		apiError(w, 400, "invalid_input", "Parameter timeline tidak valid.")
		return
	}
	page, err := h.Service.Timeline(r.Context(), auth.SessionHash(r), options)
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, page)
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	chain := r.URL.Query().Get("chain")
	if chain != "" && !uuidPattern.MatchString(chain) {
		apiError(w, 400, "invalid_input", "ID rantai tidak valid.")
		return
	}
	result, err := h.Service.Verify(r.Context(), auth.SessionHash(r), strings.ToLower(chain))
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, result)
}
