package database

import (
	"context"
	"strings"
	"testing"
)

func TestInvalidConfigurationDoesNotLeakCredentials(t *testing.T) {
	for _, value := range []string{"", "postgres://user:private-password@localhost:invalid/db"} {
		pool, err := Open(context.Background(), value)
		if err == nil || pool != nil {
			t.Fatal("expected invalid configuration to fail")
		}
		if strings.Contains(err.Error(), "private-password") {
			t.Fatal("error exposed database credentials")
		}
	}
}
