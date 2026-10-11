package prescription

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

// CancelPending runs inside care's episode-end transaction, after locking the
// episode. Already approved pharmacy assignments continue independently.
func CancelPending(ctx context.Context, tx pgx.Tx, writer *audit.Writer, actor auth.User, episodeID, patientID string) error {
	rows, err := tx.Query(ctx, `SELECT id::text FROM prescriptions WHERE episode_id=$1 AND status='pending_patient_approval' ORDER BY id FOR UPDATE`, episodeID)
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
		if _, err := tx.Exec(ctx, `UPDATE prescriptions SET status='cancelled',cancelled_at=clock_timestamp(),cancellation_reason='episode_ended',updated_at=clock_timestamp() WHERE id=$1`, id); err != nil {
			return err
		}
		if err := writer.Append(ctx, tx, actor, audit.Event{ChainID: patientID, Action: "prescription.cancel", ResourceType: "prescription", ResourceID: id, Outcome: "allow", Basis: "episode_ended"}); err != nil {
			return err
		}
	}
	return nil
}
