// Package demo creates an isolated synthetic walkthrough, never patient data.
package demo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
)

const PatientRef = "HYG-PILOT-SYNTH-001"

// Stable IDs form a separate seed namespace. All inserts are atomic; repeats
// preserve password changes and any workflow progress, including ended episodes.
func id(n int) string { return fmt.Sprintf("dead2026-1028-4000-8000-%012d", n) }

func CheckEnvironment(environment string) error {
	if environment != "development" && environment != "test" {
		return errors.New("synthetic workflow seed is only permitted with APP_ENV=development or test")
	}
	return nil
}

func SeedWorkflow(ctx context.Context, pool *pgxpool.Pool, writer *audit.Writer, environment, password string) (bool, error) {
	if err := CheckEnvironment(environment); err != nil {
		return false, err
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return false, errors.New("DEMO_PASSWORD must contain 12 to 1024 bytes")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	// Serialize this seed, including concurrent first runs, without a new table.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(20261028, 1)`); err != nil {
		return false, err
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM care_episodes WHERE id=$1)`, id(20)).Scan(&exists); err != nil {
		return false, err
	}
	if exists {
		return false, nil
	}
	actors := map[string]auth.User{}
	for i, role := range []string{"patient", "doctor", "lab", "pharmacy", "admin"} {
		actor := auth.User{ID: id(i + 1), Email: "pilot." + role + "@demo.hygeia.test", FullName: "Demo Sintetis " + role, Role: role}
		// Deliberately no upsert: a namespace collision aborts instead of changing
		// an existing identity, password, role, organization, or patient record.
		if _, err := tx.Exec(ctx, `INSERT INTO users(id,email,password_hash,full_name,role) VALUES($1,$2,$3,$4,$5)`, actor.ID, actor.Email, hash, actor.FullName, role); err != nil {
			return false, fmt.Errorf("synthetic account namespace conflict or insert failed: %w", err)
		}
		if role == "doctor" || role == "lab" || role == "pharmacy" {
			kind := map[string]string{"doctor": "clinic", "lab": "laboratory", "pharmacy": "pharmacy"}[role]
			org := &auth.Organization{ID: id(i + 10), Name: "Demo Sintetis " + kind, Type: kind}
			if _, err := tx.Exec(ctx, `INSERT INTO organizations(id,name,type) VALUES($1,$2,$3)`, org.ID, org.Name, org.Type); err != nil {
				return false, err
			}
			if _, err := tx.Exec(ctx, `INSERT INTO organization_memberships(user_id,organization_id,staff_role) VALUES($1,$2,$3)`, actor.ID, org.ID, role); err != nil {
				return false, err
			}
			actor.Organization = org
		}
		actors[role] = actor
	}
	if _, err := tx.Exec(ctx, `INSERT INTO patients(id,user_id,patient_ref,sex,birth_year) VALUES($1,$2,$3,'unknown',2000)`, id(10), actors["patient"].ID, PatientRef); err != nil {
		return false, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO care_episodes(id,patient_id,doctor_id,organization_id,status,accepted_at) VALUES($1,$2,$3,$4,'active',now())`, id(20), id(10), actors["doctor"].ID, actors["doctor"].Organization.ID); err != nil {
		return false, err
	}
	for _, n := range []int{30, 31} {
		status := "completed"
		if n == 31 {
			status = "sent"
		}
		if _, err := tx.Exec(ctx, `INSERT INTO lab_requests(id,episode_id,lab_org_id,test_code,test_name,status,approved_at,completed_at) VALUES($1,$2,$3,'SYNTHETIC','Pemeriksaan sintetis bukan diagnosis',$4,now(),CASE WHEN $4='completed' THEN now() END)`, id(n), id(20), actors["lab"].Organization.ID, status); err != nil {
			return false, err
		}
	}
	if _, err := tx.Exec(ctx, `INSERT INTO lab_results(id,lab_request_id,value,unit,reference_range,flag,note,submitted_by) VALUES($1,$2,'DEMO-42','demo','sintetis','unknown','Bukan hasil medis. Hanya demonstrasi.', $3)`, id(32), id(30), actors["lab"].ID); err != nil {
		return false, err
	}
	for _, n := range []int{40, 41} {
		status := "ready"
		if n == 41 {
			status = "sent"
		}
		if _, err := tx.Exec(ctx, `INSERT INTO prescriptions(id,episode_id,pharmacy_org_id,status,approved_at) VALUES($1,$2,$3,$4,now())`, id(n), id(20), actors["pharmacy"].Organization.ID, status); err != nil {
			return false, err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO prescription_items(prescription_id,position,medication_name,dosage,instructions,quantity) VALUES($1,1,'Obat simulasi tanpa bahan aktif','Dosis simulasi','Jangan digunakan sebagai petunjuk pengobatan',1)`, id(n)); err != nil {
			return false, err
		}
	}
	for _, event := range []struct {
		role, action, resource string
		number                 int
	}{
		{"doctor", "episode.invite", "care_episode", 20}, {"patient", "episode.accept", "care_episode", 20},
		{"doctor", "lab_request.create", "lab_request", 30}, {"patient", "lab_request.approve", "lab_request", 30},
		{"lab", "lab_request.accepted", "lab_request", 30}, {"lab", "lab_request.in_progress", "lab_request", 30}, {"lab", "lab_result.submit", "lab_result", 32},
		{"doctor", "lab_request.create", "lab_request", 31}, {"patient", "lab_request.approve", "lab_request", 31},
		{"doctor", "prescription.create", "prescription", 40}, {"patient", "prescription.approve", "prescription", 40},
		{"pharmacy", "prescription.accepted", "prescription", 40}, {"pharmacy", "prescription.preparing", "prescription", 40}, {"pharmacy", "prescription.ready", "prescription", 40},
		{"doctor", "prescription.create", "prescription", 41}, {"patient", "prescription.approve", "prescription", 41},
	} {
		if err := writer.Append(ctx, tx, actors[event.role], audit.Event{ChainID: id(10), Action: event.action, ResourceType: event.resource, ResourceID: id(event.number), Outcome: "allow", Basis: "synthetic_demo_seed"}); err != nil {
			return false, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
