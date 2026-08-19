package auth_test

import (
	"context"
	"os"
	"testing"

	"github.com/untappedtech/conduit/internal/auth/impl"
	"github.com/untappedtech/conduit/internal/domain"
)

func TestEnvAuth_AdminToken(t *testing.T) {
	os.Setenv("AUTH_TOKEN_ADMIN", "admintoken")
	defer os.Unsetenv("AUTH_TOKEN_ADMIN")

	auth := impl.NewEnvAuthenticator()

	allowed, handled, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "admintoken",
		Action: domain.ActionMutateSchema,
	})
	if err != nil || !handled || !allowed {
		t.Fatalf("expected admin full access")
	}
}

func TestEnvAuth_ReadWrite_NoSchemaMutation(t *testing.T) {
	os.Setenv("AUTH_TOKEN_READ_WRITE", "rwtoken")
	defer os.Unsetenv("AUTH_TOKEN_READ_WRITE")

	auth := impl.NewEnvAuthenticator()

	allowed, _, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "rwtoken",
		Action: domain.ActionMutateSchema,
	})
	if err == nil && allowed {
		t.Fatalf("expected read-write to deny schema mutation")
	}
}

func TestEnvAuth_ReadOnly_OnlyReads(t *testing.T) {
	os.Setenv("AUTH_TOKEN_READ_ONLY", "rotoken")
	defer os.Unsetenv("AUTH_TOKEN_READ_ONLY")

	auth := impl.NewEnvAuthenticator()

	allowed, _, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "rotoken",
		Action: domain.ActionWriteTable,
	})
	if err == nil && allowed {
		t.Fatalf("expected read-only to deny writes")
	}
}

func TestEnvAuth_WrongToken(t *testing.T) {
	os.Setenv("AUTH_TOKEN_ADMIN", "admintoken")
	defer os.Unsetenv("AUTH_TOKEN_ADMIN")

	auth := impl.NewEnvAuthenticator()

	allowed, handled, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Token:  "wrong",
		Action: domain.ActionReadTable,
	})
	if err == nil || allowed || !handled {
		t.Fatalf("expected unauthorized wrong token")
	}
}

func TestEnvAuth_NoTokens_IsNoop(t *testing.T) {
	os.Unsetenv("AUTH_TOKEN_ADMIN")
	os.Unsetenv("AUTH_TOKEN_READ_WRITE")
	os.Unsetenv("AUTH_TOKEN_READ_ONLY")

	auth := impl.NewEnvAuthenticator()

	if !auth.IsNoop() {
		t.Fatalf("expected IsNoop=true")
	}
}
