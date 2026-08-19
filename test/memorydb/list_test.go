package memorydb_test

import (
    "context"
    "strconv"
    "testing"

    "github.com/untappedtech/conduit/internal/domain"
)

func TestMemoryDB_ListBasic(t *testing.T) {
    db := setupMemoryDB()
    ctx := context.Background()

    cols := []domain.ColumnDef{
        {Name: "id", Type: "INTEGER", PK: ptrBool(true), Autoincrement: ptrBool(true)},
        {Name: "name", Type: "TEXT"},
    }

    db.CreateTable(ctx, "items", cols)

    for i := 0; i < 5; i++ {
        db.Insert(ctx, "items", map[string]any{"name": "n" + strconv.Itoa(i)})
    }

    rows, err := db.List(ctx, "items", 5, 0)
    if err != nil {
        t.Fatalf("List error: %v", err)
    }
    if len(rows) != 5 {
        t.Fatalf("expected 5 rows")
    }
}
