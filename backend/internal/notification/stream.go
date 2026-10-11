package notification

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"net/http"
	"strconv"
	"time"
)

type batch struct {
	UserID        string
	Notifications []Notification
}

const streamBatchSize = 50

// Each poll reauthenticates from PostgreSQL. No database connection is held
// while waiting on a client or between polls.
func (s *Service) since(ctx context.Context, token []byte, after int64) (batch, error) {
	b := batch{Notifications: make([]Notification, 0)}
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, a auth.User) error {
		b.UserID = a.ID
		rows, err := tx.Query(ctx, `SELECT `+columns+` FROM notifications WHERE user_id=$1 AND id>$2 ORDER BY id LIMIT $3`, a.ID, after, streamBatchSize)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			n, err := scan(rows)
			if err != nil {
				return err
			}
			b.Notifications = append(b.Notifications, n)
		}
		return rows.Err()
	})
	return b, err
}
func (h *Handler) acquire(user string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.total >= 128 || h.streams[user] >= 2 {
		return false
	}
	if h.streams == nil {
		h.streams = make(map[string]int)
	}
	h.streams[user]++
	h.total++
	return true
}
func (h *Handler) release(user string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.streams[user]--
	if h.streams[user] == 0 {
		delete(h.streams, user)
	}
	h.total--
}
func cursor(r *http.Request) (int64, error) {
	q, err := apiutil.Query(r, "after_id")
	if err != nil {
		return 0, err
	}
	value := q.Get("after_id")
	if r.Header.Get("Last-Event-ID") != "" {
		value = r.Header.Get("Last-Event-ID")
	}
	if value == "" {
		if q.Has("after_id") {
			return 0, apiutil.ErrInvalid
		}
		return 0, nil
	}
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil || n < 0 {
		return 0, apiutil.ErrInvalid
	}
	return n, nil
}
func (h *Handler) stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.Header().Set("Allow", "GET")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	after, err := cursor(r)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	token := auth.SessionHash(r)
	b, err := h.Service.since(r.Context(), token, after)
	if err != nil {
		apiutil.Failure(w, r, err)
		return
	}
	if !h.acquire(b.UserID) {
		w.Header().Set("Retry-After", "5")
		apiutil.JSON(w, 429, map[string]any{"error": map[string]string{"code": "stream_limit", "message": "Batas koneksi notifikasi tercapai.", "request_id": w.Header().Get("X-Request-ID")}})
		return
	}
	defer h.release(b.UserID)
	controller := http.NewResponseController(w)
	// Real servers support deadlines; recorders may not. Other failures are fatal.
	if err := controller.SetWriteDeadline(time.Time{}); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	write := func(value string) bool {
		if err := controller.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
			return false
		}
		if _, err := fmt.Fprint(w, value); err != nil {
			return false
		}
		return controller.Flush() == nil
	}
	if !write("retry: 3000\n: connected\n\n") {
		return
	}
	poll := time.NewTicker(2 * time.Second)
	defer poll.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		for _, n := range b.Notifications {
			data, err := json.Marshal(n)
			if err != nil {
				return
			}
			if !write(fmt.Sprintf("id: %d\nevent: notification\ndata: %s\n\n", n.ID, data)) {
				return
			}
			after = n.ID
		}
		if len(b.Notifications) < streamBatchSize {
			if !waitForPoll(r.Context(), poll.C, heartbeat.C, write) {
				return
			}
		}
		b, err = h.Service.since(r.Context(), token, after)
		if err != nil {
			if errors.Is(err, apiutil.ErrUnauthenticated) {
				write("event: session-ended\ndata: {}\n\n")
			} else if r.Context().Err() == nil {
				write("event: unavailable\ndata: {}\n\n")
			}
			return
		}
	}
}

// Heartbeats keep the connection alive while waiting; only a poll releases the
// caller to fetch a fresh authenticated batch. They never replay consumed data.
func waitForPoll(ctx context.Context, poll, heartbeat <-chan time.Time, write func(string) bool) bool {
	for {
		select {
		case <-ctx.Done():
			return false
		case <-poll:
			return true
		case <-heartbeat:
			if !write(": heartbeat\n\n") {
				return false
			}
		}
	}
}
