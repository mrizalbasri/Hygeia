package breakglass

import (
	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"net/http"
)

type Handler struct{ Service *Service }

func (h *Handler) Register(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/break-glass", h.start)
	m.HandleFunc("GET /api/v1/break-glass", h.list)
	m.HandleFunc("GET /api/v1/break-glass/{id}/resources", h.resources)
	m.HandleFunc("GET /api/v1/admin/break-glass", h.list)
	m.HandleFunc("PATCH /api/v1/admin/break-glass/{id}", h.review)
}
func respond(w http.ResponseWriter, r *http.Request, key string, value any, err error, status int) {
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, status, map[string]any{key: value})
}
func (h *Handler) start(w http.ResponseWriter, r *http.Request) {
	var in struct {
		PatientRef string `json:"patient_ref"`
		ReasonCode string `json:"reason_code"`
		ReasonText string `json:"reason_text"`
	}
	if !apiutil.Decode(w, r, &in) {
		return
	}
	g, err := h.Service.Start(r.Context(), auth.SessionHash(r), in.PatientRef, in.ReasonCode, in.ReasonText)
	respond(w, r, "session", g, err, 201)
}
func page(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	q, err := apiutil.Query(r, "limit", "offset")
	if err != nil {
		apiutil.Failure(w, r, err)
		return 0, 0, false
	}
	l, o, err := apiutil.Pagination(q)
	if err != nil {
		apiutil.Failure(w, r, err)
		return 0, 0, false
	}
	return l, o, true
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	l, o, ok := page(w, r)
	if !ok {
		return
	}
	g, err := h.Service.List(r.Context(), auth.SessionHash(r), r.URL.Path == "/api/v1/admin/break-glass", l, o)
	respond(w, r, "sessions", g, err, 200)
}
func (h *Handler) resources(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !apiutil.UUID(id) {
		apiutil.Failure(w, r, apiutil.ErrInvalid)
		return
	}
	l, o, ok := page(w, r)
	if !ok {
		return
	}
	g, err := h.Service.Resources(r.Context(), auth.SessionHash(r), id, l, o)
	respond(w, r, "resources", g, err, 200)
}
func (h *Handler) review(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Decision string `json:"decision"`
		Note     string `json:"note"`
	}
	if !apiutil.Decode(w, r, &in) {
		return
	}
	g, err := h.Service.Review(r.Context(), auth.SessionHash(r), r.PathValue("id"), in.Decision, in.Note)
	respond(w, r, "session", g, err, 200)
}
