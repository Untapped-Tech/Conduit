package service_test

import (
	"context"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
)

func TestNonAddressableTableDetection(t *testing.T) {
	api := setupTestAPIService()
	ctx := context.Background()

	cols := []domain.ColumnDef{
		{Name: "sensor_name", Type: "TEXT"},
		{Name: "reading", Type: "REAL"},
	}

	if err := api.CreateTable(ctx, "telemetry", cols); err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}

	if _, err := api.Insert(ctx, "telemetry", map[string]any{"sensor_name": "temp", "reading": 23.4}); err != nil {
		t.Fatalf("Insert error: %v", err)
	}

	if _, err := api.GetByID(ctx, "telemetry", "1"); err != domain.ErrPrimaryKeyMissing {
		t.Fatalf("expected ErrPrimaryKeyMissing, got %v", err)
	}
}

func TestPrimaryKeyEdgeCases(t *testing.T) {
	api := setupTestAPIService()
	ctx := context.Background()

	cols := []domain.ColumnDef{
		{Name: "id", Type: "INTEGER"},
	}
	if err := api.CreateTable(ctx, "nopk", cols); err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}
	if _, err := api.GetByID(ctx, "nopk", "1"); err != domain.ErrPrimaryKeyMissing {
		t.Fatalf("expected ErrPrimaryKeyMissing, got %v", err)
	}
}
