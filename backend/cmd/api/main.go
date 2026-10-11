package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/config"
	"github.com/mrizalbasri/Hygeia/backend/internal/database"
	"github.com/mrizalbasri/Hygeia/backend/internal/httpapi"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
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

	server := &http.Server{
		Addr:              net.JoinHostPort("", port),
		Handler:           httpapi.NewAuthenticatedHandler(logger, pool.Ping, authHandler, csrfKey, secure),
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
