package care

import (
	"context"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// Exercise grant expiry while a real PostgreSQL row lock delays the reader.
func TestTemporaryGrantExpiryAfterLockWait(t *testing.T) {
	for _, kind := range []string{"emergency", "carepass"} {
		t.Run(kind, func(t *testing.T) {
			f := integrationFixture(t)
			ctx := context.Background()
			e := activeLabEpisode(t, f)
			rx := newPrescription(t, f, e)
			approvePrescription(t, f, rx.ID)
			var clock atomic.Int64
			clock.Store(time.Now().UTC().UnixMicro())
			now := func() time.Time { return time.UnixMicro(clock.Load()) }
			f.prescriptionService.Now = now
			f.breakGlassService.Now = now
			f.carePassService.Now = now
			table, alias := "break_glass_sessions", "g"
			var id string
			var expiry time.Time
			if kind == "emergency" {
				g, err := f.breakGlassService.Start(ctx, f.tokens["colleague"], "TEST-patient", "unconscious", "Synthetic emergency reason")
				if err != nil {
					t.Fatal(err)
				}
				id, expiry = g.ID, g.ExpiresAt
			} else {
				table, alias = "carepasses", "c"
				c := createPass(t, f, "prescription", rx.ID)
				if _, err := f.carePassService.Redeem(ctx, f.tokens["colleague"], c.Token); err != nil {
					t.Fatal(err)
				}
				id, expiry = c.Pass.ID, c.Pass.ExpiresAt
			}
			locker, err := f.pool.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer locker.Rollback(ctx)
			if _, err := locker.Exec(ctx, `SELECT id FROM `+table+` WHERE id=$1 FOR UPDATE`, id); err != nil {
				t.Fatal(err)
			}
			done := make(chan *httptest.ResponseRecorder, 1)
			go func() { done <- f.request("colleague", "GET", "/api/v1/prescriptions/"+rx.ID, "", false) }()
			deadline := time.Now().Add(2 * time.Second)
			waiting := false
			for time.Now().Before(deadline) {
				if err := f.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE wait_event_type='Lock' AND query LIKE $1)`, "%FOR SHARE OF "+alias+"%").Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !waiting {
				t.Fatal("reader did not wait for grant lock")
			}
			clock.Store(expiry.UnixMicro())
			if err := locker.Commit(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case response := <-done:
				expectStatus(t, response, 403)
			case <-time.After(6 * time.Second):
				t.Fatal("reader stuck")
			}
		})
	}
}
func TestTemporaryGrantClinicBinding(t *testing.T) {
	for _, kind := range []string{"emergency", "carepass"} {
		t.Run(kind, func(t *testing.T) {
			f := integrationFixture(t)
			ctx := context.Background()
			e := activeLabEpisode(t, f)
			rx := newPrescription(t, f, e)
			approvePrescription(t, f, rx.ID)
			if kind == "emergency" {
				if _, err := f.breakGlassService.Start(ctx, f.tokens["colleague"], "TEST-patient", "unconscious", "Synthetic emergency reason"); err != nil {
					t.Fatal(err)
				}
			} else {
				c := createPass(t, f, "prescription", rx.ID)
				if _, err := f.carePassService.Redeem(ctx, f.tokens["colleague"], c.Token); err != nil {
					t.Fatal(err)
				}
			}
			expectStatus(t, f.request("colleague", "GET", "/api/v1/prescriptions/"+rx.ID, "", false), 200)
			var org string
			if err := f.pool.QueryRow(ctx, `INSERT INTO organizations(type,name) VALUES('clinic','Synthetic changed clinic') RETURNING id::text`).Scan(&org); err != nil {
				t.Fatal(err)
			}
			if _, err := f.pool.Exec(ctx, `UPDATE organization_memberships SET organization_id=$2 WHERE user_id=$1`, f.ids["colleague"], org); err != nil {
				t.Fatal(err)
			}
			expectStatus(t, f.request("colleague", "GET", "/api/v1/prescriptions/"+rx.ID, "", false), 403)
		})
	}
}
