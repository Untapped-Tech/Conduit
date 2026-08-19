package auth_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/untappedtech/conduit/internal/auth/impl"
	"github.com/untappedtech/conduit/internal/domain"
)

// Each test gets its own isolated in-memory SQLite DB.
// Using unique DSNs prevents "table already exists" errors.
func setupSQLiteAuthDB(t *testing.T, name string) string {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("failed to open sqlite: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE auth_tokens (token TEXT PRIMARY KEY, role TEXT)`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	_, err = db.Exec(`INSERT INTO auth_tokens VALUES 
        ('admintoken','admin'),
        ('rwtoken','readwrite'),
        ('rotoken','readonly')`)
	if err != nil {
		t.Fatalf("failed to insert rows: %v", err)
	}

	return dsn
}

func newDBAuth(t *testing.T, dsn string) *impl.DBAuth {
	auth, err := impl.NewDBAuthenticator(&domain.DBAuthConfig{
		Driver:      "sqlite",
		DSN:         dsn,
		Table:       "auth_tokens",
		TokenColumn: "token",
		RoleColumn:  "role",
		Roles: map[string]string{
			"admin":     "admin",
			"readWrite": "readwrite",
			"readOnly":  "readonly",
		},
		Cache: domain.CacheConfig{Capacity: 100, TTLSeconds: 1},
	})
	if err != nil {
		t.Fatalf("failed to init db auth: %v", err)
	}
	return auth
}

func TestDBAuth_Admin(t *testing.T) {
	dsn := setupSQLiteAuthDB(t, "db_auth_admin")
	auth := newDBAuth(t, dsn)

	allowed, handled, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "admintoken",
		Action: domain.ActionMutateSchema,
	})
	if err != nil || !handled || !allowed {
		t.Fatalf("expected admin full access")
	}
}

func TestDBAuth_ReadWrite_NoSchemaMutation(t *testing.T) {
	dsn := setupSQLiteAuthDB(t, "db_auth_rw")
	auth := newDBAuth(t, dsn)

	allowed, _, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "rwtoken",
		Action: domain.ActionMutateSchema,
	})
	if err == nil && allowed {
		t.Fatalf("expected read-write to deny schema mutation")
	}
}

func TestDBAuth_ReadOnly_OnlyReads(t *testing.T) {
	dsn := setupSQLiteAuthDB(t, "db_auth_ro")
	auth := newDBAuth(t, dsn)

	allowed, _, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "rotoken",
		Action: domain.ActionWriteTable,
	})
	if err == nil && allowed {
		t.Fatalf("expected read-only to deny writes")
	}
}

func TestDBAuth_InvalidToken_Cached(t *testing.T) {
	dsn := setupSQLiteAuthDB(t, "db_auth_invalid")
	auth := newDBAuth(t, dsn)

	// First lookup → DB miss → unauthorized
	_, handled, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "badtoken",
		Action: domain.ActionReadTable,
	})
	if err == nil || !handled {
		t.Fatalf("expected unauthorized for invalid token")
	}

	// Second lookup → cache hit → no DB query
	_, handled2, err2 := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "badtoken",
		Action: domain.ActionReadTable,
	})
	if err2 == nil || !handled2 {
		t.Fatalf("expected cached unauthorized")
	}
}

func TestDBAuth_TTLExpiration(t *testing.T) {
	dsn := setupSQLiteAuthDB(t, "db_auth_ttl")
	auth := newDBAuth(t, dsn)

	// First lookup → cached
	allowed, handled, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "admintoken",
		Action: domain.ActionReadTable,
	})
	if err != nil || !handled || !allowed {
		t.Fatalf("expected admin allowed")
	}

	// Wait for TTL expiration
	time.Sleep(2 * time.Second)

	// Should query DB again
	allowed2, handled2, err2 := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "admintoken",
		Action: domain.ActionReadTable,
	})
	if err2 != nil || !handled2 || !allowed2 {
		t.Fatalf("expected admin allowed after TTL expiration")
	}
}
