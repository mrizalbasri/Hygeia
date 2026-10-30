package laboratory

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"sort"
)

func (s *Service) Receipts(ctx context.Context, token []byte, limit, offset int) ([]ReceiptMetadata, error) {
	receipts := make([]ReceiptMetadata, 0)
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if actor.Role != "lab" || actor.Organization == nil || actor.Organization.Type != "laboratory" {
			return ErrForbidden
		}
		rows, err := tx.Query(ctx, `SELECT x.id::text,r.id::text,r.status,x.submitted_at,p.patient_ref,r.test_code,r.test_name,p.id::text
   FROM lab_results x JOIN lab_requests r ON r.id=x.lab_request_id
   JOIN care_episodes e ON e.id=r.episode_id JOIN patients p ON p.id=e.patient_id
   WHERE r.lab_org_id=$1 ORDER BY x.submitted_at DESC,x.id DESC LIMIT $2 OFFSET $3`, actor.Organization.ID, limit, offset)
		if err != nil {
			return err
		}
		chains := make([]string, 0)
		for rows.Next() {
			var receipt ReceiptMetadata
			var chain string
			if err := rows.Scan(&receipt.ID, &receipt.LabRequestID, &receipt.Status, &receipt.SubmittedAt, &receipt.PatientRef, &receipt.TestCode, &receipt.TestName, &chain); err != nil {
				rows.Close()
				return err
			}
			receipts = append(receipts, receipt)
			chains = append(chains, chain)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		// Acquire multiple patient audit chains in a stable order, as other lists do.
		order := make([]int, len(receipts))
		for i := range order {
			order[i] = i
		}
		sort.Slice(order, func(i, j int) bool { return chains[order[i]] < chains[order[j]] })
		for _, i := range order {
			if err := s.Audit.Append(ctx, tx, actor, audit.Event{ChainID: chains[i], Action: "lab_receipt.view", ResourceType: "lab_result", ResourceID: receipts[i].ID, Outcome: "allow", Basis: "approved_lab_assignment"}); err != nil {
				return err
			}
		}
		return nil
	})
	return receipts, err
}
