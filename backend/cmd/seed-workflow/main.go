package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/config"
	"github.com/mrizalbasri/Hygeia/backend/internal/database"
	"github.com/mrizalbasri/Hygeia/backend/internal/demo"
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
	if err := demo.CheckEnvironment(os.Getenv("APP_ENV")); err != nil {
		return err
	}
	key, err := hex.DecodeString(os.Getenv("AUDIT_HMAC_KEY"))
	if err != nil || len(key) != 32 {
		return errors.New("AUDIT_HMAC_KEY must contain 64 hexadecimal characters")
	}
	writer, err := audit.NewWriter(key)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()
	created, err := demo.SeedWorkflow(ctx, pool, writer, os.Getenv("APP_ENV"), os.Getenv("DEMO_PASSWORD"))
	if err != nil {
		return err
	}
	if created {
		fmt.Println("Synthetic workflow created; not medical data.")
	} else {
		fmt.Println("Synthetic workflow already exists; passwords and progress preserved.")
	}
	fmt.Println("Patient code: " + demo.PatientRef)
	for _, role := range []string{"patient", "doctor", "lab", "pharmacy", "admin"} {
		fmt.Printf("%s: pilot.%s@demo.hygeia.test\n", role, role)
	}
	return nil
}
