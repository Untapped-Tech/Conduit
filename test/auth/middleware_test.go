package auth_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authPkg "github.com/untappedtech/conduit/internal/auth"
	"github.com/untappedtech/conduit/internal/auth/impl"
	"github.com/untappedtech/conduit/internal/domain"
)

type dummyResponderMW struct{}

func (d dummyResponderMW) EncodeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	http.Error(w, msg, status)
}

func TestMiddleware_Unauthorized(t *testing.T) {
	policy := domain.PolicyConfig{
		PublicReads:    false,
		PublicWrites:   false,
		PublicMutation: false,
	}
	authChain := []domain.AuthProvider{impl.NewGlobalPolicyAuth(&policy)}

	mw := authPkg.AuthMiddleware(authChain, authPkg.NewDefaultTokenExtractor(), dummyResponderMW{})

	req := httptest.NewRequest("GET", "/v1/table", nil)
	rec := httptest.NewRecorder()

	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not reach handler")
	})).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401")
	}
}

func TestMiddleware_Authorized(t *testing.T) {
	policy := domain.PolicyConfig{
		PublicReads:    true,
		PublicWrites:   true,
		PublicMutation: true,
	}
	authChain := []domain.AuthProvider{impl.NewGlobalPolicyAuth(&policy)}

	mw := authPkg.AuthMiddleware(authChain, authPkg.NewDefaultTokenExtractor(), dummyResponderMW{})

	req := httptest.NewRequest("GET", "/v1/table", nil)
	rec := httptest.NewRecorder()

	called := false
	mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})).ServeHTTP(rec, req)

	if !called {
		t.Fatalf("expected handler to be called")
	}
}

func TestMiddleware_ActionClassification(t *testing.T) {
	policy := domain.PolicyConfig{
		PublicReads:    true,
		PublicWrites:   true,
		PublicMutation: true,
	}
	authChain := []domain.AuthProvider{impl.NewGlobalPolicyAuth(&policy)}

	mw := authPkg.AuthMiddleware(authChain, authPkg.NewDefaultTokenExtractor(), dummyResponderMW{})

	tests := []struct {
		method string
		path   string
		action domain.ActionType
	}{
		{"GET", "/v1/table", domain.ActionReadTable},
		{"POST", "/v1/table", domain.ActionWriteTable},
		{"PUT", "/v1/table/1", domain.ActionWriteTable},
		{"DELETE", "/v1/table/1", domain.ActionWriteTable},
		{"POST", "/v1/schema/table", domain.ActionMutateSchema},
		{"DELETE", "/v1/schema/table", domain.ActionMutateSchema},
	}

	for _, tc := range tests {
		t.Run(tc.method+" "+tc.path, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()

			mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				role := domain.RoleFromContext(r.Context())
				if role != domain.RoleAdmin {
					t.Fatalf("expected RoleAdmin injected, got %v (method=%s path=%s)", role, tc.method, tc.path)
				}
			})).ServeHTTP(rec, req)

			if rec.Code >= 400 {
				t.Fatalf("unexpected error response %d for method=%s path=%s", rec.Code, tc.method, tc.path)
			}
		})
	}
}
