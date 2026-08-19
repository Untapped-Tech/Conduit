package memorydb_test

import (
    "context"
    "testing"

    "github.com/untappedtech/conduit/internal/domain"
)

func TestMemoryDB_InsertBasic(t *testing.T) {
    db := setupMemoryDB()
    ctx := context.Background()

    cols := []domain.ColumnDef{
        {Name: "id", Type: "INTEGER", PK: ptrBool(true), Autoincrement: ptrBool(true)},
        {Name: "name", Type: "TEXT"},
    }

    if err := db.CreateTable(ctx, "items", cols); err != nil {
        t.Fatalf("CreateTable error: %v", err)
    }

    rec, err := db.Insert(ctx, "items", map[string]any{"name": "alpha"})
    if err != nil {
        t.Fatalf("Insert error: %v", err)
    }

    if rec["id"] == nil {
        t.Fatalf("expected autoincrement id")
    }
}
