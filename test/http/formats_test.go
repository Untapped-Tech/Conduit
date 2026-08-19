package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTP_ListFormats_All(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	type testCase struct {
		format       string
		expectSnip   []string
		expectStatus int
	}
	cases := []testCase{
		{"json", []string{`"sports": [`, `"name": "Golf"`}, http.StatusOK},
		{"yaml", []string{"sports:", "name: Golf"}, http.StatusOK},
		{"toml", []string{"[[sports]]", `name = "Golf"`}, http.StatusOK},
		{"xml", []string{"<sports>", "<row>", "<name>Golf</name>"}, http.StatusOK},
		{"ndjson", []string{`"name":"Golf"`}, http.StatusOK},
		{"csv", []string{"id,name,players,created_at", "Golf"}, http.StatusOK},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/v1/sports?format="+tc.format, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != tc.expectStatus {
			t.Fatalf("[%s] expected %d, got %d", tc.format, tc.expectStatus, rec.Code)
		}
		body := rec.Body.String()
		for _, snip := range tc.expectSnip {
			if !strings.Contains(body, snip) {
				t.Fatalf("[%s] expected body to contain %q, got:\n%s", tc.format, snip, body)
			}
		}
	}
}

func TestHTTP_SingleRecordFormats_All(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	type testCase struct {
		format       string
		expectSnip   []string
		expectStatus int
	}
	cases := []testCase{
		{"json", []string{`"sports": {`, `"name": "Golf"`}, http.StatusOK},
		{"yaml", []string{"sports:", "name: Golf"}, http.StatusOK},
		{"toml", []string{"[sports]", `name = "Golf"`}, http.StatusOK},
		{"xml", []string{"<sports>", "<name>Golf</name>"}, http.StatusOK},
		{"ndjson", []string{`"name":"Golf"`}, http.StatusOK},
		{"csv", []string{"id,name,players,created_at", "Golf"}, http.StatusOK},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/v1/sports/1?format="+tc.format, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != tc.expectStatus {
			t.Fatalf("[%s] expected %d, got %d", tc.format, tc.expectStatus, rec.Code)
		}
		body := rec.Body.String()
		for _, snip := range tc.expectSnip {
			if !strings.Contains(body, snip) {
				t.Fatalf("[%s] expected body to contain %q, got:\n%s", tc.format, snip, body)
			}
		}
	}
}

func TestHTTP_ErrorFormats_All(t *testing.T) {
	srv := setupTestHTTPServer()

	type testCase struct {
		format     string
		path       string
		expectSnip []string
	}
	cases := []testCase{
		{"json", "/v1/sports/999?format=json", []string{`"error": "not found"`, `"code": 404`}},
		{"yaml", "/v1/sports/999?format=yaml", []string{"error: not found", "code: 404"}},
		{"toml", "/v1/sports/999?format=toml", []string{`error = "not found"`, "code = 404"}},
		{"xml", "/v1/sports/999?format=xml", []string{"<response>", "<code>404</code>"}},
		{"ndjson", "/v1/sports/999?format=ndjson", []string{`"error":"not found"`, `"code":404`}},
		{"csv", "/v1/sports/999?format=csv", []string{"error,code", "not found,404"}},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("[%s] expected 404, got %d", tc.format, rec.Code)
		}
		body := rec.Body.String()
		for _, snip := range tc.expectSnip {
			if !strings.Contains(body, snip) {
				t.Fatalf("[%s] expected body to contain %q, got:\n%s", tc.format, snip, body)
			}
		}
	}
}

func TestHTTP_SchemaFormats_All(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)

	type testCase struct {
		format     string
		expectSnip []string
	}
	cases := []testCase{
		{"json", []string{`"columns": [`, `"name": "id"`}},
		{"yaml", []string{"columns:", "name: id"}},
		{"toml", []string{"[[columns]]", `name = "id"`}},
		{"xml", []string{"<columns>", "<column>", "<name>id</name>"}},
		{"ndjson", []string{`"name":"id"`}},
		{"csv", []string{"name,type,cid", "id,INTEGER"}},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/v1/schema/sports?format="+tc.format, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("[%s] expected 200, got %d", tc.format, rec.Code)
		}
		body := rec.Body.String()
		for _, snip := range tc.expectSnip {
			if !strings.Contains(body, snip) {
				t.Fatalf("[%s] expected body to contain %q, got:\n%s", tc.format, snip, body)
			}
		}
	}
}

func TestHTTP_XMLNaming(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	reqList := httptest.NewRequest(http.MethodGet, "/v1/sports?format=xml", nil)
	recList := httptest.NewRecorder()
	srv.ServeHTTP(recList, reqList)
	bodyList := recList.Body.String()
	if !strings.Contains(bodyList, "<sports>") || !strings.Contains(bodyList, "<row>") {
		t.Fatalf("expected XML list with <sports> and <row>, got:\n%s", bodyList)
	}

	reqSchema := httptest.NewRequest(http.MethodGet, "/v1/schema/sports?format=xml", nil)
	recSchema := httptest.NewRecorder()
	srv.ServeHTTP(recSchema, reqSchema)
	bodySchema := recSchema.Body.String()
	if !strings.Contains(bodySchema, "<columns>") || !strings.Contains(bodySchema, "<column>") {
		t.Fatalf("expected XML schema with <columns> and <column>, got:\n%s", bodySchema)
	}
}

func TestHTTP_TOMLSingleVsMulti(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	reqSingle := httptest.NewRequest(http.MethodGet, "/v1/sports/1?format=toml", nil)
	recSingle := httptest.NewRecorder()
	srv.ServeHTTP(recSingle, reqSingle)
	bodySingle := recSingle.Body.String()
	if !strings.Contains(bodySingle, "[sports]") {
		t.Fatalf("expected single-record TOML to contain [sports], got:\n%s", bodySingle)
	}

	reqMulti := httptest.NewRequest(http.MethodGet, "/v1/sports?format=toml", nil)
	recMulti := httptest.NewRecorder()
	srv.ServeHTTP(recMulti, reqMulti)
	bodyMulti := recMulti.Body.String()
	if !strings.Contains(bodyMulti, "[[sports]]") {
		t.Fatalf("expected multi-record TOML to contain [[sports]], got:\n%s", bodyMulti)
	}
}

func TestHTTP_YAMLMappingVsSequence(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	reqSingle := httptest.NewRequest(http.MethodGet, "/v1/sports/1?format=yaml", nil)
	recSingle := httptest.NewRecorder()
	srv.ServeHTTP(recSingle, reqSingle)
	bodySingle := recSingle.Body.String()
	if !strings.Contains(bodySingle, "sports:") || strings.Contains(bodySingle, "- ") {
		t.Fatalf("expected single-record YAML mapping, got:\n%s", bodySingle)
	}

	reqMulti := httptest.NewRequest(http.MethodGet, "/v1/sports?format=yaml", nil)
	recMulti := httptest.NewRecorder()
	srv.ServeHTTP(recMulti, reqMulti)
	bodyMulti := recMulti.Body.String()
	if !strings.Contains(bodyMulti, "sports:") || !strings.Contains(bodyMulti, "- ") {
		t.Fatalf("expected multi-record YAML sequence, got:\n%s", bodyMulti)
	}
}

func TestHTTP_CSVHeaderOrdering(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	req := httptest.NewRequest(http.MethodGet, "/v1/sports?format=csv", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	body := rec.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected CSV with header and rows, got:\n%s", body)
	}
	header := lines[0]
	if !strings.HasPrefix(header, "id,name,players,created_at") {
		t.Fatalf("expected CSV header ordered by schema CID, got: %s", header)
	}
}

func TestHTTP_SpecialCharacters_AllFormats(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)

	insertReq := httptest.NewRequest(http.MethodPost, "/v1/sports", strings.NewReader(`{"name":"<Golf & \"Fun\" 🏌️>","players":4}`))
	insertReq.Header.Set("Content-Type", "application/json")
	insertRec := httptest.NewRecorder()
	srv.ServeHTTP(insertRec, insertReq)
	if insertRec.Code != http.StatusCreated {
		t.Fatalf("insert failed: %d", insertRec.Code)
	}

	formats := []string{"json", "yaml", "toml", "xml", "ndjson", "csv"}
	for _, fmtStr := range formats {
		req := httptest.NewRequest(http.MethodGet, "/v1/sports/1?format="+fmtStr, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		body := rec.Body.String()
		if !strings.Contains(body, "Golf") {
			t.Fatalf("[%s] expected special-name to contain Golf, got:\n%s", fmtStr, body)
		}
	}
}

func TestHTTP_ContentTypeHeaders(t *testing.T) {
	srv := setupTestHTTPServer()
	createSportsSchema(t, srv)
	insertSportsRecords(t, srv)

	type testCase struct {
		format string
		expect string
	}
	cases := []testCase{
		{"json", "application/json"},
		{"yaml", "yaml"},
		{"toml", "toml"},
		{"xml", "xml"},
		{"ndjson", "ndjson"},
		{"csv", "text/csv"},
	}

	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/v1/sports?format="+tc.format, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		ct := rec.Header().Get("Content-Type")
		if !strings.Contains(ct, tc.expect) {
			t.Fatalf("[%s] expected Content-Type to contain %q, got %q", tc.format, tc.expect, ct)
		}
	}
}
