package notification

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/apiutil"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

type Service struct{ DB apiutil.DB }
type Notification struct {
	ID           int64      `json:"id"`
	Action       string     `json:"action"`
	ResourceType string     `json:"resource_type"`
	ResourceID   string     `json:"resource_id"`
	Message      string     `json:"message"`
	CreatedAt    time.Time  `json:"created_at"`
	ReadAt       *time.Time `json:"read_at"`
}
type Page struct {
	Notifications []Notification `json:"notifications"`
	Limit         int            `json:"limit"`
	NextBeforeID  *int64         `json:"next_before_id"`
}

const columns = `id,action,resource_type,resource_id::text,message,created_at,read_at`

func scan(row interface{ Scan(...any) error }) (Notification, error) {
	var n Notification
	err := row.Scan(&n.ID, &n.Action, &n.ResourceType, &n.ResourceID, &n.Message, &n.CreatedAt, &n.ReadAt)
	return n, err
}

func (s *Service) List(ctx context.Context, token []byte, limit int, before int64, unread bool) (Page, error) {
	page := Page{Notifications: make([]Notification, 0), Limit: limit}
	if limit < 1 || limit > 100 || before < 0 {
		return page, apiutil.ErrInvalid
	}
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		rows, err := tx.Query(ctx, `SELECT `+columns+` FROM notifications WHERE user_id=$1 AND ($2::bigint=0 OR id<$2) AND (NOT $3::boolean OR read_at IS NULL) ORDER BY id DESC LIMIT $4`, actor.ID, before, unread, limit+1)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			n, err := scan(rows)
			if err != nil {
				return err
			}
			page.Notifications = append(page.Notifications, n)
		}
		return rows.Err()
	})
	if err != nil {
		return page, err
	}
	if len(page.Notifications) > limit {
		page.Notifications = page.Notifications[:limit]
		last := page.Notifications[limit-1].ID
		page.NextBeforeID = &last
	}
	return page, nil
}
func (s *Service) Unread(ctx context.Context, token []byte) (int64, error) {
	var count int64
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id=$1 AND read_at IS NULL`, actor.ID).Scan(&count)
	})
	return count, err
}
func (s *Service) Read(ctx context.Context, token []byte, id int64) (Notification, error) {
	var n Notification
	err := apiutil.Transact(ctx, s.DB, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if id <= 0 {
			return apiutil.ErrInvalid
		}
		var err error
		n, err = scan(tx.QueryRow(ctx, `UPDATE notifications SET read_at=COALESCE(read_at,clock_timestamp()) WHERE id=$1 AND user_id=$2 RETURNING `+columns, id, actor.ID))
		if errors.Is(err, pgx.ErrNoRows) {
			return apiutil.ErrForbidden
		}
		return err
	})
	return n, err
}
