package impl

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
	"github.com/untappedtech/conduit/internal/auth/cache"
	"github.com/untappedtech/conduit/internal/domain"
	_ "modernc.org/sqlite"
)

type DBAuth struct {
	databaseConnection *sql.DB
	tableName          string
	tokenColumn        string
	roleColumn         string
	roleMapping        map[string]domain.Role
	roleCache          *cache.TTLLRUCache
	driverName         string
}

func NewDBAuthenticator(dbConfig *domain.DBAuthConfig) (*DBAuth, error) {
	dbConn, err := sql.Open(dbConfig.Driver, dbConfig.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open auth database: %w", err)
	}
	if err := dbConn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping auth database: %w", err)
	}

	compiledRoles := make(map[string]domain.Role)
	for configKey, dbRoleValue := range dbConfig.Roles {
		normalizedKey := strings.TrimSpace(configKey)
		normalizedVal := strings.ToLower(strings.TrimSpace(dbRoleValue))
		switch normalizedKey {
		case "admin":
			compiledRoles[normalizedVal] = domain.RoleAdmin
		case "readWrite":
			compiledRoles[normalizedVal] = domain.RoleReadWrite
		case "readOnly":
			compiledRoles[normalizedVal] = domain.RoleReadOnly
		}
	}

	return &DBAuth{
		databaseConnection: dbConn,
		tableName:          dbConfig.Table,
		tokenColumn:        dbConfig.TokenColumn,
		roleColumn:         dbConfig.RoleColumn,
		roleMapping:        compiledRoles,
		roleCache:          cache.NewTTLLRUCache(dbConfig.Cache.Capacity, dbConfig.Cache.TTLSeconds),
		driverName:         dbConfig.Driver,
	}, nil
}

func (dbAuth *DBAuth) lookupRole(ctx context.Context, authToken string) (string, error) {
	if cachedRole, cacheHit := dbAuth.roleCache.Get(authToken); cacheHit {
		if cachedRole == cache.SentinelInvalidToken {
			return "", domain.ErrUnauthorized
		}
		return cachedRole, nil
	}

	roleFromDB, queryError := dbAuth.queryRoleFromDB(ctx, authToken)
	if queryError != nil {
		dbAuth.roleCache.Set(authToken, cache.SentinelInvalidToken)
		return "", queryError
	}

	dbAuth.roleCache.Set(authToken, roleFromDB)
	return roleFromDB, nil
}

func (dbAuth *DBAuth) queryRoleFromDB(ctx context.Context, authToken string) (string, error) {
	selectQuery := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", dbAuth.roleColumn, dbAuth.tableName, dbAuth.tokenColumn)
	if dbAuth.driverName == "postgres" {
		selectQuery = fmt.Sprintf("SELECT %s FROM %s WHERE %s = $1", dbAuth.roleColumn, dbAuth.tableName, dbAuth.tokenColumn)
	} else if dbAuth.driverName == "sqlserver" {
		selectQuery = fmt.Sprintf("SELECT %s FROM %s WHERE %s = @p1", dbAuth.roleColumn, dbAuth.tableName, dbAuth.tokenColumn)
	}

	var rawRoleFromDB string
	rowError := dbAuth.databaseConnection.QueryRowContext(ctx, selectQuery, authToken).Scan(&rawRoleFromDB)
	if rowError != nil {
		if rowError == sql.ErrNoRows {
			return "", domain.ErrUnauthorized
		}
		return "", rowError
	}
	return strings.ToLower(strings.TrimSpace(rawRoleFromDB)), nil
}

func (dbAuth *DBAuth) Authorize(ctx context.Context, request domain.AuthRequest) (bool, bool, error) {
	if request.Token == "" {
		return false, false, nil
	}

	rawRole, lookupError := dbAuth.lookupRole(ctx, request.Token)
	if lookupError != nil {
		return false, true, domain.ErrUnauthorized
	}

	mappedRole, exists := dbAuth.roleMapping[rawRole]
	if !exists {
		mappedRole = domain.RoleReadOnly
	}

	switch mappedRole {
	case domain.RoleAdmin:
		return true, true, nil
	case domain.RoleReadWrite:
		return request.Action != domain.ActionMutateSchema, true, nil
	case domain.RoleReadOnly:
		return request.Action == domain.ActionReadTable, true, nil
	default:
		return false, true, domain.ErrUnauthorized
	}
}

func (dbAuth *DBAuth) IsNoop() bool {
	return strings.TrimSpace(dbAuth.tableName) == "" ||
		strings.TrimSpace(dbAuth.tokenColumn) == "" ||
		strings.TrimSpace(dbAuth.roleColumn) == "" ||
		len(dbAuth.roleMapping) == 0
}

func (dbAuth *DBAuth) Close() error {
	return dbAuth.databaseConnection.Close()
}
