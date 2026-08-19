package domain

import (
	"context"
	"encoding/xml"
	"errors"
)

var (
	ErrPrimaryKeyMissing = errors.New("table does not expose a single primary key column and is non-addressable by ID")
)

type ColumnDef struct {
	Name          string  `json:"name" yaml:"name" xml:"name" toml:"name"`
	Type          string  `json:"type" yaml:"type" xml:"type" toml:"type"`
	Nullable      *bool   `json:"nullable,omitempty" yaml:"nullable,omitempty" xml:"nullable,omitempty" toml:"nullable,omitempty"`
	Unique        *bool   `json:"unique,omitempty" yaml:"unique,omitempty" xml:"unique,omitempty" toml:"unique,omitempty"`
	Default       *string `json:"default,omitempty" yaml:"default,omitempty" xml:"default,omitempty" toml:"default,omitempty"`
	PK            *bool   `json:"pk,omitempty" yaml:"pk,omitempty" xml:"pk,omitempty" toml:"pk,omitempty"`
	Autoincrement *bool   `json:"autoincrement,omitempty" yaml:"autoincrement,omitempty" xml:"autoincrement,omitempty" toml:"autoincrement,omitempty"`
	CID           *int    `json:"cid,omitempty" yaml:"cid,omitempty" xml:"cid,omitempty" toml:"cid,omitempty"`
}

type XMLSchema struct {
	XMLName xml.Name    `xml:"schema"`
	Columns []ColumnDef `xml:"column"`
}

type XMLTableList struct {
	XMLName xml.Name `xml:"tables"`
	Tables  []string `xml:"table"`
}

type DatabaseDriver interface {
	Schema(ctx context.Context, tableName string) ([]ColumnDef, error)
	ListTables(ctx context.Context) ([]string, error)
	CreateTable(ctx context.Context, tableName string, columns []ColumnDef) error
	DropTable(ctx context.Context, tableName string) error

	List(ctx context.Context, tableName string, queryLimit int, queryOffset int) ([]map[string]any, error)
	GetByID(ctx context.Context, tableName string, recordID string) (map[string]any, error)
	Insert(ctx context.Context, tableName string, recordData map[string]any) (map[string]any, error)
	Update(ctx context.Context, tableName string, recordID string, recordData map[string]any) (map[string]any, error)
	Delete(ctx context.Context, tableName string, recordID string) error
	HealthCheck(ctx context.Context) error
	Close() error
}
