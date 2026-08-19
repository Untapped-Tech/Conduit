package memorydb_test

import (
    "context"
    "fmt"
    "testing"

    "github.com/untappedtech/conduit/internal/domain"
)

func TestMemoryDB_UpdateBasic(t *testing.T) {
    db := setupMemoryDB()
    ctx := context.Background()

    cols := []domain.ColumnDef{
        {Name: "id", Type: "INTEGER", PK: ptrBool(true), Autoincrement: ptrBool(true)},
        {Name: "value", Type: "INTEGER"},
    }

    db.CreateTable(ctx, "nums", cols)
    rec, _ := db.Insert(ctx, "nums", map[string]any{"value": 10})
    idStr := fmt.Sprintf("%v", rec["id"])

    updated, err := db.Update(ctx, "nums", idStr, map[string]any{"value": 20})
    if err != nil {
        t.Fatalf("Update error: %v", err)
    }
    if updated["value"] != 20 {
        t.Fatalf("expected updated value=20")
    }
}
