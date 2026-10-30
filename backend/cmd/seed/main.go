package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/config"
	"github.com/mrizalbasri/Hygeia/backend/internal/database"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if err := config.LoadEnv(); err != nil {
		return err
	}
	appEnv := os.Getenv("APP_ENV")
	if appEnv != "" && appEnv != "development" && appEnv != "test" {
		return errors.New("demo seed is only permitted with APP_ENV=development or test")
	}
	pwd := os.Getenv("DEMO_PASSWORD")
	if pwd == "" {
		pwd = "Hygeia2026!#"
	}
	hash, err := auth.HashPassword(pwd)
	if err != nil {
		return errors.New("DEMO_PASSWORD must contain 12 to 1024 bytes")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return errors.New("could not start seed transaction")
	}
	defer tx.Rollback(ctx)

	// Professional judge/demo accounts (@hygeia.id) + legacy demo accounts
	accounts := []struct {
		email, name, role, org, orgType string
	}{
		// Professional Judge/Official accounts
		{"pasien@hygeia.id", "Aulia Rahman", "patient", "", ""},
		{"dokter@hygeia.id", "dr. Maya Sp.PD", "doctor", "Klinik Pratama Sentra", "clinic"},
		{"lab@hygeia.id", "Analis Laboratorium Sentra", "lab", "Laboratorium Sentra Medika", "laboratory"},
		{"apotek@hygeia.id", "Apoteker Sentra Farma", "pharmacy", "Apotek Sentra Farma", "pharmacy"},
		{"admin@hygeia.id", "Administrator Sistem Hygeia", "admin", "", ""},

		// Standard test accounts
		{"aulia@demo.hygeia.test", "Aulia Rahman", "patient", "", ""},
		{"maya@demo.hygeia.test", "Dr. Maya", "doctor", "Klinik Sentra", "clinic"},
		{"budi@demo.hygeia.test", "Dr. Budi", "doctor", "Klinik Harapan", "clinic"},
		{"rina@demo.hygeia.test", "Dr. Rina", "doctor", "IGD Harapan", "clinic"},
		{"lab@demo.hygeia.test", "Staf Lab Sentra", "lab", "Lab Sentra", "laboratory"},
		{"apotek@demo.hygeia.test", "Staf Apotek Sentra", "pharmacy", "Apotek Sentra", "pharmacy"},
		{"admin@demo.hygeia.test", "Admin Hygeia", "admin", "", ""},
	}

	for _, account := range accounts {
		_, err := tx.Exec(ctx, `INSERT INTO users(email,password_hash,full_name,role) VALUES($1,$2,$3,$4) 
ON CONFLICT(email) DO UPDATE SET password_hash=EXCLUDED.password_hash, full_name=EXCLUDED.full_name`, account.email, hash, account.name, account.role)
		if err != nil {
			return fmt.Errorf("seed user %s failed: %w", account.email, err)
		}
		if account.org != "" {
			_, err = tx.Exec(ctx, `INSERT INTO organizations(name,type) VALUES($1,$2) ON CONFLICT(type,name) DO NOTHING`, account.org, account.orgType)
			if err != nil {
				return errors.New("seed organizations failed")
			}
			_, err = tx.Exec(ctx, `INSERT INTO organization_memberships(user_id,organization_id,staff_role)
SELECT u.id,o.id,u.role FROM users u, organizations o WHERE u.email=$1 AND o.name=$2 AND o.type=$3
ON CONFLICT(user_id) DO UPDATE SET organization_id=EXCLUDED.organization_id`, account.email, account.org, account.orgType)
			if err != nil {
				return errors.New("seed memberships failed")
			}
		}
	}

	// 1. Seed Patient Profiles
	patients := []struct {
		email, ref, sex, birthDate, phone, addr, ecName, ecPhone string
		birthYear                                                int
	}{
		{"pasien@hygeia.id", "HYG-2026-PASIEN-01", "female", "1998-05-14", "081234567890", "Jl. Sudirman No. 45, Jakarta Selatan", "Budi Rahman", "081298765432", 1998},
		{"aulia@demo.hygeia.test", "HYG-DEMO-001", "female", "1998-08-20", "081234567891", "Jl. Asia Afrika No. 12, Bandung", "Siti Rahma", "081298765433", 1998},
	}

	for _, p := range patients {
		_, err = tx.Exec(ctx, `INSERT INTO patients(user_id,patient_ref,sex,birth_year,birth_date,phone,address,emergency_contact_name,emergency_contact_phone)
SELECT id, $2, $3, $4, $5::date, $6, $7, $8, $9 FROM users WHERE email=$1
ON CONFLICT(user_id) DO UPDATE SET patient_ref=EXCLUDED.patient_ref, sex=EXCLUDED.sex, birth_year=EXCLUDED.birth_year,
birth_date=EXCLUDED.birth_date, phone=EXCLUDED.phone, address=EXCLUDED.address,
emergency_contact_name=EXCLUDED.emergency_contact_name, emergency_contact_phone=EXCLUDED.emergency_contact_phone`,
			p.email, p.ref, p.sex, p.birthYear, p.birthDate, p.phone, p.addr, p.ecName, p.ecPhone)
		if err != nil {
			return fmt.Errorf("seed patient %s failed: %w", p.email, err)
		}
	}

	// 2. Seed Rich Clinical Workflow (Episode, Lab Requests, Lab Results, Prescriptions) for pasien@hygeia.id
	var patientID, doctorID, clinicOrgID, labOrgID, pharmacyOrgID, labUserID string
	err = tx.QueryRow(ctx, `SELECT p.id::text, u_doc.id::text, o_clinic.id::text, o_lab.id::text, o_pharm.id::text, u_lab.id::text
FROM patients p
JOIN users u_pat ON p.user_id = u_pat.id AND u_pat.email = 'pasien@hygeia.id'
JOIN users u_doc ON u_doc.email = 'dokter@hygeia.id'
JOIN organization_memberships om_doc ON om_doc.user_id = u_doc.id
JOIN organizations o_clinic ON om_doc.organization_id = o_clinic.id
JOIN organizations o_lab ON o_lab.name = 'Laboratorium Sentra Medika' AND o_lab.type = 'laboratory'
JOIN users u_lab ON u_lab.email = 'lab@hygeia.id'
JOIN organizations o_pharm ON o_pharm.name = 'Apotek Sentra Farma' AND o_pharm.type = 'pharmacy'`).
		Scan(&patientID, &doctorID, &clinicOrgID, &labOrgID, &pharmacyOrgID, &labUserID)
	if err != nil {
		return fmt.Errorf("failed looking up relation IDs: %w", err)
	}

	// Active care episode
	var episodeID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM care_episodes WHERE patient_id=$1 AND doctor_id=$2 AND status='active'`, patientID, doctorID).Scan(&episodeID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO care_episodes(patient_id,doctor_id,organization_id,status,accepted_at)
VALUES($1,$2,$3,'active',now()) RETURNING id::text`, patientID, doctorID, clinicOrgID).Scan(&episodeID)
		if err != nil {
			return fmt.Errorf("seed care episode failed: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("check episode failed: %w", err)
	}

	// Lab Request 1: Completed with Results
	var labReq1 string
	err = tx.QueryRow(ctx, `SELECT id::text FROM lab_requests WHERE episode_id=$1 AND test_code='HEMATOLOGI'`, episodeID).Scan(&labReq1)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO lab_requests(episode_id,lab_org_id,test_code,test_name,status,approved_at,completed_at)
VALUES($1,$2,'HEMATOLOGI','Pemeriksaan Darah Lengkap (Hematologi)','completed',now()-interval '2 hours',now()-interval '30 minutes')
RETURNING id::text`, episodeID, labOrgID).Scan(&labReq1)
		if err != nil {
			return fmt.Errorf("seed lab req 1 failed: %w", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO lab_results(lab_request_id,value,unit,reference_range,flag,note,submitted_by)
VALUES($1,'Hb: 13.8 g/dL (Normal), Leukosit: 6.800 /uL (Normal), Trombosit: 240.000 /uL (Normal)','panel','Nilai Standar Lab Sentra','normal','Hasil tes hematologi rutin dalam batas normal. Pasien stabil.',$2)
ON CONFLICT(lab_request_id) DO NOTHING`, labReq1, labUserID)
		if err != nil {
			return fmt.Errorf("seed lab result failed: %w", err)
		}
	}

	// Lab Request 2: Sent (in queue for lab staff to process)
	var labReq2 string
	err = tx.QueryRow(ctx, `SELECT id::text FROM lab_requests WHERE episode_id=$1 AND test_code='LIPID'`, episodeID).Scan(&labReq2)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO lab_requests(episode_id,lab_org_id,test_code,test_name,status,approved_at)
VALUES($1,$2,'LIPID','Pemeriksaan Profil Lipid & Kolesterol Total','sent',now()-interval '1 hour')
RETURNING id::text`, episodeID, labOrgID).Scan(&labReq2)
		if err != nil {
			return fmt.Errorf("seed lab req 2 failed: %w", err)
		}
	}

	// Prescription 1: Ready at pharmacy
	var rx1 string
	err = tx.QueryRow(ctx, `SELECT id::text FROM prescriptions WHERE episode_id=$1 AND status='ready'`, episodeID).Scan(&rx1)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO prescriptions(episode_id,pharmacy_org_id,status,approved_at)
VALUES($1,$2,'ready',now()-interval '1 hour') RETURNING id::text`, episodeID, pharmacyOrgID).Scan(&rx1)
		if err != nil {
			return fmt.Errorf("seed rx1 failed: %w", err)
		}
		meds := []struct {
			pos  int
			name string
			dos  string
			inst string
			qty  int
		}{
			{1, "Amoxicillin 500mg", "500 mg", "3x sehari 1 kaplet sesudah makan (habiskan)", 15},
			{2, "Paracetamol 500mg", "500 mg", "3x sehari 1 tablet bila demam/nyeri", 10},
			{3, "Vitamin C 500mg", "500 mg", "1x sehari 1 tablet sesudah makan", 10},
		}
		for _, m := range meds {
			_, err = tx.Exec(ctx, `INSERT INTO prescription_items(prescription_id,position,medication_name,dosage,instructions,quantity)
VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(prescription_id,position) DO NOTHING`, rx1, m.pos, m.name, m.dos, m.inst, m.qty)
			if err != nil {
				return fmt.Errorf("seed rx1 items failed: %w", err)
			}
		}
	}

	// Prescription 2: Pending patient approval (so user/judge can experience approving a prescription!)
	var rx2 string
	err = tx.QueryRow(ctx, `SELECT id::text FROM prescriptions WHERE episode_id=$1 AND status='pending_patient_approval'`, episodeID).Scan(&rx2)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `INSERT INTO prescriptions(episode_id,status)
VALUES($1,'pending_patient_approval') RETURNING id::text`, episodeID).Scan(&rx2)
		if err != nil {
			return fmt.Errorf("seed rx2 failed: %w", err)
		}
		_, err = tx.Exec(ctx, `INSERT INTO prescription_items(prescription_id,position,medication_name,dosage,instructions,quantity)
VALUES($1,1,'Cetirizine 10mg','10 mg','1x sehari malam sebelum tidur bila gatal/alergi',5)
ON CONFLICT(prescription_id,position) DO NOTHING`, rx2)
		if err != nil {
			return fmt.Errorf("seed rx2 items failed: %w", err)
		}
	}

	// Generate audit head if not exists
	var hasHead bool
	_ = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM audit_chain_heads WHERE chain_id=$1)`, patientID).Scan(&hasHead)
	if !hasHead {
		rawHash := make([]byte, 32)
		_, _ = rand.Read(rawHash)
		_, _ = tx.Exec(ctx, `INSERT INTO audit_chain_heads(chain_id,last_seq,last_hash) VALUES($1,1,$2) ON CONFLICT(chain_id) DO NOTHING`, patientID, rawHash)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("seed commit failed: %w", err)
	}

	fmt.Println("===================================================================")
	fmt.Println("🚀 SEED SUKSES! AKUN DEMO RESMI (@hygeia.id) SIAP DIGUNAKAN:")
	fmt.Printf("Password untuk SEMUA akun: %s\n\n", pwd)
	fmt.Println("1. Pasien     : pasien@hygeia.id (Aulia Rahman)")
	fmt.Println("2. Dokter     : dokter@hygeia.id (dr. Maya Sp.PD - Klinik Pratama Sentra)")
	fmt.Println("3. Staf Lab   : lab@hygeia.id    (Analis Lab - Laboratorium Sentra Medika)")
	fmt.Println("4. Apotek     : apotek@hygeia.id (Apoteker - Apotek Sentra Farma)")
	fmt.Println("5. Admin      : admin@hygeia.id  (Administrator Sistem Hygeia)")
	fmt.Println("===================================================================")
	fmt.Println("Data klinis dummy lengkap (Rekam Medis, Lab Hematologi, Resep) sudah aktif!")
	return nil
}
