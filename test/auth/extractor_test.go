package auth_test

import (
	"net/http"
	"testing"

	authPkg "github.com/untappedtech/conduit/internal/auth"
)

func TestExtractor_Bearer(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("Authorization", "Bearer abc123")

	ex := authPkg.NewDefaultTokenExtractor()
	if ex.ExtractToken(req) != "abc123" {
		t.Fatalf("expected abc123")
	}
}

func TestExtractor_APIKeyHeader(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "xyz")

	ex := authPkg.NewDefaultTokenExtractor()
	if ex.ExtractToken(req) != "xyz" {
		t.Fatalf("expected xyz")
	}
}

func TestExtractor_QueryParam(t *testing.T) {
	req, _ := http.NewRequest("GET", "/?api_key=qqq", nil)

	ex := authPkg.NewDefaultTokenExtractor()
	if ex.ExtractToken(req) != "qqq" {
		t.Fatalf("expected qqq")
	}
}

func TestExtractor_Empty(t *testing.T) {
	req, _ := http.NewRequest("GET", "/", nil)

	ex := authPkg.NewDefaultTokenExtractor()
	if ex.ExtractToken(req) != "" {
		t.Fatalf("expected empty")
	}
}
