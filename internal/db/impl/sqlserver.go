package impl

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/untappedtech/conduit/internal/domain"
)

var validSQLServerIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type SQLServerEngine struct {
	sqlDatabase *sql.DB
}

func NewSQLServerEngine(dataSourceName string) (domain.DatabaseDriver, error) {
	dbConn, err := sql.Open("sqlserver", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := dbConn.Ping(); err != nil {
		return nil, err
	}
	return &SQLServerEngine{sqlDatabase: dbConn}, nil
}

func quoteSQLServerIdent(identifier string) string {
	return "[" + strings.ReplaceAll(identifier, "]", "]]") + "]"
}

func (engine *SQLServerEngine) Schema(ctx context.Context, tableName string) ([]domain.ColumnDef, error) {
	if !validSQLServerIdent.MatchString(tableName) {
		return nil, domain.ErrInvalidID
	}

	query := `SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_DEFAULT, COLUMNPROPERTY(OBJECT_ID(TABLE_NAME), COLUMN_NAME, 'IsIdentity') 
		FROM INFORMATION_SCHEMA.COLUMNS 
		WHERE TABLE_NAME = @p1 ORDER BY ORDINAL_POSITION;`

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
		var colDefault sql.NullString
		var isIdentity sql.NullInt64
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

		isAuto := isIdentity.Valid && isIdentity.Int64 == 1
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

func (engine *SQLServerEngine) getPKMap(ctx context.Context, tableName string) (map[string]bool, error) {
	query := `SELECT c.COLUMN_NAME 
		FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc 
		JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE c 
		  ON c.CONSTRAINT_NAME = tc.CONSTRAINT_NAME 
		WHERE tc.TABLE_NAME = @p1 AND tc.CONSTRAINT_TYPE = 'PRIMARY KEY';`

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

func (engine *SQLServerEngine) detectPK(ctx context.Context, tableName string) (string, error) {
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

func (engine *SQLServerEngine) ListTables(ctx context.Context) ([]string, error) {
	query := `SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME;`
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

func (engine *SQLServerEngine) CreateTable(ctx context.Context, tableName string, columns []domain.ColumnDef) error {
	if !validSQLServerIdent.MatchString(tableName) {
		return domain.ErrInvalidID
	}
	if len(columns) == 0 {
		return fmt.Errorf("at least one column required")
	}

	var columnDeclarations []string
	for _, col := range columns {
		if !validSQLServerIdent.MatchString(col.Name) {
			return fmt.Errorf("invalid column name: %s", col.Name)
		}
		colSQL := fmt.Sprintf("%s %s", quoteSQLServerIdent(col.Name), strings.ToUpper(col.Type))

		if col.PK != nil && *col.PK {
			if col.Autoincrement != nil && *col.Autoincrement {
				colSQL = fmt.Sprintf("%s INT IDENTITY(1,1) PRIMARY KEY", quoteSQLServerIdent(col.Name))
			} else {
				colSQL += " PRIMARY KEY"
			}
		}
		if col.Nullable != nil && !*col.Nullable {
			colSQL += " NOT NULL"
		}
		columnDeclarations = append(columnDeclarations, colSQL)
	}

	query := fmt.Sprintf(`
		IF NOT EXISTS (
			SELECT * FROM sys.objects 
			WHERE object_id = OBJECT_ID(N'%s') AND type = N'U'
		)
		BEGIN
			CREATE TABLE %s (%s);
		END;
		`,
		quoteSQLServerIdent(tableName),         // OBJECT_ID lookup
		quoteSQLServerIdent(tableName),         // CREATE TABLE name
		strings.Join(columnDeclarations, ", "), // column definitions
	)

	_, err := engine.sqlDatabase.ExecContext(ctx, query)
	return err
}

func (engine *SQLServerEngine) DropTable(ctx context.Context, tableName string) error {
	if !validSQLServerIdent.MatchString(tableName) {
		return domain.ErrInvalidID
	}
	query := fmt.Sprintf(`IF EXISTS (SELECT * FROM sys.objects WHERE object_id = OBJECT_ID(N'%s') AND type in (N'U'))
	BEGIN
		DROP TABLE %s;
	END;`, quoteSQLServerIdent(tableName), quoteSQLServerIdent(tableName))
	_, err := engine.sqlDatabase.ExecContext(ctx, query)
	return err
}

func (engine *SQLServerEngine) List(ctx context.Context, tableName string, queryLimit int, queryOffset int) ([]map[string]any, error) {
	if !validSQLServerIdent.MatchString(tableName) {
		return nil, domain.ErrInvalidID
	}

	var (
		query string
		rows  *sql.Rows
		err   error
	)

	// Unlimited mode: limit < 0 → omit FETCH clause
	if queryLimit < 0 {
		query = fmt.Sprintf(
			`SELECT * FROM %s ORDER BY (SELECT NULL) OFFSET @p1 ROWS`,
			quoteSQLServerIdent(tableName),
		)
		rows, err = engine.sqlDatabase.QueryContext(ctx, query, queryOffset)
	} else {
		// Normal bounded mode
		query = fmt.Sprintf(
			`SELECT * FROM %s ORDER BY (SELECT NULL) OFFSET @p1 ROWS FETCH NEXT @p2 ROWS ONLY`,
			quoteSQLServerIdent(tableName),
		)
		rows, err = engine.sqlDatabase.QueryContext(ctx, query, queryOffset, queryLimit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRowsToMapSlice(rows)
}

func (engine *SQLServerEngine) GetByID(ctx context.Context, tableName string, recordID string) (map[string]any, error) {
	pkCol, err := engine.detectPK(ctx, tableName)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`SELECT TOP 1 * FROM %s WHERE %s = @p1`, quoteSQLServerIdent(tableName), quoteSQLServerIdent(pkCol))
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

func (engine *SQLServerEngine) Insert(ctx context.Context, tableName string, recordData map[string]any) (map[string]any, error) {
	columnNames := make([]string, 0, len(recordData))
	placeholderMarks := make([]string, 0, len(recordData))
	valuesList := make([]any, 0, len(recordData))

	paramIndex := 1
	for key, val := range recordData {
		columnNames = append(columnNames, quoteSQLServerIdent(key))
		placeholderMarks = append(placeholderMarks, fmt.Sprintf("@p%d", paramIndex))
		valuesList = append(valuesList, val)
		paramIndex++
	}

	query := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s);`, quoteSQLServerIdent(tableName), strings.Join(columnNames, ", "), strings.Join(placeholderMarks, ", "))
	_, err := engine.sqlDatabase.ExecContext(ctx, query, valuesList...)
	if err != nil {
		return nil, err
	}

	return recordData, nil
}

func (engine *SQLServerEngine) Update(ctx context.Context, tableName string, recordID string, recordData map[string]any) (map[string]any, error) {
	pkCol, err := engine.detectPK(ctx, tableName)
	if err != nil {
		return nil, err
	}

	setAssignments := make([]string, 0, len(recordData))
	valuesList := make([]any, 0, len(recordData)+1)

	paramIndex := 1
	for key, val := range recordData {
		setAssignments = append(setAssignments, fmt.Sprintf(`%s = @p%d`, quoteSQLServerIdent(key), paramIndex))
		valuesList = append(valuesList, val)
		paramIndex++
	}
	valuesList = append(valuesList, recordID)

	query := fmt.Sprintf(`UPDATE %s SET %s WHERE %s = @p%d;`, quoteSQLServerIdent(tableName), strings.Join(setAssignments, ", "), quoteSQLServerIdent(pkCol), paramIndex)
	_, err = engine.sqlDatabase.ExecContext(ctx, query, valuesList...)
	if err != nil {
		return nil, err
	}

	return engine.GetByID(ctx, tableName, recordID)
}

func (engine *SQLServerEngine) Delete(ctx context.Context, tableName string, recordID string) error {
	pkCol, err := engine.detectPK(ctx, tableName)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE %s = @p1;`, quoteSQLServerIdent(tableName), quoteSQLServerIdent(pkCol))
	_, err = engine.sqlDatabase.ExecContext(ctx, query, recordID)
	return err
}

func (engine *SQLServerEngine) HealthCheck(ctx context.Context) error {
	return engine.sqlDatabase.PingContext(ctx)
}

func (engine *SQLServerEngine) Close() error {
	return engine.sqlDatabase.Close()
}
