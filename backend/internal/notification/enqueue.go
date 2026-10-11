package notification

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

var messages = map[string]string{
	"episode.invite": "Ada undangan perawatan baru.", "episode.accept": "Undangan perawatan diterima.", "episode.reject": "Undangan perawatan ditolak.", "episode.end": "Episode perawatan diakhiri.",
	"lab_request.create": "Permintaan laboratorium menunggu persetujuan pasien.", "lab_request.approve": "Pengiriman permintaan laboratorium disetujui.", "lab_request.reject": "Pengiriman permintaan laboratorium ditolak.", "lab_request.cancel": "Permintaan laboratorium dibatalkan.", "lab_request.accepted": "Laboratorium menerima permintaan.", "lab_request.in_progress": "Pemeriksaan laboratorium sedang diproses.", "lab_result.submit": "Hasil laboratorium tersedia.",
	"prescription.create": "Resep menunggu pilihan apotek dan persetujuan pasien.", "prescription.approve": "Resep dikirim ke apotek pilihan pasien.", "prescription.reject": "Pengiriman resep ditolak.", "prescription.cancel": "Resep dibatalkan.", "prescription.accepted": "Apotek menerima resep.", "prescription.preparing": "Apotek sedang menyiapkan resep.", "prescription.ready": "Resep siap diambil.", "prescription.completed": "Pemenuhan resep selesai.",
	"consent.request": "Ada permintaan izin akses data.", "consent.approve": "Izin akses data disetujui.", "consent.deny": "Izin akses data ditolak.", "consent.revoke": "Izin akses data dicabut.",
}

// Enqueue shares the business/audit transaction. Only known successful
// milestones fan out; reads and denials never create notifications.
func Enqueue(ctx context.Context, tx pgx.Tx, chain string, seq int64, actorID, action, resourceType, resourceID string) error {
	message, ok := messages[action]
	if !ok {
		return nil
	}
	recipients := `SELECT user_id FROM patients WHERE id=$1`
	switch {
	case strings.HasPrefix(action, "episode."):
		recipients += ` UNION SELECT doctor_id FROM care_episodes WHERE id=$2`
	case strings.HasPrefix(action, "lab_request.") || action == "lab_result.submit":
		requestCondition := `r.id=$2`
		if action == "lab_result.submit" {
			requestCondition = `r.id=(SELECT lab_request_id FROM lab_results WHERE id=$2)`
		}
		recipients += ` UNION SELECT e.doctor_id FROM lab_requests r JOIN care_episodes e ON e.id=r.episode_id JOIN organization_memberships d ON d.user_id=e.doctor_id AND d.organization_id=e.organization_id AND d.staff_role='doctor' WHERE e.status='active' AND ` + requestCondition
		recipients += ` UNION SELECT m.user_id FROM lab_requests r JOIN organization_memberships m ON m.organization_id=r.lab_org_id AND m.staff_role='lab' WHERE ` + requestCondition + ` AND r.status IN ('sent','accepted','in_progress')`
	case strings.HasPrefix(action, "prescription."):
		recipients += ` UNION SELECT e.doctor_id FROM prescriptions r JOIN care_episodes e ON e.id=r.episode_id JOIN organization_memberships d ON d.user_id=e.doctor_id AND d.organization_id=e.organization_id AND d.staff_role='doctor' WHERE r.id=$2 AND e.status='active'
   UNION SELECT m.user_id FROM prescriptions r JOIN organization_memberships m ON m.organization_id=r.pharmacy_org_id AND m.staff_role='pharmacy' WHERE r.id=$2 AND r.status IN ('sent','accepted','preparing','ready')`
	case strings.HasPrefix(action, "consent."):
		recipients += ` UNION SELECT grantee_user_id FROM consents WHERE id=$2`
	}
	_, err := tx.Exec(ctx, `INSERT INTO notifications(user_id,source_chain,source_seq,action,resource_type,resource_id,message)
  SELECT u.id,$1,$3,$4,$5,$2,$6 FROM users u WHERE u.disabled_at IS NULL AND u.id<>$7
  AND u.id IN (`+recipients+`) ON CONFLICT(user_id,source_chain,source_seq) DO NOTHING`, chain, resourceID, seq, action, resourceType, message, actorID)
	return err
}
