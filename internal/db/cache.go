package db

import (
	"container/list"
	"context"
	"sync"

	"github.com/untappedtech/conduit/internal/domain"
)

type schemaCacheEntry struct {
	tableName string
	columns   []domain.ColumnDef
}

type CachedDatabase struct {
	domain.DatabaseDriver
	mutex       sync.RWMutex
	capacity    int
	schemaMap   map[string]*list.Element
	lruList     *list.List
	tablesCache []string
}

func NewCachedDatabase(driver domain.DatabaseDriver, capacity int) domain.DatabaseDriver {
	if capacity <= 0 {
		capacity = 256
	}
	return &CachedDatabase{
		DatabaseDriver: driver,
		capacity:       capacity,
		schemaMap:      make(map[string]*list.Element),
		lruList:        list.New(),
	}
}

func (cachedDB *CachedDatabase) Schema(ctx context.Context, tableName string) ([]domain.ColumnDef, error) {
	cachedDB.mutex.Lock()
	if element, exists := cachedDB.schemaMap[tableName]; exists {
		cachedDB.lruList.MoveToFront(element)
		cachedDB.mutex.Unlock()
		return element.Value.(*schemaCacheEntry).columns, nil
	}
	cachedDB.mutex.Unlock()

	columns, err := cachedDB.DatabaseDriver.Schema(ctx, tableName)
	if err != nil {
		return nil, err
	}

	cachedDB.mutex.Lock()
	if cachedDB.lruList.Len() >= cachedDB.capacity {
		oldest := cachedDB.lruList.Back()
		if oldest != nil {
			cachedDB.lruList.Remove(oldest)
			delete(cachedDB.schemaMap, oldest.Value.(*schemaCacheEntry).tableName)
		}
	}

	newEntry := &schemaCacheEntry{tableName: tableName, columns: columns}
	newElem := cachedDB.lruList.PushFront(newEntry)
	cachedDB.schemaMap[tableName] = newElem
	cachedDB.mutex.Unlock()

	return columns, nil
}

func (cachedDB *CachedDatabase) ListTables(ctx context.Context) ([]string, error) {
	cachedDB.mutex.RLock()
	if cachedDB.tablesCache != nil {
		resultCopy := make([]string, len(cachedDB.tablesCache))
		copy(resultCopy, cachedDB.tablesCache)
		cachedDB.mutex.RUnlock()
		return resultCopy, nil
	}
	cachedDB.mutex.RUnlock()

	tablesList, err := cachedDB.DatabaseDriver.ListTables(ctx)
	if err != nil {
		return nil, err
	}

	cachedDB.mutex.Lock()
	cachedDB.tablesCache = tablesList
	cachedDB.mutex.Unlock()

	return tablesList, nil
}

func (cachedDB *CachedDatabase) CreateTable(ctx context.Context, tableName string, columns []domain.ColumnDef) error {
	err := cachedDB.DatabaseDriver.CreateTable(ctx, tableName, columns)
	if err != nil {
		return err
	}
	cachedDB.Invalidate()
	return nil
}

func (cachedDB *CachedDatabase) DropTable(ctx context.Context, tableName string) error {
	err := cachedDB.DatabaseDriver.DropTable(ctx, tableName)
	if err != nil {
		return err
	}
	cachedDB.Invalidate()
	return nil
}

func (cachedDB *CachedDatabase) Invalidate() {
	cachedDB.mutex.Lock()
	defer cachedDB.mutex.Unlock()
	cachedDB.schemaMap = make(map[string]*list.Element)
	cachedDB.lruList.Init()
	cachedDB.tablesCache = nil
}
