package impl

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"

	"github.com/untappedtech/conduit/internal/domain"
	_ "modernc.org/sqlite"
)

var validSQLiteIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type SQLiteEngine struct {
	sqlDatabase *sql.DB
}

func NewSQLiteEngine(dataSourceName string) (domain.DatabaseDriver, error) {
	dbConn, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		return nil, err
	}
	if err := dbConn.Ping(); err != nil {
		return nil, err
	}
	return &SQLiteEngine{sqlDatabase: dbConn}, nil
}

func quoteSQLiteIdent(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func (engine *SQLiteEngine) Schema(ctx context.Context, tableName string) ([]domain.ColumnDef, error) {
	if !validSQLiteIdent.MatchString(tableName) {
		return nil, domain.ErrInvalidID
	}

	schemaQuery := fmt.Sprintf(`PRAGMA table_info(%s)`, quoteSQLiteIdent(tableName))
	rows, err := engine.sqlDatabase.QueryContext(ctx, schemaQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ddlSQL sql.NullString
	_ = engine.sqlDatabase.QueryRowContext(ctx, "SELECT sql FROM sqlite_master WHERE type='table' AND name=?", tableName).Scan(&ddlSQL)
	hasAutoIncrement := ddlSQL.Valid && strings.Contains(strings.ToUpper(ddlSQL.String), "AUTOINCREMENT")

	var columnDefinitions []domain.ColumnDef
	for rows.Next() {
		var cid int
		var colName, colType string
		var notNull, isPrimaryKey int
		var defaultValue sql.NullString

		if err := rows.Scan(&cid, &colName, &colType, &notNull, &defaultValue, &isPrimaryKey); err != nil {
			return nil, err
		}

		isNullable := notNull == 0
		pkFlag := isPrimaryKey > 0
		cidValue := cid

		columnDef := domain.ColumnDef{
			Name:     colName,
			Type:     colType,
			Nullable: &isNullable,
			PK:       &pkFlag,
			CID:      &cidValue,
		}

		if defaultValue.Valid {
			strVal := defaultValue.String
			columnDef.Default = &strVal
		}

		if pkFlag && hasAutoIncrement {
			isAuto := true
			columnDef.Autoincrement = &isAuto
		}

		columnDefinitions = append(columnDefinitions, columnDef)
	}

	if len(columnDefinitions) == 0 {
		return nil, domain.ErrNotFound
	}

	return columnDefinitions, nil
}

func (engine *SQLiteEngine) detectPK(ctx context.Context, tableName string) (string, error) {
	cols, err := engine.Schema(ctx, tableName)
	if err != nil {
		return "", err
	}
	pkCount := 0
	pkName := ""
	for _, col := range cols {
		if col.PK != nil && *col.PK {
			pkCount++
			pkName = col.Name
		}
	}
	if pkCount != 1 {
		return "", domain.ErrPrimaryKeyMissing
	}
	return pkName, nil
}

func (engine *SQLiteEngine) ListTables(ctx context.Context) ([]string, error) {
	tablesQuery := `SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' AND name NOT LIKE 'auth_%' ORDER BY name;`
	rows, err := engine.sqlDatabase.QueryContext(ctx, tablesQuery)
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

func (engine *SQLiteEngine) CreateTable(ctx context.Context, tableName string, columns []domain.ColumnDef) error {
	if !validSQLiteIdent.MatchString(tableName) {
		return domain.ErrInvalidID
	}
	if len(columns) == 0 {
		return fmt.Errorf("at least one column required")
	}

	dbTransaction, err := engine.sqlDatabase.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer dbTransaction.Rollback()

	var columnDeclarations []string
	for _, col := range columns {
		if !validSQLiteIdent.MatchString(col.Name) {
			return fmt.Errorf("invalid column name: %s", col.Name)
		}
		colSQL := fmt.Sprintf("%s %s", quoteSQLiteIdent(col.Name), strings.ToUpper(col.Type))

		isPK := col.PK != nil && *col.PK
		isAuto := col.Autoincrement != nil && *col.Autoincrement

		if isAuto {
			colSQL = fmt.Sprintf("%s INTEGER PRIMARY KEY AUTOINCREMENT", quoteSQLiteIdent(col.Name))
		} else if isPK {
			colSQL += " PRIMARY KEY"
		}

		if col.Nullable != nil && !*col.Nullable && !isAuto {
			colSQL += " NOT NULL"
		}
		if col.Default != nil && *col.Default != "" {
			colSQL += fmt.Sprintf(" DEFAULT %s", *col.Default)
		}
		columnDeclarations = append(columnDeclarations, colSQL)
	}

	createQuery := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (%s);`, quoteSQLiteIdent(tableName), strings.Join(columnDeclarations, ", "))
	if _, err := dbTransaction.ExecContext(ctx, createQuery); err != nil {
		return err
	}

	return dbTransaction.Commit()
}

func (engine *SQLiteEngine) DropTable(ctx context.Context, tableName string) error {
	if !validSQLiteIdent.MatchString(tableName) {
		return domain.ErrInvalidID
	}
	dropQuery := fmt.Sprintf(`DROP TABLE IF EXISTS %s;`, quoteSQLiteIdent(tableName))
	_, err := engine.sqlDatabase.ExecContext(ctx, dropQuery)
	return err
}

func (engine *SQLiteEngine) List(ctx context.Context, tableName string, queryLimit int, queryOffset int) ([]map[string]any, error) {
	if !validSQLiteIdent.MatchString(tableName) {
		return nil, domain.ErrInvalidID
	}

	var (
		selectQuery string
		rows        *sql.Rows
		err         error
	)

	// Unlimited mode: limit < 0 → no LIMIT clause
	if queryLimit < 0 {
		selectQuery = fmt.Sprintf(`SELECT * FROM %s OFFSET ?`, quoteSQLiteIdent(tableName))
		rows, err = engine.sqlDatabase.QueryContext(ctx, selectQuery, queryOffset)
	} else {
		// Normal bounded mode
		selectQuery = fmt.Sprintf(`SELECT * FROM %s LIMIT ? OFFSET ?`, quoteSQLiteIdent(tableName))
		rows, err = engine.sqlDatabase.QueryContext(ctx, selectQuery, queryLimit, queryOffset)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRowsToMapSlice(rows)
}

func (engine *SQLiteEngine) GetByID(ctx context.Context, tableName string, recordID string) (map[string]any, error) {
	pkCol, err := engine.detectPK(ctx, tableName)
	if err != nil {
		return nil, err
	}

	selectQuery := fmt.Sprintf(`SELECT * FROM %s WHERE %s = ? LIMIT 1`, quoteSQLiteIdent(tableName), quoteSQLiteIdent(pkCol))
	rows, err := engine.sqlDatabase.QueryContext(ctx, selectQuery, recordID)
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

func (engine *SQLiteEngine) Insert(ctx context.Context, tableName string, recordData map[string]any) (map[string]any, error) {
	columnNames := make([]string, 0, len(recordData))
	placeholderMarks := make([]string, 0, len(recordData))
	valuesList := make([]any, 0, len(recordData))

	for key, val := range recordData {
		columnNames = append(columnNames, quoteSQLiteIdent(key))
		placeholderMarks = append(placeholderMarks, "?")
		valuesList = append(valuesList, val)
	}

	insertQuery := fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, quoteSQLiteIdent(tableName), strings.Join(columnNames, ", "), strings.Join(placeholderMarks, ", "))
	result, err := engine.sqlDatabase.ExecContext(ctx, insertQuery, valuesList...)
	if err != nil {
		return nil, err
	}

	lastInsertedID, err := result.LastInsertId()
	if err == nil && lastInsertedID > 0 {
		if _, pkErr := engine.detectPK(ctx, tableName); pkErr == nil {
			return engine.GetByID(ctx, tableName, fmt.Sprintf("%d", lastInsertedID))
		}
	}
	return recordData, nil
}

func (engine *SQLiteEngine) Update(ctx context.Context, tableName string, recordID string, recordData map[string]any) (map[string]any, error) {
	pkCol, err := engine.detectPK(ctx, tableName)
	if err != nil {
		return nil, err
	}

	setAssignments := make([]string, 0, len(recordData))
	valuesList := make([]any, 0, len(recordData)+1)

	for key, val := range recordData {
		setAssignments = append(setAssignments, fmt.Sprintf(`%s = ?`, quoteSQLiteIdent(key)))
		valuesList = append(valuesList, val)
	}
	valuesList = append(valuesList, recordID)

	updateQuery := fmt.Sprintf(`UPDATE %s SET %s WHERE %s = ?`, quoteSQLiteIdent(tableName), strings.Join(setAssignments, ", "), quoteSQLiteIdent(pkCol))
	_, err = engine.sqlDatabase.ExecContext(ctx, updateQuery, valuesList...)
	if err != nil {
		return nil, err
	}

	return engine.GetByID(ctx, tableName, recordID)
}

func (engine *SQLiteEngine) Delete(ctx context.Context, tableName string, recordID string) error {
	pkCol, err := engine.detectPK(ctx, tableName)
	if err != nil {
		return err
	}

	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE %s = ?`, quoteSQLiteIdent(tableName), quoteSQLiteIdent(pkCol))
	_, err = engine.sqlDatabase.ExecContext(ctx, deleteQuery, recordID)
	return err
}

func (engine *SQLiteEngine) HealthCheck(ctx context.Context) error {
	return engine.sqlDatabase.PingContext(ctx)
}

func (engine *SQLiteEngine) Close() error {
	return engine.sqlDatabase.Close()
}

func scanRowsToMapSlice(sqlRows *sql.Rows) ([]map[string]any, error) {
	columnNames, err := sqlRows.Columns()
	if err != nil {
		return nil, err
	}

	var resultMapSlice []map[string]any
	for sqlRows.Next() {
		scannedValues := make([]any, len(columnNames))
		pointerPointers := make([]any, len(columnNames))
		for i := range scannedValues {
			pointerPointers[i] = &scannedValues[i]
		}

		if err := sqlRows.Scan(pointerPointers...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]any)
		for index, colName := range columnNames {
			var valueObject any
			if bytesVal, isBytes := scannedValues[index].([]byte); isBytes {
				valueObject = string(bytesVal)
			} else {
				valueObject = scannedValues[index]
			}
			rowMap[colName] = valueObject
		}
		resultMapSlice = append(resultMapSlice, rowMap)
	}
	return resultMapSlice, nil
}
