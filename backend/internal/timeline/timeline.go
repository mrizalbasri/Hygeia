// Package timeline projects committed business milestones from the patient's
// audit chain. Access attempts and reads remain in the separate access summary.
package timeline

import (
	"context"
	"errors"
	"sort"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrInvalidOptions = errors.New("invalid timeline options")

type Options struct {
	Kind      string
	Limit     int
	BeforeSeq int64
}

func (o Options) Valid() bool {
	if o.Limit < 1 || o.Limit > 100 || o.BeforeSeq < 0 {
		return false
	}
	switch o.Kind {
	case "", "episode", "laboratory", "prescription", "consent":
		return true
	default:
		return false
	}
}

type Event struct {
	Seq          int64     `json:"seq"`
	OccurredAt   time.Time `json:"occurred_at"`
	Kind         string    `json:"kind"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	ActorName    string    `json:"actor_name"`
	Message      string    `json:"message"`
}

type Page struct {
	Events        []Event `json:"events"`
	Limit         int     `json:"limit"`
	NextBeforeSeq *int64  `json:"next_before_seq"`
}

type milestone struct{ kind, resourceType, description string }

// Explicit allowlist: new audit actions never become patient milestones by
// accident. These records were committed atomically with their business action.
var milestones = map[string]milestone{
	"lab_result.correct":      {"laboratory", "lab_result", "mengirim koreksi hasil laboratorium Anda"},
	"episode.invite":          {"episode", "care_episode", "mengundang Anda ke episode perawatan"},
	"episode.accept":          {"episode", "care_episode", "menerima undangan perawatan"},
	"episode.reject":          {"episode", "care_episode", "menolak undangan perawatan"},
	"episode.end":             {"episode", "care_episode", "mengakhiri episode perawatan"},
	"lab_request.create":      {"laboratory", "lab_request", "membuat permintaan laboratorium untuk Anda"},
	"lab_request.approve":     {"laboratory", "lab_request", "menyetujui pengiriman permintaan laboratorium"},
	"lab_request.reject":      {"laboratory", "lab_request", "menolak pengiriman permintaan laboratorium"},
	"lab_request.cancel":      {"laboratory", "lab_request", "membatalkan permintaan laboratorium Anda"},
	"lab_request.accepted":    {"laboratory", "lab_request", "menerima permintaan laboratorium Anda"},
	"lab_request.in_progress": {"laboratory", "lab_request", "memproses permintaan laboratorium Anda"},
	"lab_result.submit":       {"laboratory", "lab_result", "mengirim hasil laboratorium Anda"},
	"prescription.create":     {"prescription", "prescription", "membuat resep untuk Anda"},
	"prescription.approve":    {"prescription", "prescription", "menyetujui pengiriman resep ke apotek"},
	"prescription.reject":     {"prescription", "prescription", "menolak pengiriman resep"},
	"prescription.cancel":     {"prescription", "prescription", "membatalkan resep Anda"},
	"prescription.accepted":   {"prescription", "prescription", "menerima resep Anda"},
	"prescription.preparing":  {"prescription", "prescription", "menyiapkan resep Anda"},
	"prescription.ready":      {"prescription", "prescription", "menandai resep Anda siap diambil"},
	"prescription.completed":  {"prescription", "prescription", "menyelesaikan resep Anda"},
	"consent.request":         {"consent", "consent", "meminta izin untuk melihat hasil laboratorium Anda"},
	"consent.approve":         {"consent", "consent", "menyetujui izin akses hasil laboratorium"},
	"consent.deny":            {"consent", "consent", "menolak izin akses hasil laboratorium"},
	"consent.revoke":          {"consent", "consent", "mencabut izin akses hasil laboratorium"},
}

func actionsForKind(kind string) []string {
	actions := make([]string, 0, len(milestones))
	for action, m := range milestones {
		if kind == "" || kind == m.kind {
			actions = append(actions, action)
		}
	}
	sort.Strings(actions)
	return actions
}

// List expects an already-authorized patient chain from the calling service.
// The primary key (chain_id,seq) supports the descending exclusive cursor.
// Reading adds no audit event, so refreshing never grows the timeline itself.
func List(ctx context.Context, tx pgx.Tx, chain string, options Options) (Page, error) {
	page := Page{Events: make([]Event, 0), Limit: options.Limit}
	if !options.Valid() {
		return page, ErrInvalidOptions
	}
	actions := actionsForKind(options.Kind)
	resourceTypes := make([]string, len(actions))
	for i, action := range actions {
		resourceTypes[i] = milestones[action].resourceType
	}
	rows, err := tx.Query(ctx, `SELECT seq,occurred_at,action,resource_type,resource_id::text,
        COALESCE(context->>'actor_name','') FROM audit_events
        WHERE chain_id=$1 AND outcome='allow'
        AND (action,resource_type) IN (SELECT * FROM unnest($2::text[],$3::text[]))
        AND ($4::bigint=0 OR seq<$4) ORDER BY seq DESC LIMIT $5`,
		chain, actions, resourceTypes, options.BeforeSeq, options.Limit+1)
	if err != nil {
		return page, err
	}
	defer rows.Close()
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.Seq, &event.OccurredAt, &event.Action, &event.ResourceType, &event.ResourceID, &event.ActorName); err != nil {
			return page, err
		}
		m := milestones[event.Action]
		event.Kind = m.kind
		event.Message = event.ActorName + " " + m.description + "."
		page.Events = append(page.Events, event)
	}
	if err := rows.Err(); err != nil {
		return page, err
	}
	if len(page.Events) > options.Limit {
		page.Events = page.Events[:options.Limit]
		last := page.Events[len(page.Events)-1].Seq
		page.NextBeforeSeq = &last
	}
	return page, nil
}
