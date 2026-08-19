package memorydb_test

import (
	"context"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
)

func TestMemoryDB_SchemaBasic(t *testing.T) {
	db := setupMemoryDB()
	ctx := context.Background()

	cols := []domain.ColumnDef{
		{Name: "id", Type: "INTEGER"},
		{Name: "name", Type: "TEXT"},
	}

	if err := db.CreateTable(ctx, "alpha", cols); err != nil {
		t.Fatalf("CreateTable error: %v", err)
	}

	schema, err := db.Schema(ctx, "alpha")
	if err != nil {
		t.Fatalf("GetSchema error: %v", err)
	}

	if len(schema) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(schema))
	}

	if schema[0].Name != "id" || schema[1].Name != "name" {
		t.Fatalf("unexpected schema: %+v", schema)
	}
}
