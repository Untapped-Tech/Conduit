package impl

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/lib/pq"
	"github.com/untappedtech/conduit/internal/domain"
)

var validPostgresIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type PostgresEngine struct {
	sqlDatabase *sql.DB
}

func NewPostgresEngine(dataSourceName string) (domain.DatabaseDriver, error) {
	dbConn, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := dbConn.Ping(); err != nil {
		return nil, err
	}
	return &PostgresEngine{sqlDatabase: dbConn}, nil
}

func quotePostgresIdent(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func (engine *PostgresEngine) Schema(ctx context.Context, tableName string) ([]domain.ColumnDef, error) {
	if !validPostgresIdent.MatchString(tableName) {
		return nil, domain.ErrInvalidID
	}

	query := `SELECT column_name, data_type, is_nullable, column_default, is_identity 
		FROM information_schema.columns 
		WHERE table_name = $1 ORDER BY ordinal_position;`

	rows, err := engine.sqlDatabase.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pkMap, _ := engine.getPKMap(ctx, tableName)

	var columnDefinitions []domain.ColumnDef
	cid := 0
	for rows.Next() {
		var colName, colType, isNullable string
		var colDefault, isIdentity sql.NullString
		if err := rows.Scan(&colName, &colType, &isNullable, &colDefault, &isIdentity); err != nil {
			return nil, err
		}
		nullableFlag := isNullable == "YES"
		isPK := pkMap[colName]
		cidValue := cid
		col := domain.ColumnDef{
			Name:     colName,
			Type:     colType,
			Nullable: &nullableFlag,
			PK:       &isPK,
			CID:      &cidValue,
		}
		if colDefault.Valid {
			strVal := colDefault.String
			col.Default = &strVal
		}

		isAuto := (isIdentity.Valid && isIdentity.String == "YES") || (colDefault.Valid && strings.HasPrefix(colDefault.String, "nextval"))
		if isAuto {
			col.Autoincrement = &isAuto
		}

		columnDefinitions = append(columnDefinitions, col)
		cid++
	}

	if len(columnDefinitions) == 0 {
		return nil, domain.ErrNotFound
	}
	return columnDefinitions, nil
}

func (engine *PostgresEngine) getPKMap(ctx context.Context, tableName string) (map[string]bool, error) {
	query := `SELECT kcu.column_name 
		FROM information_schema.table_constraints tc 
		JOIN information_schema.key_column_usage kcu 
		  ON tc.constraint_name = kcu.constraint_name 
		 AND tc.table_schema = kcu.table_schema
		WHERE tc.constraint_type = 'PRIMARY KEY' AND tc.table_name = $1;`

	rows, err := engine.sqlDatabase.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pks := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			pks[name] = true
		}
	}
	return pks, nil
}

func (engine *PostgresEngine) detectPK(ctx context.Context, tableName string) (string, error) {
	pkMap, err := engine.getPKMap(ctx, tableName)
	if err != nil {
		return "", err
	}
	if len(pkMap) != 1 {
		return "", domain.ErrPrimaryKeyMissing
	}
	for name := range pkMap {
		return name, nil
	}
	return "", domain.ErrPrimaryKeyMissing
}

func (engine *PostgresEngine) ListTables(ctx context.Context) ([]string, error) {
	query := `SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' ORDER BY table_name;`
	rows, err := engine.sqlDatabase.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err == nil {
			tableNames = append(tableNames, tableName)
		}
	}
	return tableNames, nil
}

func (engine *PostgresEngine) CreateTable(ctx context.Context, tableName string, columns []domain.ColumnDef) error {
	if !validPostgresIdent.MatchString(tableName) {
		return domain.ErrInvalidID
	}
	if len(columns) == 0 {
		return fmt.Errorf("at least one column required")
	}

	var columnDeclarations []string
	for _, col := range columns {
		if !validPostgresIdent.MatchString(col.Name) {
			return fmt.Errorf("invalid column name: %s", col.Name)
		}
		colSQL := fmt.Sprintf("%s %s", quotePostgresIdent(col.Name), strings.ToUpper(col.Type))

		if col.PK != nil && *col.PK {
			if col.Autoincrement != nil && *col.Autoincrement {
				colSQL = fmt.Sprintf("%s SERIAL PRIMARY KEY", quotePostgresIdent(col.Name))
			} else {
				colSQL += " PRIMARY KEY"
			}
		}
		if col.Nullable != nil && !*col.Nullable {
			colSQL += " NOT NULL"
		}
		columnDeclarations = append(columnDeclarations, colSQL)
	}

	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (%s);`, quotePostgresIdent(tableName), strings.Join(columnDeclarations, ", "))
	_, err := engine.sqlDatabase.ExecContext(ctx, query)
	return err
}

func (engine *PostgresEngine) DropTable(ctx context.Context, tableName string) error {
	if !validPostgresIdent.MatchString(tableName) {
		return domain.ErrInvalidID
	}
	_, err := engine.sqlDatabase.ExecContext(ctx, fmt.Sprintf(`DROP TABLE IF EXISTS %s CASCADE;`, quotePostgresIdent(tableName)))
	return err
}

func (engine *PostgresEngine) List(ctx context.Context, tableName string, queryLimit int, queryOffset int) ([]map[string]any, error) {
	if !validPostgresIdent.MatchString(tableName) {
		return nil, domain.ErrInvalidID
	}

	var (
		query string
		rows  *sql.Rows
		err   error
	)

	// Unlimited mode: limit < 0 → no LIMIT clause
	if queryLimit < 0 {
		query = fmt.Sprintf(`SELECT * FROM %s OFFSET $1`, quotePostgresIdent(tableName))
		rows, err = engine.sqlDatabase.QueryContext(ctx, query, queryOffset)
	} else {
		// Normal bounded mode
		query = fmt.Sprintf(`SELECT * FROM %s LIMIT $1 OFFSET $2`, quotePostgresIdent(tableName))
		rows, err = engine.sqlDatabase.QueryContext(ctx, query, queryLimit, queryOffset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRowsToMapSlice(rows)
}

func (engine *PostgresEngine) GetByID(ctx context.Context, tableName string, recordID string) (map[string]any, error) {
	pkCol, err := engine.detectPK(ctx, tableName)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT * FROM %s WHERE %s = $1 LIMIT 1`, quotePostgresIdent(tableName), quotePostgresIdent(pkCol))
	rows, err := engine.sqlDatabase.QueryContext(ctx, query, recordID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanRowsToMapSlice(rows)
	if err != nil || len(results) == 0 {
		return nil, domain.ErrNotFound
	}
	return results[0], nil
}

func (engine *PostgresEngine) Insert(ctx context.Context, tableName string, recordData map[string]any) (map[string]any, error) {
	columnNames := make([]string, 0, len(recordData))
	placeholderMarks := make([]string, 0, len(recordData))
	valuesList := make([]any, 0, len(recordData))

	paramIndex := 1
	for key, val := range recordData {
		columnNames = append(columnNames, quotePostgresIdent(key))
		placeholderMarks = append(placeholderMarks, fmt.Sprintf("$%d", paramIndex))
		valuesList = append(valuesList, val)
		paramIndex++
	}

	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s) RETURNING *;`, quotePostgresIdent(tableName), strings.Join(columnNames, ", "), strings.Join(placeholderMarks, ", "))
	rows, err := engine.sqlDatabase.QueryContext(ctx, query, valuesList...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanRowsToMapSlice(rows)
	if err != nil || len(results) == 0 {
		return recordData, nil
	}
	return results[0], nil
}

func (engine *PostgresEngine) Update(ctx context.Context, tableName string, recordID string, recordData map[string]any) (map[string]any, error) {
	pkCol, err := engine.detectPK(ctx, tableName)
	if err != nil {
		return nil, err
	}

	setAssignments := make([]string, 0, len(recordData))
	valuesList := make([]any, 0, len(recordData)+1)

	paramIndex := 1
	for key, val := range recordData {
		setAssignments = append(setAssignments, fmt.Sprintf(`%s = $%d`, quotePostgresIdent(key), paramIndex))
		valuesList = append(valuesList, val)
		paramIndex++
	}
	valuesList = append(valuesList, recordID)

	query := fmt.Sprintf(`UPDATE %s SET %s WHERE %s = $%d RETURNING *;`, quotePostgresIdent(tableName), strings.Join(setAssignments, ", "), quotePostgresIdent(pkCol), paramIndex)
	rows, err := engine.sqlDatabase.QueryContext(ctx, query, valuesList...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results, err := scanRowsToMapSlice(rows)
	if err != nil || len(results) == 0 {
		return nil, domain.ErrNotFound
	}
	return results[0], nil
}

func (engine *PostgresEngine) Delete(ctx context.Context, tableName string, recordID string) error {
	pkCol, err := engine.detectPK(ctx, tableName)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE %s = $1;`, quotePostgresIdent(tableName), quotePostgresIdent(pkCol))
	_, err = engine.sqlDatabase.ExecContext(ctx, query, recordID)
	return err
}

func (engine *PostgresEngine) HealthCheck(ctx context.Context) error {
	return engine.sqlDatabase.PingContext(ctx)
}

func (engine *PostgresEngine) Close() error {
	return engine.sqlDatabase.Close()
}
