package auth_test

import (
	"database/sql"
	"os"
	"testing"

	authPkg "github.com/untappedtech/conduit/internal/auth"
	"github.com/untappedtech/conduit/internal/domain"
)

func TestChainFactory_NoProviders_UsesNoop(t *testing.T) {
	cfg := &domain.ServerConfig{
		Policy: domain.PolicyConfig{
			PublicReads:    true,
			PublicWrites:   true,
			PublicMutation: true,
		},
		Auth: domain.AuthConfig{
			EnvironmentEnabled: false,
			DBEnabled:          false,
		},
	}

	chain, err := authPkg.BuildAuthChain(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain) != 1 {
		t.Fatalf("expected 1 provider (noop), got %d", len(chain))
	}

	if !chain[0].IsNoop() {
		t.Fatalf("expected noop provider")
	}
}

func TestChainFactory_GlobalPolicyOnly(t *testing.T) {
	cfg := &domain.ServerConfig{
		Policy: domain.PolicyConfig{
			PublicReads:    false,
			PublicWrites:   false,
			PublicMutation: false,
		},
		Auth: domain.AuthConfig{
			EnvironmentEnabled: false,
			DBEnabled:          false,
		},
	}

	chain, err := authPkg.BuildAuthChain(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain) != 1 {
		t.Fatalf("expected 1 provider (global policy), got %d", len(chain))
	}

	if chain[0].IsNoop() {
		t.Fatalf("global policy should not be noop")
	}
}

func TestChainFactory_EnvOnly(t *testing.T) {
	os.Setenv("AUTH_TOKEN_ADMIN", "admintoken")
	defer os.Unsetenv("AUTH_TOKEN_ADMIN")

	cfg := &domain.ServerConfig{
		Policy: domain.PolicyConfig{
			PublicReads:    false,
			PublicWrites:   false,
			PublicMutation: false,
		},
		Auth: domain.AuthConfig{
			EnvironmentEnabled: true,
			DBEnabled:          false,
		},
	}

	chain, err := authPkg.BuildAuthChain(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain) != 2 {
		t.Fatalf("expected 2 providers (global + env), got %d", len(chain))
	}
}

func TestChainFactory_DBOnly(t *testing.T) {
	cfg := &domain.ServerConfig{
		Policy: domain.PolicyConfig{
			PublicReads:    false,
			PublicWrites:   false,
			PublicMutation: false,
		},
		Auth: domain.AuthConfig{
			EnvironmentEnabled: false,
			DBEnabled:          true,
			DBAuth: domain.DBAuthConfig{
				Driver:      "sqlite",
				DSN:         "file::memory:?cache=shared",
				Table:       "auth_tokens",
				TokenColumn: "token",
				RoleColumn:  "role",
				Roles: map[string]string{
					"admin":     "admin",
					"readWrite": "readwrite",
					"readOnly":  "readonly",
				},
				Cache: domain.CacheConfig{Capacity: 10, TTLSeconds: 1},
			},
		},
	}

	// Create the table manually
	db, err := sql.Open("sqlite", cfg.Auth.DBAuth.DSN)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}
	_, _ = db.Exec(`CREATE TABLE auth_tokens (token TEXT PRIMARY KEY, role TEXT)`)

	chain, err := authPkg.BuildAuthChain(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(chain) != 2 {
		t.Fatalf("expected 2 providers (global + db), got %d", len(chain))
	}
}

func TestChainFactory_DBInitError(t *testing.T) {
	cfg := &domain.ServerConfig{
		Policy: domain.PolicyConfig{},
		Auth: domain.AuthConfig{
			DBEnabled: true,
			DBAuth: domain.DBAuthConfig{
				Driver:      "sqlite",
				DSN:         "/invalid/path/does/not/exist",
				Table:       "auth_tokens",
				TokenColumn: "token",
				RoleColumn:  "role",
				Roles:       map[string]string{"admin": "admin"},
				Cache:       domain.CacheConfig{Capacity: 10, TTLSeconds: 1},
			},
		},
	}

	_, err := authPkg.BuildAuthChain(cfg)
	if err == nil {
		t.Fatalf("expected DB init error")
	}
}
