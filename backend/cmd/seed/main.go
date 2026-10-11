package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

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
	if os.Getenv("APP_ENV") != "development" && os.Getenv("APP_ENV") != "test" {
		return errors.New("demo seed is only permitted with APP_ENV=development or test")
	}
	hash, err := auth.HashPassword(os.Getenv("DEMO_PASSWORD"))
	if err != nil {
		return errors.New("DEMO_PASSWORD must contain 12 to 1024 bytes")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	accounts := []struct{ email, name, role, org, orgType string }{
		{"aulia@demo.hygeia.test", "Aulia Rahman", "patient", "", ""},
		{"maya@demo.hygeia.test", "Dr. Maya", "doctor", "Klinik Sentra", "clinic"},
		{"budi@demo.hygeia.test", "Dr. Budi", "doctor", "Klinik Harapan", "clinic"},
		{"rina@demo.hygeia.test", "Dr. Rina", "doctor", "IGD Harapan", "clinic"},
		{"lab@demo.hygeia.test", "Staf Lab Sentra", "lab", "Lab Sentra", "laboratory"},
		{"apotek@demo.hygeia.test", "Staf Apotek Sentra", "pharmacy", "Apotek Sentra", "pharmacy"},
		{"admin@demo.hygeia.test", "Admin Hygeia", "admin", "", ""},
	}
	for _, account := range accounts {
		_, err := tx.Exec(ctx, `INSERT INTO users(email,password_hash,full_name,role) VALUES($1,$2,$3,$4) ON CONFLICT(email) DO NOTHING`, account.email, hash, account.name, account.role)
		if err != nil {
			return errors.New("seed users failed; run migrations first")
		}
		if account.org != "" {
			_, err = tx.Exec(ctx, `INSERT INTO organizations(name,type) VALUES($1,$2) ON CONFLICT(type,name) DO NOTHING`, account.org, account.orgType)
			if err != nil {
				return errors.New("seed organizations failed")
			}
			_, err = tx.Exec(ctx, `INSERT INTO organization_memberships(user_id,organization_id,staff_role)
SELECT u.id,o.id,u.role FROM users u, organizations o WHERE u.email=$1 AND o.name=$2 AND o.type=$3
ON CONFLICT(user_id) DO NOTHING`, account.email, account.org, account.orgType)
			if err != nil {
				return errors.New("seed memberships failed")
			}
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO patients(user_id,patient_ref,sex,birth_year) SELECT id,'HYG-DEMO-001','female',1998 FROM users WHERE email='aulia@demo.hygeia.test' ON CONFLICT(user_id) DO NOTHING`)
	if err != nil {
		return errors.New("seed patient failed")
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.New("seed commit failed")
	}
	fmt.Println("Synthetic demo accounts ready. Existing passwords and accounts were preserved.")
	for _, account := range accounts {
		fmt.Printf("%s: %s\n", account.role, account.email)
	}
	return nil
}
