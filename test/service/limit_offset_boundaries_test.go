package service_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
)

func TestLimitOffsetBoundaries(t *testing.T) {
	api := setupTestAPIServiceWithLimit(5)
	ctx := context.Background()

	isPK := true
	isAuto := true
	cols := []domain.ColumnDef{
		{Name: "id", Type: "INTEGER", PK: &isPK, Autoincrement: &isAuto},
		{Name: "name", Type: "TEXT"},
	}

	if err := api.CreateTable(ctx, "bounds", cols); err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}

	for i := 0; i < 5; i++ {
		api.Insert(ctx, "bounds", map[string]any{"name": "n" + strconv.Itoa(i)})
	}

	listOffset, _ := api.List(ctx, "bounds", 5, 10)
	if len(listOffset) != 0 {
		t.Fatalf("expected 0 items, got %d", len(listOffset))
	}
}
