package service_test

import (
	"context"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
)

func TestDropTable(t *testing.T) {
	api := setupTestAPIService()
	ctx := context.Background()

	cols := []domain.ColumnDef{
		{Name: "id", Type: "INTEGER"},
	}
	if err := api.CreateTable(ctx, "temp", cols); err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}

	if err := api.DropTable(ctx, "temp"); err != nil {
		t.Fatalf("DropTable error: %v", err)
	}

	if _, err := api.GetSchema(ctx, "temp"); err == nil {
		t.Fatalf("expected error after drop, got nil")
	}
}
