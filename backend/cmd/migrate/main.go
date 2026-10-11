package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/mrizalbasri/Hygeia/backend/db"
	"github.com/mrizalbasri/Hygeia/backend/internal/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "version") {
		return errors.New("usage: go run ./cmd/migrate [up|version]")
	}
	if err := config.LoadEnv(); err != nil {
		return err
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		return errors.New("DATABASE_URL is required")
	}
	if !strings.HasPrefix(url, "postgres://") && !strings.HasPrefix(url, "postgresql://") {
		return errors.New("DATABASE_URL must be a PostgreSQL URL")
	}
	url = "pgx5://" + strings.SplitN(url, "://", 2)[1]
	source, err := iofs.New(db.Migrations, "migrations")
	if err != nil {
		return errors.New("could not load embedded migrations")
	}
	m, err := migrate.NewWithSourceInstance("iofs", source, url)
	if err != nil {
		return errors.New("could not initialize migrations; check database configuration")
	}
	defer m.Close()
	if os.Args[1] == "up" {
		err = m.Up()
		if err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return errors.New("migration failed; inspect schema_migrations and database logs before retrying")
		}
		fmt.Println("Migrations applied (or already up to date).")
	}
	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("read migration version: %w", err)
	}
	fmt.Printf("Schema version: %d; dirty: %t\n", version, dirty)
	return nil
}
