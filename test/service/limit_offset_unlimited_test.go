package service_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
)

func TestLimitOffsetListing_UnlimitedMode(t *testing.T) {
	api := setupTestAPIServiceWithLimit(0)
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

	list0, err := api.List(ctx, "items", 0, 0)
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(list0) != 10 {
		t.Fatalf("expected 10 items, got %d", len(list0))
	}
}
