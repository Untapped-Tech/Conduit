package service_test

import (
	"github.com/untappedtech/conduit/internal/db/impl"
	"github.com/untappedtech/conduit/internal/domain"
	"github.com/untappedtech/conduit/internal/service"
)

func setupTestAPIServiceWithLimit(defaultLimit int) *service.APIService {
	cfg := &domain.ServerConfig{}
	cfg.Server.DefaultLimit = defaultLimit
	memoryDB := impl.NewMemoryDB()
	return service.NewAPIService(memoryDB, cfg)
}

func setupTestAPIService() *service.APIService {
	return setupTestAPIServiceWithLimit(50)
}
