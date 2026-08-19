package db

import (
	"errors"

	"github.com/untappedtech/conduit/internal/db/impl"
	"github.com/untappedtech/conduit/internal/domain"
)

func NewDatabase(serverConfig *domain.ServerConfig) (domain.DatabaseDriver, error) {
	var rawEngine domain.DatabaseDriver
	var err error

	switch serverConfig.Database.Driver {
	case "sqlite":
		rawEngine, err = impl.NewSQLiteEngine(serverConfig.Database.DSN)
	case "memory":
		rawEngine = impl.NewMemoryDB()
	case "postgres":
		rawEngine, err = impl.NewPostgresEngine(serverConfig.Database.DSN)
	case "mysql":
		rawEngine, err = impl.NewMySQLEngine(serverConfig.Database.DSN)
	case "sqlserver":
		rawEngine, err = impl.NewSQLServerEngine(serverConfig.Database.DSN)
	default:
		return nil, errors.New("unsupported database driver: " + serverConfig.Database.Driver)
	}

	if err != nil {
		return nil, err
	}

	return NewCachedDatabase(rawEngine, 256), nil
}
