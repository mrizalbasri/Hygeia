package demo

import (
	"context"
	"testing"
)

func TestProductionSeedRejectedBeforeDatabaseAccess(t *testing.T) {
	for _, environment := range []string{"production", "", "Development", "prod"} {
		if _, err := SeedWorkflow(context.Background(), nil, nil, environment, ""); err == nil {
			t.Fatalf("accepted %q", environment)
		}
	}
}
