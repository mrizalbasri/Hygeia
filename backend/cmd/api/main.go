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
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mrizalbasri/Hygeia/backend/internal/admin"
	"github.com/mrizalbasri/Hygeia/backend/internal/audit"
	"github.com/mrizalbasri/Hygeia/backend/internal/auth"
	"github.com/mrizalbasri/Hygeia/backend/internal/breakglass"
	"github.com/mrizalbasri/Hygeia/backend/internal/care"
	"github.com/mrizalbasri/Hygeia/backend/internal/carepass"
	"github.com/mrizalbasri/Hygeia/backend/internal/config"
	"github.com/mrizalbasri/Hygeia/backend/internal/consent"
	"github.com/mrizalbasri/Hygeia/backend/internal/database"
	"github.com/mrizalbasri/Hygeia/backend/internal/httpapi"
	"github.com/mrizalbasri/Hygeia/backend/internal/laboratory"
	"github.com/mrizalbasri/Hygeia/backend/internal/notification"
	"github.com/mrizalbasri/Hygeia/backend/internal/patientprofile"
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
	trustedProxies, err := auth.ParseTrustedProxyCIDRs(os.Getenv("TRUSTED_PROXY_CIDRS"))
	if err != nil {
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
	var fallbackKeys [][]byte
	if prev := os.Getenv("AUDIT_HMAC_KEY_PREVIOUS"); prev != "" {
		for _, value := range strings.Split(prev, ",") {
			pk, err := hex.DecodeString(strings.TrimSpace(value))
			if err != nil || len(pk) != 32 {
				return errors.New("AUDIT_HMAC_KEY_PREVIOUS must contain comma-separated 64-character hexadecimal keys")
			}
			if hmac.Equal(pk, csrfKey) {
				return errors.New("historical audit keys must be different from SESSION_SECRET")
			}
			fallbackKeys = append(fallbackKeys, pk)
		}
	}
	auditWriter, err := audit.NewWriter(auditKey, fallbackKeys...)
	if err != nil {
		return err
	}
	appEnv := os.Getenv("APP_ENV")
	if appEnv != "" && appEnv != "development" && appEnv != "test" && appEnv != "production" {
		return errors.New("APP_ENV must be development, test, or production")
	}
	secure := appEnv == "production"
	provider := os.Getenv("AUTH_PROVIDER")
	if provider == "" {
		provider = "local"
	}
	if provider != "local" && provider != "clerk" {
		return errors.New("AUTH_PROVIDER must be local or clerk")
	}
	if provider == "clerk" && secure {
		pk := os.Getenv("CLERK_PUBLISHABLE_KEY")
		if pk == "" {
			pk = os.Getenv("VITE_CLERK_PUBLISHABLE_KEY")
		}
		if !strings.HasPrefix(pk, "pk_live_") || !strings.HasPrefix(os.Getenv("CLERK_SECRET_KEY"), "sk_live_") {
			return errors.New("production requires Clerk live instance keys")
		}
		for _, origin := range strings.Split(os.Getenv("CLERK_AUTHORIZED_PARTIES"), ",") {
			if !strings.HasPrefix(strings.TrimSpace(origin), "https://") {
				return errors.New("production Clerk authorized parties must use HTTPS")
			}
		}
	}
	shareBase, err := carepass.ShareBase(os.Getenv("PUBLIC_APP_URL"), secure)
	if err != nil {
		return err
	}
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

	// Background session cleanup every hour.
	go func() {
		authStore := auth.Store{DB: pool}
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cleanupCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
				n, err := authStore.DeleteExpired(cleanupCtx)
				cancel()
				if err != nil {
					logger.Error("session cleanup failed", "error", err)
				} else if n > 0 {
					logger.Info("expired sessions cleaned", "count", n)
				}
			}
		}
	}()

	authHandler, err := auth.NewHandler(auth.Store{DB: pool}, secure, trustedProxies...)
	if err != nil {
		return errors.New("could not initialize authentication")
	}
	frontendURL, _ := url.Parse(shareBase) // ShareBase validated the origin above.
	var trustedOrigins []string
	if frontendURL.Host != "" {
		trustedOrigins = append(trustedOrigins, frontendURL.Host)
	}
	if extraOrigins := os.Getenv("CSRF_TRUSTED_ORIGINS"); extraOrigins != "" {
		for _, o := range strings.Split(extraOrigins, ",") {
			o = strings.TrimSpace(o)
			if u, err := url.Parse(o); err == nil && u.Host != "" {
				trustedOrigins = append(trustedOrigins, u.Host)
			} else if o != "" {
				trustedOrigins = append(trustedOrigins, o)
			}
		}
	}
	authHandler.CSRFTrustedOrigins = trustedOrigins
	var clerkGate *auth.ClerkGate
	if provider == "clerk" {
		publishable := os.Getenv("CLERK_PUBLISHABLE_KEY")
		if publishable == "" {
			publishable = os.Getenv("VITE_CLERK_PUBLISHABLE_KEY")
		}
		clerkGate, err = auth.NewClerkGate(auth.Store{DB: pool}, publishable, os.Getenv("CLERK_SECRET_KEY"), strings.Split(os.Getenv("CLERK_AUTHORIZED_PARTIES"), ","), csrfKey)
		if err != nil {
			return err
		}
		authHandler.UseClerk(clerkGate)
	}
	careHandler := &care.Handler{Service: &care.Service{DB: pool, Audit: auditWriter}, Logger: logger}
	labHandler := &laboratory.Handler{Service: &laboratory.Service{DB: pool, Audit: auditWriter}, Logger: logger}
	consentHandler := &consent.Handler{Service: &consent.Service{DB: pool, Audit: auditWriter}, Logger: logger}
	prescriptionHandler := &prescription.Handler{Service: &prescription.Service{DB: pool, Audit: auditWriter}, Logger: logger}
	adminHandler := &admin.Handler{Service: &admin.Service{DB: pool}}
	if clerkGate != nil {
		adminHandler.Service.ClerkUsers = clerkGate.LookupUser
	}
	notificationHandler := &notification.Handler{Service: &notification.Service{DB: pool}, Shutdown: ctx}
	breakGlassHandler := &breakglass.Handler{Service: &breakglass.Service{DB: pool, Audit: auditWriter}}
	carePassHandler := &carepass.Handler{Service: &carepass.Service{DB: pool, Audit: auditWriter, ShareBaseURL: shareBase}}
	profileHandler := &patientprofile.Handler{Service: &patientprofile.Service{DB: pool, Audit: auditWriter}}
	var api http.Handler = httpapi.NewAuthenticatedHandler(logger, pool.Ping, authHandler, csrfKey, secure, careHandler.Register, labHandler.Register, consentHandler.Register, prescriptionHandler.Register, adminHandler.Register, notificationHandler.Register, breakGlassHandler.Register, carePassHandler.Register, profileHandler.Register)
	if clerkGate != nil {
		api = clerkGate.Wrap(api)
	}

	server := &http.Server{
		Addr:              net.JoinHostPort("", port),
		Handler:           api,
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
