package care

import (
	"context"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Read authorization must be checked after the audit head wait too, not only
// after the grant lock. A still-valid grant must continue to work.
func assertGrantAfterAuditWait(t *testing.T, f *fixture, path string, expiry time.Time, expired bool) {
	t.Helper()
	ctx := context.Background()
	var clock atomic.Int64
	clock.Store(expiry.Add(-time.Microsecond).UnixMicro())
	now := func() time.Time { return time.UnixMicro(clock.Load()) }
	f.labService.Now = now
	f.prescriptionService.Now = now
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT chain_id FROM audit_chain_heads WHERE chain_id=$1 FOR UPDATE`, f.patientID); err != nil {
		t.Fatal(err)
	}
	var holder int
	if err := tx.QueryRow(ctx, `SELECT pg_backend_pid()`).Scan(&holder); err != nil {
		t.Fatal(err)
	}
	done := make(chan *httptest.ResponseRecorder, 1)
	go func() { done <- f.request("colleague", "GET", path, "", false) }()
	deadline := time.Now().Add(2 * time.Second)
	waiting := false
	for time.Now().Before(deadline) {
		if err := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE wait_event_type='Lock' AND $1=ANY(pg_blocking_pids(pid)) AND query LIKE '%SELECT last_seq,last_hash FROM audit_chain_heads%')`, holder).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("read never waited for audit head")
	}
	if expired {
		clock.Store(expiry.UnixMicro())
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case response := <-done:
		want := 200
		if expired {
			want = 403
		}
		expectStatus(t, response, want)
		if expired && (strings.Contains(response.Body.String(), "SYNTHETIC-RESULT") || strings.Contains(response.Body.String(), "SYNTHETIC-MEDICATION")) {
			t.Fatal("expired grant exposed clinical payload")
		}
	case <-time.After(6 * time.Second):
		t.Fatal("read stuck after releasing audit head")
	}
	var outcome string
	if err := f.pool.QueryRow(ctx, `SELECT outcome FROM audit_events WHERE chain_id=$1 AND actor_id=$2 AND action IN ('lab_result.view','prescription.view') ORDER BY seq DESC LIMIT 1`, f.patientID, f.ids["colleague"]).Scan(&outcome); err != nil {
		t.Fatal(err)
	}
	want := "allow"
	if expired {
		want = "deny"
	}
	if outcome != want {
		t.Fatalf("audit outcome %s, want %s", outcome, want)
	}
	verification, err := f.service.Verify(ctx, f.tokens["patient"], "")
	if err != nil || !verification.Intact {
		t.Fatal("audit chain invalid", err)
	}
}

func TestEmergencyReadAfterAuditWait(t *testing.T) {
	for _, resource := range []string{"laboratory", "prescription"} {
		for _, expired := range []bool{false, true} {
			label := "valid"
			if expired {
				label = "expired"
			}
			t.Run(resource+"/"+label, func(t *testing.T) {
				f := integrationFixture(t)
				e := activeLabEpisode(t, f)
				path := ""
				if resource == "laboratory" {
					r := newLabRequest(t, f, e)
					approveLab(t, f, r.ID)
					processLab(t, f, r.ID)
					result := submitLab(t, f, r.ID)
					path = "/api/v1/lab-results/" + result.ID
				} else {
					r := newPrescription(t, f, e)
					approvePrescription(t, f, r.ID)
					path = "/api/v1/prescriptions/" + r.ID
				}
				g, err := f.breakGlassService.Start(context.Background(), f.tokens["colleague"], "TEST-patient", "unconscious", "Synthetic reason for audit wait test")
				if err != nil {
					t.Fatal(err)
				}
				assertGrantAfterAuditWait(t, f, path, g.ExpiresAt, expired)
			})
		}
	}
}
