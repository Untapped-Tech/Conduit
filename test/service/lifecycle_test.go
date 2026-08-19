package service_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
)

func TestInsertUpdateDeleteLifecycle(t *testing.T) {
	api := setupTestAPIService()
	ctx := context.Background()

	isPK := true
	isAuto := true
	cols := []domain.ColumnDef{
		{Name: "id", Type: "INTEGER", PK: &isPK, Autoincrement: &isAuto},
		{Name: "name", Type: "TEXT"},
		{Name: "players", Type: "INTEGER"},
	}

	if err := api.CreateTable(ctx, "sports", cols); err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}

	rec, err := api.Insert(ctx, "sports", map[string]any{"name": "Golf", "players": 4})
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}
	if rec["id"] == nil {
		t.Fatalf("expected autoincrement id")
	}

	idStr := fmt.Sprintf("%v", rec["id"])

	updated, err := api.Update(ctx, "sports", idStr, map[string]any{"players": 2})
	if err != nil {
		t.Fatalf("Update error: %v", err)
	}
	if updated["players"] != 2 {
		t.Fatalf("expected players=2, got %v", updated["players"])
	}

	got, err := api.GetByID(ctx, "sports", idStr)
	if err != nil {
		t.Fatalf("GetByID error: %v", err)
	}
	if got["players"] != 2 {
		t.Fatalf("expected players=2, got %v", got["players"])
	}

	if err := api.Delete(ctx, "sports", idStr); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
	if _, err := api.GetByID(ctx, "sports", idStr); err == nil {
		t.Fatalf("expected error after delete, got nil")
	}
}
