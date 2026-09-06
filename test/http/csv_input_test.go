package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP_InsertCSV(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)

	insertReq := httptest.NewRequest(http.MethodPost, "/v1/sports", strings.NewReader("name,players,created_at\nGolf,4,2026-08-11 17:57:30"))
	insertReq.Header.Set("Content-Type", "text/csv")
	insertRec := httptest.NewRecorder()
	srv.ServeHTTP(insertRec, insertReq)
	if insertRec.Code != http.StatusCreated {
		t.Fatalf("csv insert failed: %d, body=%s", insertRec.Code, insertRec.Body.String())
	}
	if ct := insertRec.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("expected CSV response Content-Type, got %q", ct)
	}
	if !strings.Contains(insertRec.Body.String(), "Golf") {
		t.Fatalf("expected Golf in CSV insert response, got:\n%s", insertRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/sports/1?format=json", nil)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("get after csv insert failed: %d", getRec.Code)
	}
	if !strings.Contains(getRec.Body.String(), `"Golf"`) {
		t.Fatalf("expected Golf after csv insert, got:\n%s", getRec.Body.String())
	}
}

func TestHTTP_PatchCSV(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	patchReq := httptest.NewRequest(http.MethodPatch, "/v1/sports/1", strings.NewReader("players\n9"))
	patchReq.Header.Set("Content-Type", "text/csv")
	patchRec := httptest.NewRecorder()
	srv.ServeHTTP(patchRec, patchReq)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("csv patch failed: %d, body=%s", patchRec.Code, patchRec.Body.String())
	}
	if !strings.Contains(patchRec.Body.String(), "Golf") {
		t.Fatalf("expected name to remain Golf after CSV PATCH, got:\n%s", patchRec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/sports/1?format=json", nil)
	getRec := httptest.NewRecorder()
	srv.ServeHTTP(getRec, getReq)
	body := getRec.Body.String()
	if !strings.Contains(body, `"players": 9`) {
		t.Fatalf("expected players=9 after csv patch, got:\n%s", body)
	}
	if !strings.Contains(body, `"Golf"`) {
		t.Fatalf("expected Golf to remain after csv patch, got:\n%s", body)
	}
}

func TestHTTP_CreateSchemaCSV(t *testing.T) {
	srv := setupTestHTTPServer()

	req := httptest.NewRequest(http.MethodPost, "/v1/schema/teams", strings.NewReader("name,type,pk,autoincrement\nid,integer,true,true\nname,text,false,false"))
	req.Header.Set("Content-Type", "text/csv")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("csv schema create failed: %d, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "id") || !strings.Contains(rec.Body.String(), "name") {
		t.Fatalf("expected created columns in response, got:\n%s", rec.Body.String())
	}

	insertReq := httptest.NewRequest(http.MethodPost, "/v1/teams", strings.NewReader(`{"name":"Aces"}`))
	insertReq.Header.Set("Content-Type", "application/json")
	insertRec := httptest.NewRecorder()
	srv.ServeHTTP(insertRec, insertReq)
	if insertRec.Code != http.StatusCreated {
		t.Fatalf("insert into csv-created table failed: %d, body=%s", insertRec.Code, insertRec.Body.String())
	}
}
