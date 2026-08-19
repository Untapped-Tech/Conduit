package memorydb_test

import (
    "github.com/untappedtech/conduit/internal/db/impl"
)

func setupMemoryDB() *impl.MemoryDB {
    return impl.NewMemoryDB().(*impl.MemoryDB)
}

func ptrBool(v bool) *bool { return &v }
