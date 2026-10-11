package main

import (
	"context"
	"crypto/hmac"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/mrizalbasri/Hygeia/backend/internal/admin"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/breakglass"
	"github.com/mrizalbasri/Hygeia/backend/internal/care"
	"github.com/mrizalbasri/Hygeia/backend/internal/config"
	"github.com/mrizalbasri/Hygeia/backend/internal/consent"
	"github.com/mrizalbasri/Hygeia/backend/internal/database"
	"github.com/mrizalbasri/Hygeia/backend/internal/httpapi"
	"github.com/mrizalbasri/Hygeia/backend/internal/laboratory"
	"github.com/mrizalbasri/Hygeia/backend/internal/notification"
	"github.com/mrizalbasri/Hygeia/backend/internal/prescription"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	if err := config.LoadEnv(); err != nil {
		return err
	}
	csrfKey, err := hex.DecodeString(os.Getenv("SESSION_SECRET"))
	if err != nil || len(csrfKey) != 32 {
		return errors.New("SESSION_SECRET must contain 64 hexadecimal characters")
	}
	auditKey, err := hex.DecodeString(os.Getenv("AUDIT_HMAC_KEY"))
	if err != nil || len(auditKey) != 32 {
		return errors.New("AUDIT_HMAC_KEY must contain 64 hexadecimal characters")
	}
	if hmac.Equal(auditKey, csrfKey) {
		return errors.New("AUDIT_HMAC_KEY must be different from SESSION_SECRET")
	}
	auditWriter, err := audit.NewWriter(auditKey)
	if err != nil {
		return err
	}
	appEnv := os.Getenv("APP_ENV")
	if appEnv != "" && appEnv != "development" && appEnv != "test" && appEnv != "production" {
		return errors.New("APP_ENV must be development, test, or production")
	}
	secure := appEnv == "production"
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	value, err := strconv.Atoi(port)
	if err != nil || value < 1 || value > 65535 {
		return fmt.Errorf("PORT must be a number between 1 and 65535")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.Open(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()
	logger.Info("connected to PostgreSQL")
	authHandler, err := auth.NewHandler(auth.Store{DB: pool}, secure)
	if err != nil {
		return errors.New("could not initialize authentication")
	}
	careHandler := &care.Handler{Service: &care.Service{DB: pool, Audit: auditWriter}, Logger: logger}
	labHandler := &laboratory.Handler{Service: &laboratory.Service{DB: pool, Audit: auditWriter}, Logger: logger}
	consentHandler := &consent.Handler{Service: &consent.Service{DB: pool, Audit: auditWriter}, Logger: logger}
	prescriptionHandler := &prescription.Handler{Service: &prescription.Service{DB: pool, Audit: auditWriter}, Logger: logger}
	adminHandler := &admin.Handler{Service: &admin.Service{DB: pool}}
	notificationHandler := &notification.Handler{Service: &notification.Service{DB: pool}}
	breakGlassHandler := &breakglass.Handler{Service: &breakglass.Service{DB: pool, Audit: auditWriter}}

	server := &http.Server{
		Addr:              net.JoinHostPort("", port),
		Handler:           httpapi.NewAuthenticatedHandler(logger, pool.Ping, authHandler, csrfKey, secure, careHandler.Register, labHandler.Register, consentHandler.Register, prescriptionHandler.Register, adminHandler.Register, notificationHandler.Register, breakGlassHandler.Register),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting hygeia API", "address", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		return err
	case <-ctx.Done():
		stop()
	}

	logger.Info("shutting down hygeia API")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-serverErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
