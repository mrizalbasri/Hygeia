package admin

import (
	"net/http"
	"strings"

	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

type Handler struct{ Service *Service }

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/admin/users", h.users)
	mux.HandleFunc("POST /api/v1/admin/users", h.createUser)
	mux.HandleFunc("PATCH /api/v1/admin/users/{id}", h.updateUser)
	mux.HandleFunc("GET /api/v1/admin/organizations", h.organizations)
	mux.HandleFunc("POST /api/v1/admin/organizations", h.createOrganization)
	mux.HandleFunc("PATCH /api/v1/admin/organizations/{id}", h.renameOrganization)
	mux.HandleFunc("GET /api/v1/admin/audit-chains", h.chains)
	mux.HandleFunc("GET /api/v1/admin/activity", h.activity)
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
func output(w http.ResponseWriter, r *http.Request, status int, key string, value any, err error) {
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, status, map[string]any{key: value})
}
func (h *Handler) users(w http.ResponseWriter, r *http.Request) {
	l, o, ok := page(w, r)
	if !ok {
		return
	}
	v, err := h.Service.Users(r.Context(), auth.SessionHash(r), l, o)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, 200, map[string]any{"users": v, "limit": l, "offset": o})
}
func (h *Handler) organizations(w http.ResponseWriter, r *http.Request) {
	l, o, ok := page(w, r)
	if !ok {
		return
	}
	v, err := h.Service.Organizations(r.Context(), auth.SessionHash(r), l, o)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, 200, map[string]any{"organizations": v, "limit": l, "offset": o})
}
func (h *Handler) chains(w http.ResponseWriter, r *http.Request) {
	l, o, ok := page(w, r)
	if !ok {
		return
	}
	v, err := h.Service.Chains(r.Context(), auth.SessionHash(r), l, o)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, 200, map[string]any{"chains": v, "limit": l, "offset": o})
}
func (h *Handler) activity(w http.ResponseWriter, r *http.Request) {
	l, o, ok := page(w, r)
	if !ok {
		return
	}
	v, err := h.Service.Activity(r.Context(), auth.SessionHash(r), l, o)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, 200, map[string]any{"activity": v, "limit": l, "offset": o})
}
func (h *Handler) createUser(w http.ResponseWriter, r *http.Request) {
	var in CreateAccount
	if !apiutil.Decode(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.FullName = strings.TrimSpace(in.FullName)
	if !ValidCreate(in) {
		apiutil.Failure(w, r, apiutil.ErrInvalid)
		return
	}
	v, err := h.Service.CreateUser(r.Context(), auth.SessionHash(r), in)
	if err == nil {
		w.Header().Set("Location", "/api/v1/admin/users/"+v.ID)
	}
	output(w, r, 201, "user", v, err)
}
func (h *Handler) updateUser(w http.ResponseWriter, r *http.Request) {
	var in UpdateAccount
	if !apiutil.Decode(w, r, &in) {
		return
	}
	if in.FullName != nil {
		v := strings.TrimSpace(*in.FullName)
		in.FullName = &v
	}
	id := strings.ToLower(r.PathValue("id"))
	if !apiutil.UUID(id) {
		apiutil.Failure(w, r, apiutil.ErrInvalid)
		return
	}
	v, err := h.Service.UpdateUser(r.Context(), auth.SessionHash(r), id, in)
	output(w, r, 200, "user", v, err)
}
func (h *Handler) createOrganization(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if !apiutil.Decode(w, r, &in) {
		return
	}
	v, err := h.Service.SaveOrganization(r.Context(), auth.SessionHash(r), "", strings.TrimSpace(in.Name), in.Type)
	output(w, r, 201, "organization", v, err)
}
func (h *Handler) renameOrganization(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name string `json:"name"`
	}
	if !apiutil.Decode(w, r, &in) {
		return
	}
	v, err := h.Service.SaveOrganization(r.Context(), auth.SessionHash(r), strings.ToLower(r.PathValue("id")), strings.TrimSpace(in.Name), "")
	output(w, r, 200, "organization", v, err)
}
