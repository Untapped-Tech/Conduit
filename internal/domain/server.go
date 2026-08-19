package domain

import (
	"encoding/xml"
	"errors"
)

var (
	ErrForbiddenTable = errors.New("action on table is forbidden by policy")
	ErrNotFound       = errors.New("resource not found")
	ErrInvalidFormat  = errors.New("unsupported format specified")
	ErrInvalidID      = errors.New("invalid identifier supplied")
	ErrReadOnlyColumn = errors.New("read-only or auto-generated column specified in write payload")
)

type ErrorResponse struct {
	XMLName xml.Name `json:"-" yaml:"-" xml:"response" toml:"-"`
	Error   string   `json:"error" yaml:"error" xml:"error" toml:"error"`
	Code    int      `json:"code" yaml:"code" xml:"code" toml:"code"`
}

type FormatType string

const (
	FormatJSON   FormatType = "json"
	FormatNDJSON FormatType = "ndjson"
	FormatYAML   FormatType = "yaml"
	FormatXML    FormatType = "xml"
	FormatTOML   FormatType = "toml"
	FormatCSV    FormatType = "csv"
)

func (formatFormat FormatType) ContentType() string {
	switch formatFormat {
	case FormatJSON:
		return "application/json"
	case FormatXML:
		return "application/xml"
	case FormatYAML:
		return "application/x-yaml"
	case FormatTOML:
		return "application/toml"
	case FormatNDJSON:
		return "application/x-ndjson"
	case FormatCSV:
		return "text/csv"
	}
	return "application/json"
}

type CacheConfig struct {
	Capacity   int `json:"capacity" yaml:"capacity" xml:"capacity" toml:"capacity"`
	TTLSeconds int `json:"ttlSeconds" yaml:"ttlSeconds" xml:"ttlSeconds" toml:"ttlSeconds"`
}

type DBAuthConfig struct {
	Driver      string            `json:"driver" yaml:"driver" xml:"driver" toml:"driver"`
	DSN         string            `json:"dsn" yaml:"dsn" xml:"dsn" toml:"dsn"`
	Table       string            `json:"table" yaml:"table" xml:"table" toml:"table"`
	TokenColumn string            `json:"tokenColumn" yaml:"tokenColumn" xml:"tokenColumn" toml:"tokenColumn"`
	RoleColumn  string            `json:"roleColumn" yaml:"roleColumn" xml:"roleColumn" toml:"roleColumn"`
	Roles       map[string]string `json:"roles" yaml:"roles" xml:"roles" toml:"roles"`
	Cache       CacheConfig       `json:"cache" yaml:"cache" xml:"cache" toml:"cache"`
}

type PolicyConfig struct {
	PublicReads    bool `json:"publicReads" yaml:"publicReads" xml:"publicReads" toml:"publicReads"`
	PublicWrites   bool `json:"publicWrites" yaml:"publicWrites" xml:"publicWrites" toml:"publicWrites"`
	PublicMutation bool `json:"publicMutation" yaml:"publicMutation" xml:"publicMutation" toml:"publicMutation"`
}

type AuthConfig struct {
	EnvironmentEnabled bool         `json:"environmentEnabled" yaml:"environmentEnabled" xml:"environmentEnabled" toml:"environmentEnabled"`
	DBEnabled          bool         `json:"dbEnabled" yaml:"dbEnabled" xml:"dbEnabled" toml:"dbEnabled"`
	DBAuth             DBAuthConfig `json:"dbAuth" yaml:"dbAuth" xml:"dbAuth" toml:"dbAuth"`
}

type DatabaseConfig struct {
	Driver string `json:"driver" yaml:"driver" xml:"driver" toml:"driver"`
	DSN    string `json:"dsn" yaml:"dsn" xml:"dsn" toml:"dsn"`
}

type ServerConfig struct {
	Server struct {
		Host         string `json:"host" yaml:"host" xml:"host" toml:"host"`
		Port         int    `json:"port" yaml:"port" xml:"port" toml:"port"`
		DefaultLimit int    `json:"default_limit" yaml:"default_limit" xml:"default_limit" toml:"default_limit"`
	} `json:"server" yaml:"server" xml:"server" toml:"server"`
	Database DatabaseConfig `json:"database" yaml:"database" xml:"database" toml:"database"`
	Policy   PolicyConfig   `json:"policy" yaml:"policy" xml:"policy" toml:"policy"`
	Auth     AuthConfig     `json:"auth" yaml:"auth" xml:"auth" toml:"auth"`
}
