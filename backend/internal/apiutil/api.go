// Package apiutil contains the transport and transaction primitives shared by
// the administrative, notification and temporary-access modules.
package apiutil

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

var (
	ErrUnauthenticated = errors.New("unauthenticated")
	ErrForbidden       = errors.New("forbidden")
	ErrConflict        = errors.New("invalid state")
	ErrInvalid         = errors.New("invalid input")
)

type DB interface {
	Begin(context.Context) (pgx.Tx, error)
}

func Transact(ctx context.Context, db DB, token []byte, fn func(context.Context, pgx.Tx, auth.User) error) error {
	if len(token) != 32 {
		return ErrUnauthenticated
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	actor, err := (auth.Store{DB: tx}).UserBySession(ctx, token)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrUnauthenticated
	}
	if err != nil {
		return err
	}
	decision := fn(ctx, tx, actor)
	if decision != nil && !errors.Is(decision, ErrForbidden) && !errors.Is(decision, ErrConflict) && !errors.Is(decision, ErrInvalid) {
		var pgerr *pgconn.PgError
		if errors.As(decision, &pgerr) && pgerr.Code == "23505" {
			return ErrConflict
		}
		return decision
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return decision
}

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func Failure(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := 503, "unavailable", "Layanan sedang tidak tersedia."
	switch {
	case errors.Is(err, ErrUnauthenticated):
		status, code, message = 401, "unauthenticated", "Silakan masuk terlebih dahulu."
	case errors.Is(err, ErrForbidden):
		status, code, message = 403, "forbidden", "Akses tidak diizinkan."
	case errors.Is(err, ErrConflict):
		status, code, message = 409, "invalid_state", "Status tidak mengizinkan tindakan ini."
	case errors.Is(err, ErrInvalid):
		status, code, message = 400, "invalid_input", "Isi permintaan tidak valid."
	}
	JSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message, "request_id": w.Header().Get("X-Request-ID")}})
}
func Decode(w http.ResponseWriter, r *http.Request, value any) bool {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		JSON(w, 415, map[string]any{"error": map[string]string{"code": "invalid_content_type", "message": "Gunakan application/json.", "request_id": w.Header().Get("X-Request-ID")}})
		return false
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 65536))
	decoder.DisallowUnknownFields()
	if decoder.Decode(value) != nil || decoder.Decode(&struct{}{}) != io.EOF {
		Failure(w, r, ErrInvalid)
		return false
	}
	return true
}

var uuid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func UUID(value string) bool { return uuid.MatchString(value) }
func Text(value string, min, max int) bool {
	return utf8.ValidString(value) && !strings.ContainsRune(value, 0) && utf8.RuneCountInString(strings.TrimSpace(value)) >= min && utf8.RuneCountInString(value) <= max
}
func Query(r *http.Request, allowed ...string) (url.Values, error) {
	values, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return nil, ErrInvalid
	}
	for key, items := range values {
		found := false
		for _, name := range allowed {
			if key == name {
				found = true
			}
		}
		if !found || len(items) != 1 {
			return nil, ErrInvalid
		}
	}
	return values, nil
}
func Pagination(values url.Values) (int, int, error) {
	limit, offset := 20, 0
	if values.Has("limit") {
		v, err := strconv.Atoi(values.Get("limit"))
		if err != nil || v < 1 || v > 100 {
			return 0, 0, ErrInvalid
		}
		limit = v
	}
	if values.Has("offset") {
		v, err := strconv.Atoi(values.Get("offset"))
		if err != nil || v < 0 || v > 10000 {
			return 0, 0, ErrInvalid
		}
		offset = v
	}
	return limit, offset, nil
}
