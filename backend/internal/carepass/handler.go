package carepass

import (
	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"net/http"
)

type Handler struct{ Service *Service }

func (h *Handler) Register(m *http.ServeMux) {
	m.HandleFunc("POST /api/v1/carepasses", h.create)
	m.HandleFunc("GET /api/v1/carepasses", h.list)
	m.HandleFunc("POST /api/v1/carepasses/redeem", h.redeem)
	m.HandleFunc("PATCH /api/v1/carepasses/{id}", h.revoke)
}
func response(w http.ResponseWriter, r *http.Request, value any, err error, status int) {
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, status, value)
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		TTL          int    `json:"ttl_minutes"`
	}
	if !apiutil.Decode(w, r, &in) {
		return
	}
	v, err := h.Service.Create(r.Context(), auth.SessionHash(r), in.ResourceType, in.ResourceID, in.TTL)
	response(w, r, v, err, 201)
}
func (h *Handler) redeem(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Token string `json:"token"`
	}
	if !apiutil.Decode(w, r, &in) {
		return
	}
	v, err := h.Service.Redeem(r.Context(), auth.SessionHash(r), in.Token)
	response(w, r, map[string]any{"carepass": v}, err, 200)
}
func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Action string `json:"action"`
	}
	if !apiutil.Decode(w, r, &in) {
		return
	}
	if in.Action != "revoke" {
		apiutil.Failure(w, r, apiutil.ErrInvalid)
		return
	}
	v, err := h.Service.Revoke(r.Context(), auth.SessionHash(r), r.PathValue("id"))
	response(w, r, map[string]any{"carepass": v}, err, 200)
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q, err := apiutil.Query(r, "limit", "offset")
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	l, o, err := apiutil.Pagination(q)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	v, err := h.Service.List(r.Context(), auth.SessionHash(r), l, o)
	response(w, r, map[string]any{"carepasses": v}, err, 200)
}
