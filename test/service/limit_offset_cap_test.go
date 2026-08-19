package service_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
)

func TestLimitOffsetListing_ServerCap(t *testing.T) {
	api := setupTestAPIServiceWithLimit(3)
	ctx := context.Background()

	isPK := true
	isAuto := true
	cols := []domain.ColumnDef{
		{Name: "id", Type: "INTEGER", PK: &isPK, Autoincrement: &isAuto},
		{Name: "name", Type: "TEXT"},
	}

	if err := api.CreateTable(ctx, "items", cols); err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}

	for i := 0; i < 10; i++ {
		if _, err := api.Insert(ctx, "items", map[string]any{"name": "item" + strconv.Itoa(i)}); err != nil {
			t.Fatalf("Insert error: %v", err)
		}
	}

	listCap, _ := api.List(ctx, "items", 10, 0)
	if len(listCap) != 3 {
		t.Fatalf("expected 3 items, got %d", len(listCap))
	}
}
