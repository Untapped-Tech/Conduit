package http_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP_NDJSONStreaming_MultiRow(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/v1/sports?format=ndjson", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)

	body := rec.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected multiple NDJSON lines, got:\n%s", body)
	}
	for _, line := range lines {
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Fatalf("invalid NDJSON line: %s, err=%v", line, err)
		}
	}
}

func TestHTTP_NDJSONErrorShape(t *testing.T) {
	srv := setupTestHTTPServer()

	req := httptest.NewRequest(http.MethodGet, "/v1/sports/999?format=ndjson", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := strings.TrimSpace(rec.Body.String())
	lines := strings.Split(body, "\n")
	if len(lines) != 1 {
		t.Fatalf("expected single NDJSON error line, got:\n%s", body)
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &obj); err != nil {
		t.Fatalf("invalid NDJSON error JSON: %s, err=%v", lines[0], err)
	}
}
