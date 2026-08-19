package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP_InsertUpdateDeleteLifecycle_JSON(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)

	insertReq := httptest.NewRequest(http.MethodPost, "/v1/sports", strings.NewReader(`{"name":"Tennis","players":2}`))
	insertReq.Header.Set("Content-Type", "application/json")
	insertRec := httptest.NewRecorder()
	srv.ServeHTTP(insertRec, insertReq)
	if insertRec.Code != http.StatusCreated {
		t.Fatalf("insert failed: %d, body=%s", insertRec.Code, insertRec.Body.String())
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/v1/sports/1", strings.NewReader(`{"name":"Tennis","players":4}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	srv.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update failed: %d, body=%s", updateRec.Code, updateRec.Body.String())
	}
	if !strings.Contains(updateRec.Body.String(), `"players": 4`) {
		t.Fatalf("expected updated players=4, got:\n%s", updateRec.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/v1/sports/1", nil)
	deleteRec := httptest.NewRecorder()
	srv.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusNoContent {
		t.Fatalf("delete failed: %d, body=%s", deleteRec.Code, deleteRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/sports/1?format=json", nil)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after delete, got %d", getRec.Code)
	}
}

func TestHTTP_RoundTrip_JSON(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)

	insertReq := httptest.NewRequest(http.MethodPost, "/v1/sports", strings.NewReader(`{"name":"RoundTrip","players":3}`))
	insertReq.Header.Set("Content-Type", "application/json")
	insertRec := httptest.NewRecorder()
	srv.ServeHTTP(insertRec, insertReq)
	if insertRec.Code != http.StatusCreated {
		t.Fatalf("insert failed: %d", insertRec.Code)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/sports/1?format=json", nil)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)
	body := getRec.Body.String()
	if !strings.Contains(body, `"RoundTrip"`) {
		t.Fatalf("expected RoundTrip in JSON, got:\n%s", body)
	}

	updateReq := httptest.NewRequest(http.MethodPut, "/v1/sports/1", strings.NewReader(`{"name":"RoundTrip2","players":5}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	srv.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update failed: %d", updateRec.Code)
	}

	getReq2 := httptest.NewRequest(http.MethodGet, "/v1/sports/1?format=json", nil)
	getRec2 := httptest.NewRecorder()
	srv.ServeHTTP(getRec2, getReq2)
	body2 := getRec2.Body.String()
	if !strings.Contains(body2, `"RoundTrip2"`) {
		t.Fatalf("expected RoundTrip2 in JSON, got:\n%s", body2)
	}
}

func TestHTTP_MalformedPayloads(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)

	req := httptest.NewRequest(http.MethodPost, "/v1/sports", strings.NewReader(`{"name":`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed JSON, got %d", rec.Code)
	}
}
