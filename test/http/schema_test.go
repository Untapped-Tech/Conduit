package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP_SchemaMutation(t *testing.T) {
	srv := setupTestHTTPServer()

	reqCreate := httptest.NewRequest(http.MethodPost, "/v1/schema/temp", strings.NewReader(`{
        "columns":[{"name":"id","type":"INTEGER","pk":true}]
    }`))
	reqCreate.Header.Set("Content-Type", "application/json")
	recCreate := httptest.NewRecorder()
	srv.ServeHTTP(recCreate, reqCreate)
	if recCreate.Code != http.StatusCreated && recCreate.Code != http.StatusOK {
		t.Fatalf("initial schema create failed: %d", recCreate.Code)
	}

	reqDrop := httptest.NewRequest(http.MethodDelete, "/v1/schema/temp", nil)
	recDrop := httptest.NewRecorder()
	srv.ServeHTTP(recDrop, reqDrop)
	if recDrop.Code != http.StatusNoContent {
		t.Fatalf("schema drop failed: %d", recDrop.Code)
	}

	reqRecreate := httptest.NewRequest(http.MethodPost, "/v1/schema/temp", strings.NewReader(`{
        "columns":[{"name":"id","type":"INTEGER","pk":true},{"name":"name","type":"TEXT"}]
    }`))
	reqRecreate.Header.Set("Content-Type", "application/json")
	recRecreate := httptest.NewRecorder()
	srv.ServeHTTP(recRecreate, reqRecreate)
	if recRecreate.Code != http.StatusCreated && recRecreate.Code != http.StatusOK {
		t.Fatalf("schema recreate failed: %d", recRecreate.Code)
	}

	reqSchema := httptest.NewRequest(http.MethodGet, "/v1/schema/temp?format=json", nil)
	recSchema := httptest.NewRecorder()
	srv.ServeHTTP(recSchema, reqSchema)
	body := recSchema.Body.String()
	if !strings.Contains(body, `"name": "name"`) {
		t.Fatalf("expected new schema to include name column, got:\n%s", body)
	}
}
