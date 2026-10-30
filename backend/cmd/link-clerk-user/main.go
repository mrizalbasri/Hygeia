// Operator-only bootstrap/link command. Never exposed as an HTTP endpoint.
package main

import (
	"context"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/config"
	"github.com/mrizalbasri/Hygeia/backend/internal/database"
	"os"
	"strings"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	external := flag.String("clerk-user-id", "", "Existing verified Clerk user ID")
	local := flag.String("local-user-id", "", "Existing local user UUID (email must match)")
	bootstrap := flag.Bool("bootstrap-admin", false, "Explicitly create the first Clerk-linked administrator; refuses if another exists")
	flag.Parse()
	if *external == "" || (*local == "" && !*bootstrap) || (*local != "" && *bootstrap) {
		return errors.New("use --clerk-user-id with either --local-user-id or --bootstrap-admin")
	}
	if err := config.LoadEnv(); err != nil {
		return err
	}
	key, err := hex.DecodeString(os.Getenv("SESSION_SECRET"))
	if err != nil {
		return errors.New("invalid SESSION_SECRET")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return errors.New("could not connect to local database")
	}
	defer pool.Close()
	pk := os.Getenv("CLERK_PUBLISHABLE_KEY")
	if pk == "" {
		pk = os.Getenv("VITE_CLERK_PUBLISHABLE_KEY")
	}
	gate, err := auth.NewClerkGate(auth.Store{DB: pool}, pk, os.Getenv("CLERK_SECRET_KEY"), strings.Split(os.Getenv("CLERK_AUTHORIZED_PARTIES"), ","), key)
	if err != nil {
		return err
	}
	identity, err := gate.LookupUser(ctx, *external)
	if err != nil {
		return errors.New("Clerk user unavailable, disabled, or primary email not verified")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(73194729)`); err != nil {
		return err
	}
	var id string
	if *bootstrap {
		var exists bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE role='admin' AND clerk_user_id IS NOT NULL)`).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return errors.New("a Clerk-linked administrator already exists; use the administrator workflow")
		}
		var role string
		err = tx.QueryRow(ctx, `SELECT id::text,role FROM users WHERE clerk_user_id=$1 AND disabled_at IS NULL FOR UPDATE`, *external).Scan(&id, &role)
		if errors.Is(err, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `INSERT INTO users(email,password_hash,full_name,role,clerk_user_id) VALUES($1,'!clerk',$2,'admin',$3) RETURNING id::text`, identity.Email, identity.Name, *external).Scan(&id)
		} else if err == nil {
			if role != "patient" {
				return errors.New("bootstrap refuses to replace an existing staff role")
			}
			var patient string
			var used bool
			if err = tx.QueryRow(ctx, `SELECT id::text FROM patients WHERE user_id=$1 FOR UPDATE`, id).Scan(&patient); err != nil {
				return err
			}
			if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM care_episodes WHERE patient_id=$1) OR EXISTS(SELECT 1 FROM audit_events WHERE chain_id=$1)`, patient).Scan(&used); err != nil {
				return err
			}
			if used {
				return errors.New("bootstrap refuses to convert an account with patient history")
			}
			if _, err = tx.Exec(ctx, `DELETE FROM patients WHERE id=$1`, patient); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE users SET role='admin' WHERE id=$1`, id)
		}
	} else {
		err = tx.QueryRow(ctx, `UPDATE users SET clerk_user_id=$2 WHERE id=$1 AND email=$3 AND disabled_at IS NULL AND (clerk_user_id IS NULL OR clerk_user_id=$2) RETURNING id::text`, *local, *external, identity.Email).Scan(&id)
	}
	if err != nil {
		return errors.New("identity link refused; check IDs, verified email, and existing account conflicts")
	}
	// Existing sessions are invalidated when an operator changes the identity link.
	if _, err = tx.Exec(ctx, `DELETE FROM sessions WHERE user_id=$1`, id); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Println("Clerk identity linked. Activate MFA and sign in again before accessing staff data.")
	return nil
}
