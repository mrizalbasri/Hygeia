package main

import (
	"io"
	"log/slog"
	"strings"
	"testing"
)

func TestInvalidTrustedProxyConfigStopsStartup(t *testing.T) {
	t.Setenv("TRUSTED_PROXY_CIDRS", "not-a-cidr")
	t.Setenv("DATABASE_URL", "invalid-database-must-not-be-opened")
	err := run(slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err == nil || !strings.Contains(err.Error(), "TRUSTED_PROXY_CIDRS") {
		t.Fatalf("wanted proxy configuration error before database access, got %v", err)
	}
}
