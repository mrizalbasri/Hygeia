package notification

import (
	"context"
	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"net/http"
	"strconv"
	"sync"
)

type Handler struct {
	Service *Service
	// Shutdown cancels streams without interrupting ordinary JSON requests.
	Shutdown context.Context
	mu       sync.Mutex
	streams  map[string]int
	total    int
}

func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/v1/notifications", h.list)
	mux.HandleFunc("GET /api/v1/notifications/unread-count", h.unread)
	mux.HandleFunc("GET /api/v1/notifications/stream", h.stream)
	mux.HandleFunc("PATCH /api/v1/notifications/{id}", h.read)
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q, err := apiutil.Query(r, "limit", "before_id", "unread")
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	limit, _, err := apiutil.Pagination(q)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	before := int64(0)
	if q.Has("before_id") {
		before, err = strconv.ParseInt(q.Get("before_id"), 10, 64)
		if err != nil || before <= 0 {
			apiutil.Failure(w, r, apiutil.ErrInvalid)
			return
		}
	}
	unread := false
	if q.Has("unread") {
		if q.Get("unread") != "true" && q.Get("unread") != "false" {
			apiutil.Failure(w, r, apiutil.ErrInvalid)
			return
		}
		unread = q.Get("unread") == "true"
	}
	page, err := h.Service.List(r.Context(), auth.SessionHash(r), limit, before, unread)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, 200, page)
}
func (h *Handler) unread(w http.ResponseWriter, r *http.Request) {
	if _, err := apiutil.Query(r); err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	count, err := h.Service.Unread(r.Context(), auth.SessionHash(r))
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, 200, map[string]int64{"unread_count": count})
}
func (h *Handler) read(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		apiutil.Failure(w, r, apiutil.ErrInvalid)
		return
	}
	var in struct {
		Read bool `json:"read"`
	}
	if !apiutil.Decode(w, r, &in) {
		return
	}
	if !in.Read {
		apiutil.Failure(w, r, apiutil.ErrInvalid)
		return
	}
	n, err := h.Service.Read(r.Context(), auth.SessionHash(r), id)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	apiutil.JSON(w, 200, map[string]any{"notification": n})
}
