package laboratory

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

// CancelPending is called by care while holding the episode row lock, in the
// same transaction as ending it. Already approved assignments are preserved.
func CancelPending(ctx context.Context, tx pgx.Tx, writer *audit.Writer, actor auth.User, episodeID, patientID string) error {
	rows, err := tx.Query(ctx, `SELECT id::text FROM lab_requests WHERE episode_id=$1 AND status='pending_patient_approval' ORDER BY id FOR UPDATE`, episodeID)
	if err != nil {
		return err
	}
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE lab_requests SET status='cancelled',cancelled_at=clock_timestamp(),cancellation_reason='episode_ended',updated_at=clock_timestamp() WHERE id=$1`, id); err != nil {
			return err
		}
		if err := writer.Append(ctx, tx, actor, audit.Event{ChainID: patientID, Action: "lab_request.cancel", ResourceType: "lab_request", ResourceID: id, Outcome: "allow", Basis: "episode_ended"}); err != nil {
			return err
		}
	}
	return nil
}
