package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP_FormatNegotiation_QueryOverridesInput(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/v1/sports?format=json", nil)
	req.Header.Set("Accept", "application/x-yaml")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected JSON Content-Type, got %s", ct)
	}
}

func TestHTTP_FormatNegotiation_InputDefault(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/v1/sports", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Fatalf("expected default JSON Content-Type, got %s", ct)
	}
}

func TestHTTP_FormatNegotiation_CSVNotDefault(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/v1/sports", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	if ct := rec.Header().Get("Content-Type"); strings.Contains(ct, "text/csv") {
		t.Fatalf("expected non-CSV default Content-Type, got %s", ct)
	}
}
