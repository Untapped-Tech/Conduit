package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	authPkg "github.com/untappedtech/conduit/internal/auth"
	"github.com/untappedtech/conduit/internal/db/impl"
	"github.com/untappedtech/conduit/internal/domain"
	httpPkg "github.com/untappedtech/conduit/internal/http"
	"github.com/untappedtech/conduit/internal/service"
)

func setupTestHTTPServerWithLimit(defaultLimit int) http.Handler {
	serverConfig := &domain.ServerConfig{}
	serverConfig.Server.DefaultLimit = defaultLimit
	serverConfig.Policy.PublicReads = true
	serverConfig.Policy.PublicWrites = true
	serverConfig.Policy.PublicMutation = true

	authChain, _ := authPkg.BuildAuthChain(serverConfig)
	tokenExtractor := authPkg.NewDefaultTokenExtractor()
	responseEncoder := httpPkg.NewResponseEncoder()

	memoryDB := impl.NewMemoryDB()
	apiService := service.NewAPIService(memoryDB, serverConfig)
	apiHandler := httpPkg.NewAPIHandler(apiService, serverConfig, responseEncoder)

	serveMux := http.NewServeMux()
	apiHandler.RegisterRoutes(serveMux)

	return authPkg.AuthMiddleware(authChain, tokenExtractor, responseEncoder)(serveMux)
}

func setupTestHTTPServer() http.Handler {
	return setupTestHTTPServerWithLimit(50)
}

func createSportsSchema(t *testing.T, srv http.Handler) {
	req := httptest.NewRequest(http.MethodPost, "/v1/schema/sports", strings.NewReader(`{
        "columns":[
            {"name":"id","type":"INTEGER","pk":true,"autoincrement":true},
            {"name":"name","type":"TEXT"},
            {"name":"players","type":"INTEGER"},
            {"name":"created_at","type":"TEXT"}
        ]
    }`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated && rec.Code != http.StatusOK {
		t.Fatalf("schema create failed: %d, body=%s", rec.Code, rec.Body.String())
	}
}

func insertSportsRecords(t *testing.T, srv http.Handler) {
	for _, body := range []string{
		`{"name":"Golf","players":4,"created_at":"2026-08-11 17:57:30"}`,
		`{"name":"Soccer","players":11,"created_at":"2026-08-11 17:57:30"}`,
	} {
		req := httptest.NewRequest(http.MethodPost, "/v1/sports", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("insert failed: %d, body=%s", rec.Code, rec.Body.String())
		}
	}
}
