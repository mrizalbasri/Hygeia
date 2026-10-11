package audit

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

// Event's field order is the canonical format, version 1. Context keys are sorted
// by encoding/json. Database timestamps are normalized to UTC microseconds.
type Event struct {
	Version      int            `json:"version"`
	ChainID      string         `json:"chain_id"`
	Seq          int64          `json:"seq"`
	OccurredAt   time.Time      `json:"occurred_at"`
	ActorID      string         `json:"actor_id"`
	Action       string         `json:"action"`
	ResourceType string         `json:"resource_type"`
	ResourceID   string         `json:"resource_id"`
	Outcome      string         `json:"outcome"`
	Basis        string         `json:"basis"`
	Context      map[string]any `json:"context"`
	PrevHash     []byte         `json:"prev_hash"`
	Hash         []byte         `json:"hash"`
	// Optional grant details enter the existing canonical context, preserving v1 hashes.
	ConsentID        string     `json:"-"`
	ConsentExpiresAt *time.Time `json:"-"`
}

type Writer struct{ key []byte }

func NewWriter(key []byte) (*Writer, error) {
	if len(key) != 32 {
		return nil, errors.New("AUDIT_HMAC_KEY must contain 32 bytes")
	}
	return &Writer{key: bytes.Clone(key)}, nil
}

func (w *Writer) digest(event Event) ([]byte, error) {
	// Hash/prev_hash are excluded from the payload; prev_hash is the HMAC prefix.
	payload := struct {
		Version      int            `json:"version"`
		ChainID      string         `json:"chain_id"`
		Seq          int64          `json:"seq"`
		OccurredAt   string         `json:"occurred_at"`
		ActorID      string         `json:"actor_id"`
		Action       string         `json:"action"`
		ResourceType string         `json:"resource_type"`
		ResourceID   string         `json:"resource_id"`
		Outcome      string         `json:"outcome"`
		Basis        string         `json:"basis"`
		Context      map[string]any `json:"context"`
	}{1, event.ChainID, event.Seq, event.OccurredAt.UTC().Format("2006-01-02T15:04:05.000000Z"), event.ActorID, event.Action, event.ResourceType, event.ResourceID, event.Outcome, event.Basis, event.Context}
	canonical, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	hash := hmac.New(sha256.New, w.key)
	hash.Write(event.PrevHash)
	hash.Write(canonical)
	return hash.Sum(nil), nil
}

// Append must share the caller's business transaction. The per-patient head lock
// serializes concurrent writers without locking unrelated patient chains.
func (w *Writer) Append(ctx context.Context, tx pgx.Tx, actor auth.User, event Event) error {
	if _, err := tx.Exec(ctx, `INSERT INTO audit_chain_heads(chain_id,last_hash) VALUES($1,$2) ON CONFLICT DO NOTHING`, event.ChainID, make([]byte, 32)); err != nil {
		return err
	}
	if err := tx.QueryRow(ctx, `SELECT last_seq,last_hash FROM audit_chain_heads WHERE chain_id=$1 FOR UPDATE`, event.ChainID).Scan(&event.Seq, &event.PrevHash); err != nil {
		return err
	}
	event.Version = 1
	event.Seq++
	event.OccurredAt = time.Now().UTC().Truncate(time.Microsecond)
	event.ActorID = actor.ID
	event.Context = map[string]any{"actor_name": actor.FullName, "actor_role": actor.Role}
	if event.ConsentID != "" {
		event.Context["consent_id"] = event.ConsentID
		if event.ConsentExpiresAt != nil {
			event.Context["consent_expires_at"] = event.ConsentExpiresAt.UTC().Format("2006-01-02T15:04:05.000000Z")
		}
	}
	if actor.Organization != nil {
		event.Context["organization_id"] = actor.Organization.ID
		event.Context["organization_name"] = actor.Organization.Name
	}
	hash, err := w.digest(event)
	if err != nil {
		return err
	}
	event.Hash = hash
	snapshot, err := json.Marshal(event.Context)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO audit_events(chain_id,seq,occurred_at,actor_id,action,resource_type,resource_id,outcome,basis,context,prev_hash,hash)
        VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`, event.ChainID, event.Seq, event.OccurredAt, event.ActorID, event.Action, event.ResourceType, event.ResourceID, event.Outcome, event.Basis, snapshot, event.PrevHash, event.Hash); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE audit_chain_heads SET last_seq=$2,last_hash=$3 WHERE chain_id=$1`, event.ChainID, event.Seq, event.Hash)
	return err
}

const eventColumns = `chain_id::text,seq,occurred_at,actor_id::text,action,resource_type,resource_id::text,outcome,basis,context,prev_hash,hash`

func scanEvent(row interface{ Scan(...any) error }) (Event, error) {
	event := Event{Version: 1}
	var snapshot []byte
	err := row.Scan(&event.ChainID, &event.Seq, &event.OccurredAt, &event.ActorID, &event.Action, &event.ResourceType, &event.ResourceID, &event.Outcome, &event.Basis, &snapshot, &event.PrevHash, &event.Hash)
	if err != nil {
		return event, err
	}
	decoder := json.NewDecoder(bytes.NewReader(snapshot))
	decoder.UseNumber()
	err = decoder.Decode(&event.Context)
	event.OccurredAt = event.OccurredAt.UTC()
	return event, err
}

type Verification struct {
	Chain            string `json:"chain"`
	Events           int64  `json:"events"`
	Intact           bool   `json:"intact"`
	FirstMismatchSeq *int64 `json:"first_mismatch_seq,omitempty"`
}

// Verify freezes the head while checking the stream and tail, detecting edits,
// gaps and deleted tails. Authorization belongs to the calling service.
func (w *Writer) Verify(ctx context.Context, tx pgx.Tx, chain string) (Verification, error) {
	result := Verification{Chain: chain, Intact: true}
	var lastSeq int64
	var lastHash []byte
	err := tx.QueryRow(ctx, `SELECT last_seq,last_hash FROM audit_chain_heads WHERE chain_id=$1 FOR SHARE`, chain).Scan(&lastSeq, &lastHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	rows, err := tx.Query(ctx, `SELECT `+eventColumns+` FROM audit_events WHERE chain_id=$1 ORDER BY seq`, chain)
	if err != nil {
		return result, err
	}
	defer rows.Close()
	prev := make([]byte, 32)
	mismatch := func(seq int64) {
		if result.Intact {
			result.Intact = false
			result.FirstMismatchSeq = &seq
		}
	}
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return result, err
		}
		result.Events++
		hash, err := w.digest(event)
		if err != nil {
			return result, err
		}
		if event.Seq != result.Events || !hmac.Equal(event.PrevHash, prev) || !hmac.Equal(hash, event.Hash) {
			mismatch(result.Events)
		}
		prev = event.Hash
	}
	if err := rows.Err(); err != nil {
		return result, err
	}
	if lastSeq != result.Events || !hmac.Equal(lastHash, prev) {
		mismatch(result.Events + 1)
	}
	return result, nil
}

type Summary struct {
	Seq              int64     `json:"seq"`
	OccurredAt       time.Time `json:"occurred_at"`
	ActorName        string    `json:"actor_name"`
	Action           string    `json:"action"`
	ResourceID       string    `json:"resource_id"`
	Outcome          string    `json:"outcome"`
	Basis            string    `json:"basis"`
	Message          string    `json:"message"`
	ConsentID        string    `json:"consent_id,omitempty"`
	ConsentExpiresAt string    `json:"consent_expires_at,omitempty"`
}

func Summaries(ctx context.Context, tx pgx.Tx, chain string, limit, offset int) ([]Summary, error) {
	rows, err := tx.Query(ctx, `SELECT `+eventColumns+` FROM audit_events WHERE chain_id=$1 ORDER BY seq DESC LIMIT $2 OFFSET $3`, chain, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Summary, 0)
	descriptions := map[string]string{
		"episode.invite": "mengirim undangan perawatan", "episode.accept": "menerima undangan perawatan",
		"episode.reject": "menolak undangan perawatan", "episode.end": "mengakhiri episode perawatan",
		"episode.view": "melihat detail episode perawatan", "episode.list": "melihat daftar episode perawatan",
		"lab_request.create": "membuat permintaan laboratorium", "lab_request.approve": "menyetujui pengiriman permintaan laboratorium",
		"lab_request.reject": "menolak pengiriman permintaan laboratorium", "lab_request.cancel": "membatalkan permintaan laboratorium",
		"lab_request.view": "melihat permintaan laboratorium", "lab_request.list": "melihat daftar permintaan laboratorium",
		"lab_request.status": "mengubah status permintaan laboratorium", "lab_request.accepted": "menerima permintaan laboratorium",
		"lab_request.in_progress": "memproses permintaan laboratorium", "lab_result.submit": "mengirim hasil laboratorium",
		"lab_result.view": "melihat hasil laboratorium",
		"consent.request": "meminta izin akses hasil laboratorium", "consent.approve": "menyetujui izin akses hasil laboratorium",
		"consent.deny": "menolak izin akses hasil laboratorium", "consent.revoke": "mencabut izin akses hasil laboratorium",
		"consent.view": "melihat rincian izin akses", "consent.list": "melihat daftar izin akses",
	}
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		description := descriptions[event.Action]
		if description == "" {
			description = "mengakses episode perawatan"
		}
		name, _ := event.Context["actor_name"].(string)
		message := name + " " + description + " Anda."
		if event.Outcome == "deny" {
			message = "Percobaan " + name + " untuk " + description + " Anda ditolak."
		}
		consentID, _ := event.Context["consent_id"].(string)
		expires, _ := event.Context["consent_expires_at"].(string)
		if event.Outcome == "allow" && event.Basis == "consent" {
			message = name + " " + description + " dengan izin Anda, berlaku sampai " + expires + "."
		}
		result = append(result, Summary{Seq: event.Seq, OccurredAt: event.OccurredAt, ActorName: name, Action: event.Action, ResourceID: event.ResourceID, Outcome: event.Outcome, Basis: event.Basis, Message: message, ConsentID: consentID, ConsentExpiresAt: expires})
	}
	return result, rows.Err()
}
