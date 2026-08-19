package memorydb_test

import (
    "context"
    "fmt"
    "testing"

    "github.com/untappedtech/conduit/internal/domain"
)

func TestMemoryDB_DeleteBasic(t *testing.T) {
    db := setupMemoryDB()
    ctx := context.Background()

    cols := []domain.ColumnDef{
        {Name: "id", Type: "INTEGER", PK: ptrBool(true), Autoincrement: ptrBool(true)},
        {Name: "name", Type: "TEXT"},
    }

    db.CreateTable(ctx, "items", cols)
    rec, _ := db.Insert(ctx, "items", map[string]any{"name": "alpha"})
    idStr := fmt.Sprintf("%v", rec["id"])

    if err := db.Delete(ctx, "items", idStr); err != nil {
        t.Fatalf("Delete error: %v", err)
    }

    if _, err := db.GetByID(ctx, "items", idStr); err == nil {
        t.Fatalf("expected error after delete")
    }
}
