package care

import (
	"context"
	"testing"

	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/demo"
)

func TestSyntheticWorkflowRepeatPreservesProgressIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	created, err := demo.SeedWorkflow(ctx, f.pool, f.service.Audit, "test", "Synthetic-password-2026")
	if err != nil || !created {
		t.Fatalf("initial seed: %v %v", created, err)
	}
	var patient, prescription, user string
	if err := f.pool.QueryRow(ctx, `SELECT p.id::text,u.id::text FROM patients p JOIN users u ON u.id=p.user_id WHERE patient_ref=$1`, demo.PatientRef).Scan(&patient, &user); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT r.id::text FROM prescriptions r JOIN care_episodes e ON e.id=r.episode_id WHERE e.patient_id=$1 AND r.status='ready'`, patient).Scan(&prescription); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE prescriptions SET status='completed',completed_at=now() WHERE id=$1`, prescription); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE users SET password_hash='preserved-password-marker',full_name='Changed demo name' WHERE id=$1`, user); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE chain_id=$1`, patient).Scan(&before); err != nil || before != 16 {
		t.Fatalf("seed events %d %v", before, err)
	}
	created, err = demo.SeedWorkflow(ctx, f.pool, f.service.Audit, "test", "Different-password-2026")
	if err != nil || created {
		t.Fatalf("repeat seed: %v %v", created, err)
	}
	var after int
	var hash, name, status string
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM audit_events WHERE chain_id=$1`, patient).Scan(&after); err != nil || after != before {
		t.Fatal("repeat duplicated audit", err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT password_hash,full_name FROM users WHERE id=$1`, user).Scan(&hash, &name); err != nil || hash != "preserved-password-marker" || name != "Changed demo name" {
		t.Fatal("repeat overwrote identity", err)
	}
	if err := f.pool.QueryRow(ctx, `SELECT status FROM prescriptions WHERE id=$1`, prescription).Scan(&status); err != nil || status != "completed" {
		t.Fatal("repeat reset progress", err)
	}
	tx, err := f.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	verified, err := f.service.Audit.Verify(ctx, tx, patient)
	if err != nil || !verified.Intact {
		t.Fatalf("seed audit verification: %+v %v", verified, err)
	}
}

func TestSyntheticWorkflowCollisionRollsBackIntegration(t *testing.T) {
	f := integrationFixture(t)
	ctx := context.Background()
	_, err := f.pool.Exec(ctx, `INSERT INTO users(email,password_hash,full_name,role) VALUES('pilot.doctor@demo.hygeia.test','existing-marker','Existing user','doctor')`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := demo.SeedWorkflow(ctx, f.pool, f.service.Audit, "test", "Synthetic-password-2026"); err == nil {
		t.Fatal("collision accepted")
	}
	var count int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE email='pilot.patient@demo.hygeia.test'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("failed seed left partial accounts", err)
	}
	_, hash, err := (auth.Store{DB: f.pool}).Credentials(ctx, "pilot.doctor@demo.hygeia.test")
	if err != nil || hash != "existing-marker" {
		t.Fatal("existing password changed", err)
	}
}
