package auth_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/untappedtech/conduit/internal/auth/impl"
	"github.com/untappedtech/conduit/internal/domain"
)

type dummyResponder struct{}

func (d dummyResponder) EncodeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	http.Error(w, msg, status)
}

func TestGlobalPolicy_ReadsDenied(t *testing.T) {
	policy := domain.PolicyConfig{
		PublicReads:    false,
		PublicWrites:   true,
		PublicMutation: true,
	}
	auth := impl.NewGlobalPolicyAuth(&policy)

	allowed, handled, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Action: domain.ActionReadTable,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled || allowed {
		t.Fatalf("expected read denied, got allowed=%v handled=%v", allowed, handled)
	}
}

func TestGlobalPolicy_WritesDenied(t *testing.T) {
	policy := domain.PolicyConfig{
		PublicReads:    true,
		PublicWrites:   false,
		PublicMutation: true,
	}
	auth := impl.NewGlobalPolicyAuth(&policy)

	allowed, handled, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Action: domain.ActionWriteTable,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled || allowed {
		t.Fatalf("expected write denied, got allowed=%v handled=%v", allowed, handled)
	}
}

func TestGlobalPolicy_MutationDenied(t *testing.T) {
	policy := domain.PolicyConfig{
		PublicReads:    true,
		PublicWrites:   true,
		PublicMutation: false,
	}
	auth := impl.NewGlobalPolicyAuth(&policy)

	allowed, handled, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Action: domain.ActionMutateSchema,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled || allowed {
		t.Fatalf("expected mutation denied, got allowed=%v handled=%v", allowed, handled)
	}
}

func TestGlobalPolicy_AllAllowed_IsNoop(t *testing.T) {
	policy := domain.PolicyConfig{
		PublicReads:    true,
		PublicWrites:   true,
		PublicMutation: true,
	}
	auth := impl.NewGlobalPolicyAuth(&policy)

	if !auth.IsNoop() {
		t.Fatalf("expected IsNoop=true")
	}

	allowed, handled, err := auth.Authorize(context.Background(), domain.AuthRequest{
		Action: domain.ActionWriteTable,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled || !allowed {
		t.Fatalf("expected allowed=true handled=true")
	}
}
