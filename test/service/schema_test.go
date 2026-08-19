package service_test

import (
	"context"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
)

func TestTableCreationAndSchemaRetrieval(t *testing.T) {
	api := setupTestAPIService()
	ctx := context.Background()

	isNullable := true
	isPK := true
	isAuto := true
	defaultVal := "CURRENT_TIMESTAMP"

	cols := []domain.ColumnDef{
		{Name: "id", Type: "INTEGER", PK: &isPK, Autoincrement: &isAuto},
		{Name: "title", Type: "TEXT", Nullable: &isNullable, Default: &defaultVal},
	}

	if err := api.CreateTable(ctx, "books", cols); err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}

	got, err := api.GetSchema(ctx, "books")
	if err != nil {
		t.Fatalf("GetSchema error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(got))
	}
	if got[0].PK == nil || !*got[0].PK {
		t.Fatalf("expected first column to be PK")
	}
	if got[1].Default == nil || *got[1].Default != defaultVal {
		t.Fatalf("expected default %q, got %v", defaultVal, got[1].Default)
	}
}
