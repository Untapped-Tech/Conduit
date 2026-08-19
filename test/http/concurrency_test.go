package http_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestHTTP_ConcurrentAccess(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			body := fmt.Sprintf(`{"name":"Team%d","players":%d}`, i, i+1)
			req := httptest.NewRequest(http.MethodPost, "/v1/sports", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			srv.ServeHTTP(rec, req)
		}(i)
	}
	wg.Wait()

	req := httptest.NewRequest(http.MethodGet, "/v1/sports?format=json&limit=100", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	if !strings.Contains(body, `"sports": [`) {
		t.Fatalf("expected JSON list after concurrent inserts, got:\n%s", body)
	}
}
