package memorydb_test

import (
    "context"
    "sync"
    "testing"

    "github.com/untappedtech/conduit/internal/domain"
)

func TestMemoryDB_ConcurrentInsert(t *testing.T) {
    db := setupMemoryDB()
    ctx := context.Background()

    cols := []domain.ColumnDef{
        {Name: "id", Type: "INTEGER", PK: ptrBool(true), Autoincrement: ptrBool(true)},
        {Name: "name", Type: "TEXT"},
    }

    db.CreateTable(ctx, "items", cols)

    var wg sync.WaitGroup
    for i := 0; i < 50; i++ {
        wg.Add(1)
        go func(i int) {
            defer wg.Done()
            db.Insert(ctx, "items", map[string]any{"name": "n"})
        }(i)
    }
    wg.Wait()

    rows, _ := db.List(ctx, "items", 100, 0)
    if len(rows) != 50 {
        t.Fatalf("expected 50 rows")
    }
}
