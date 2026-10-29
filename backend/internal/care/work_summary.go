package care

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

type PendingWork struct {
	Episodes      int64 `json:"episodes"`
	LabRequests   int64 `json:"lab_requests"`
	Prescriptions int64 `json:"prescriptions"`
	Consents      int64 `json:"consents"`
}

type WorkSummary struct {
	ActiveEpisodes     int64       `json:"active_episodes"`
	LabsInProgress     int64       `json:"labs_in_progress"`
	PrescriptionsReady int64       `json:"prescriptions_ready"`
	Pending            PendingWork `json:"pending"`
}

// WorkSummary counts the patient's complete workload, not a recent page. The
// single statement gives all counters the same snapshot without reading values.
func (s *Service) WorkSummary(ctx context.Context, token []byte) (WorkSummary, error) {
	var result WorkSummary
	err := s.transact(ctx, token, func(ctx context.Context, tx pgx.Tx, actor auth.User) error {
		if actor.Role != "patient" {
			return ErrForbidden
		}
		return tx.QueryRow(ctx, `
			WITH patient AS (SELECT id FROM patients WHERE user_id=$1),
			episodes AS (SELECT id,status FROM care_episodes WHERE patient_id IN (SELECT id FROM patient)),
			labs AS (SELECT status FROM lab_requests WHERE episode_id IN (SELECT id FROM episodes)),
			rx AS (SELECT status FROM prescriptions WHERE episode_id IN (SELECT id FROM episodes))
			SELECT
				(SELECT count(*) FROM episodes WHERE status='active'),
				(SELECT count(*) FROM labs WHERE status IN ('sent','accepted','in_progress')),
				(SELECT count(*) FROM rx WHERE status='ready'),
				(SELECT count(*) FROM episodes WHERE status='pending'),
				(SELECT count(*) FROM labs WHERE status='pending_patient_approval'),
				(SELECT count(*) FROM rx WHERE status='pending_patient_approval'),
				(SELECT count(*) FROM consents WHERE patient_id IN (SELECT id FROM patient) AND status='pending')
		`, actor.ID).Scan(
			&result.ActiveEpisodes, &result.LabsInProgress, &result.PrescriptionsReady,
			&result.Pending.Episodes, &result.Pending.LabRequests,
			&result.Pending.Prescriptions, &result.Pending.Consents,
		)
	})
	return result, err
}

func (h *Handler) workSummary(w http.ResponseWriter, r *http.Request) {
	// Patient selection and pagination are deliberately not part of this API.
	if r.URL.RawQuery != "" {
		apiError(w, 400, "invalid_input", "Ringkasan mengikuti akun pasien yang sedang masuk.")
		return
	}
	result, err := h.Service.WorkSummary(r.Context(), auth.SessionHash(r))
	if err != nil {
		h.respondError(w, r, err)
		return
	}
	jsonResponse(w, 200, map[string]any{"summary": result})
}
