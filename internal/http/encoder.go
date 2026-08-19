package http

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/untappedtech/conduit/internal/domain"
)

type ResponseEncoder struct{}

func NewResponseEncoder() *ResponseEncoder {
	return &ResponseEncoder{}
}

func (e *ResponseEncoder) NegotiateOutputFormat(r *http.Request, input domain.FormatType) domain.FormatType {
	if formatParam := strings.ToLower(r.URL.Query().Get("format")); formatParam != "" {
		switch formatParam {
		case "json":
			return domain.FormatJSON
		case "ndjson":
			return domain.FormatNDJSON
		case "yaml", "yml":
			return domain.FormatYAML
		case "xml":
			return domain.FormatXML
		case "toml":
			return domain.FormatTOML
		case "csv":
			return domain.FormatCSV
		}
	}

	if input != "" && input != domain.FormatCSV {
		return input
	}

	return domain.FormatJSON
}

func (e *ResponseEncoder) EncodeError(w http.ResponseWriter, r *http.Request, status int, msg string) {
	negotiated := e.NegotiateOutputFormat(r, domain.FormatJSON)

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Type", negotiated.ContentType())
	w.WriteHeader(status)

	switch negotiated {
	case domain.FormatXML:
		_, _ = w.Write([]byte(xml.Header))
		_, _ = fmt.Fprintf(w, "<response>\n  <error>%s</error>\n  <code>%d</code>\n</response>\n", escapeXMLText(msg), status)
	case domain.FormatYAML:
		_, _ = fmt.Fprintf(w, "error: %s\ncode: %d\n", msg, status)
	case domain.FormatTOML:
		_, _ = fmt.Fprintf(w, "error = %q\ncode = %d\n", msg, status)
	case domain.FormatCSV:
		csvWriter := csv.NewWriter(w)
		_ = csvWriter.Write([]string{"error", "code"})
		_ = csvWriter.Write([]string{msg, fmt.Sprintf("%d", status)})
		csvWriter.Flush()
	case domain.FormatNDJSON:
		_, _ = fmt.Fprintf(w, "{\"error\":%q,\"code\":%d}\n", msg, status)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return

	case domain.FormatJSON:
		fallthrough
	default:
		_, _ = fmt.Fprintf(w, "{\n  \"error\": %q,\n  \"code\": %d\n}\n", msg, status)
	}
}

type OrderedRow struct {
	Keys   []string
	Values []any
}

func derefInt(i *int) int {
	if i == nil {
		return 999999
	}
	return *i
}

func getOrderedKeys(record map[string]any, schema []domain.ColumnDef) []string {
	var keys []string
	if len(schema) > 0 {
		sortedSchema := make([]domain.ColumnDef, len(schema))
		copy(sortedSchema, schema)
		sort.Slice(sortedSchema, func(i, j int) bool {
			return derefInt(sortedSchema[i].CID) < derefInt(sortedSchema[j].CID)
		})
		seen := make(map[string]bool)
		for _, col := range sortedSchema {
			if _, exists := record[col.Name]; exists {
				keys = append(keys, col.Name)
				seen[col.Name] = true
			}
		}
		var extraKeys []string
		for k := range record {
			if !seen[k] {
				extraKeys = append(extraKeys, k)
			}
		}
		sort.Strings(extraKeys)
		keys = append(keys, extraKeys...)
	} else {
		for k := range record {
			keys = append(keys, k)
		}
		sort.Strings(keys)
	}
	return keys
}

func toOrderedRows(payload any, schema []domain.ColumnDef) ([]OrderedRow, bool, bool) {
	switch v := payload.(type) {
	case []domain.ColumnDef:
		sortedCols := make([]domain.ColumnDef, len(v))
		copy(sortedCols, v)
		sort.Slice(sortedCols, func(i, j int) bool {
			return derefInt(sortedCols[i].CID) < derefInt(sortedCols[j].CID)
		})
		var rows []OrderedRow
		for _, col := range sortedCols {
			var keys []string
			var vals []any
			keys = append(keys, "name")
			vals = append(vals, col.Name)
			keys = append(keys, "type")
			vals = append(vals, col.Type)
			if col.CID != nil {
				keys = append(keys, "cid")
				vals = append(vals, *col.CID)
			}
			if col.Nullable != nil {
				keys = append(keys, "nullable")
				vals = append(vals, *col.Nullable)
			}
			if col.Unique != nil {
				keys = append(keys, "unique")
				vals = append(vals, *col.Unique)
			}
			if col.Default != nil {
				keys = append(keys, "default")
				vals = append(vals, *col.Default)
			}
			if col.PK != nil {
				keys = append(keys, "pk")
				vals = append(vals, *col.PK)
			}
			if col.Autoincrement != nil {
				keys = append(keys, "autoincrement")
				vals = append(vals, *col.Autoincrement)
			}
			rows = append(rows, OrderedRow{Keys: keys, Values: vals})
		}
		return rows, true, true

	case []map[string]any:
		var rows []OrderedRow
		for _, record := range v {
			keys := getOrderedKeys(record, schema)
			var vals []any
			for _, k := range keys {
				vals = append(vals, record[k])
			}
			rows = append(rows, OrderedRow{Keys: keys, Values: vals})
		}
		return rows, true, true

	case map[string]any:
		keys := getOrderedKeys(v, schema)
		var vals []any
		for _, k := range keys {
			vals = append(vals, v[k])
		}
		return []OrderedRow{{Keys: keys, Values: vals}}, false, true

	default:
		return nil, false, false
	}
}

func escapeXMLText(s string) string {
	var buf bytes.Buffer
	_ = xml.EscapeText(&buf, []byte(s))
	return buf.String()
}

func formatJSONVal(val any) string {
	if val == nil {
		return "null"
	}
	switch v := val.(type) {
	case string:
		b, _ := json.Marshal(v)
		return string(b)
	case *string:
		if v == nil {
			return "null"
		}
		b, _ := json.Marshal(*v)
		return string(b)
	case bool:
		if v {
			return "true"
		}
		return "false"
	case *bool:
		if v == nil {
			return "null"
		}
		if *v {
			return "true"
		}
		return "false"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case *int:
		if v == nil {
			return "null"
		}
		return fmt.Sprintf("%d", *v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%q", fmt.Sprintf("%v", v))
		}
		return string(b)
	}
}

func formatYAMLVal(val any) string {
	if val == nil {
		return "null"
	}
	switch v := val.(type) {
	case string:
		if strings.ContainsAny(v, ":#{}[]\n\t\"'\\") || v == "" || v == "true" || v == "false" || v == "null" {
			b, _ := json.Marshal(v)
			return string(b)
		}
		return v
	case *string:
		if v == nil {
			return "null"
		}
		return formatYAMLVal(*v)
	case bool:
		return fmt.Sprintf("%t", v)
	case *bool:
		if v == nil {
			return "null"
		}
		return fmt.Sprintf("%t", *v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case *int:
		if v == nil {
			return "null"
		}
		return fmt.Sprintf("%d", *v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatTOMLVal(val any) string {
	if val == nil {
		return `""`
	}
	switch v := val.(type) {
	case string:
		b, _ := json.Marshal(v)
		return string(b)
	case *string:
		if v == nil {
			return `""`
		}
		b, _ := json.Marshal(*v)
		return string(b)
	case bool:
		return fmt.Sprintf("%t", v)
	case *bool:
		if v == nil {
			return "false"
		}
		return fmt.Sprintf("%t", *v)
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case *int:
		if v == nil {
			return "0"
		}
		return fmt.Sprintf("%d", *v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (e *ResponseEncoder) writeListOfStrings(w http.ResponseWriter, format domain.FormatType, tableName string, items []string, status int) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Type", format.ContentType())
	w.WriteHeader(status)

	switch format {
	case domain.FormatNDJSON:
		for _, item := range items {
			_, _ = fmt.Fprintf(w, "%q\n", item)
		}
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

	case domain.FormatYAML:
		_, _ = fmt.Fprintf(w, "%s:\n", tableName)
		for _, item := range items {
			_, _ = fmt.Fprintf(w, "  - %s\n", item)
		}

	case domain.FormatTOML:
		var quoted []string
		for _, item := range items {
			quoted = append(quoted, fmt.Sprintf("%q", item))
		}
		_, _ = fmt.Fprintf(w, "%s = [%s]\n", tableName, strings.Join(quoted, ", "))

	case domain.FormatXML:
		_, _ = w.Write([]byte(xml.Header))
		_, _ = fmt.Fprintf(w, "<%s>\n", tableName)
		for _, item := range items {
			_, _ = fmt.Fprintf(w, "  <row>%s</row>\n", escapeXMLText(item))
		}
		_, _ = fmt.Fprintf(w, "</%s>\n", tableName)

	case domain.FormatCSV:
		csvWriter := csv.NewWriter(w)
		_ = csvWriter.Write([]string{"table_name"})
		for _, item := range items {
			_ = csvWriter.Write([]string{item})
		}
		csvWriter.Flush()

	case domain.FormatJSON:
		fallthrough
	default:
		var quoted []string
		for _, item := range items {
			quoted = append(quoted, fmt.Sprintf("    %q", item))
		}
		if len(quoted) == 0 {
			_, _ = fmt.Fprintf(w, "{\n  %q: []\n}\n", tableName)
		} else {
			_, _ = fmt.Fprintf(w, "{\n  %q: [\n%s\n  ]\n}\n", tableName, strings.Join(quoted, ",\n"))
		}
	}
}

func (e *ResponseEncoder) EncodeResponse(w http.ResponseWriter, r *http.Request, status int, payload any, inputFormat domain.FormatType, tableName string, schema []domain.ColumnDef) {
	negotiated := e.NegotiateOutputFormat(r, inputFormat)

	if tableName == "" {
		switch payload.(type) {
		case []string:
			tableName = "tables"
		case []domain.ColumnDef:
			tableName = "columns"
		}
	}

	if tables, ok := payload.([]string); ok {
		e.writeListOfStrings(w, negotiated, tableName, tables, status)
		return
	}

	rows, isSlice, ok := toOrderedRows(payload, schema)
	if !ok {
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Content-Type", negotiated.ContentType())
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
		return
	}

	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Content-Type", negotiated.ContentType())
	w.WriteHeader(status)

	rowElementName := "row"
	if tableName == "columns" {
		rowElementName = "column"
	}

	switch negotiated {
	case domain.FormatCSV:
		if len(rows) == 0 {
			return
		}
		csvWriter := csv.NewWriter(w)
		_ = csvWriter.Write(rows[0].Keys)
		for _, row := range rows {
			var strVals []string
			for _, val := range row.Values {
				if val == nil {
					strVals = append(strVals, "")
				} else {
					strVals = append(strVals, fmt.Sprintf("%v", val))
				}
			}
			_ = csvWriter.Write(strVals)
		}
		csvWriter.Flush()

	case domain.FormatNDJSON:
		for _, row := range rows {
			var parts []string
			for i, k := range row.Keys {
				parts = append(parts, fmt.Sprintf("%q:%s", k, formatJSONVal(row.Values[i])))
			}
			_, _ = w.Write([]byte("{" + strings.Join(parts, ",") + "}\n"))
			if flusher, okFlusher := w.(http.Flusher); okFlusher {
				flusher.Flush()
			}
		}

	case domain.FormatXML:
		_, _ = w.Write([]byte(xml.Header))
		_, _ = fmt.Fprintf(w, "<%s>\n", tableName)
		if isSlice {
			for _, row := range rows {
				_, _ = fmt.Fprintf(w, "  <%s>\n", rowElementName)
				for i, k := range row.Keys {
					valStr := fmt.Sprintf("%v", row.Values[i])
					_, _ = fmt.Fprintf(w, "    <%s>%s</%s>\n", k, escapeXMLText(valStr), k)
				}
				_, _ = fmt.Fprintf(w, "  </%s>\n", rowElementName)
			}
		} else if len(rows) > 0 {
			for i, k := range rows[0].Keys {
				valStr := fmt.Sprintf("%v", rows[0].Values[i])
				_, _ = fmt.Fprintf(w, "  <%s>%s</%s>\n", k, escapeXMLText(valStr), k)
			}
		}
		_, _ = fmt.Fprintf(w, "</%s>\n", tableName)

	case domain.FormatTOML:
		if isSlice {
			for _, row := range rows {
				_, _ = fmt.Fprintf(w, "[[%s]]\n", tableName)
				for i, k := range row.Keys {
					_, _ = fmt.Fprintf(w, "%s = %s\n", k, formatTOMLVal(row.Values[i]))
				}
				_, _ = w.Write([]byte("\n"))
			}
		} else if len(rows) > 0 {
			// If it is desired to have an array of length 1, replace the following line with "[[%s]]"
			_, _ = fmt.Fprintf(w, "[%s]\n", tableName)
			for i, k := range rows[0].Keys {
				_, _ = fmt.Fprintf(w, "%s = %s\n", k, formatTOMLVal(rows[0].Values[i]))
			}
		}

	case domain.FormatYAML:
		if isSlice {
			_, _ = fmt.Fprintf(w, "%s:\n", tableName)
			for _, row := range rows {
				for i, k := range row.Keys {
					if i == 0 {
						_, _ = fmt.Fprintf(w, "  - %s: %s\n", k, formatYAMLVal(row.Values[i]))
					} else {
						_, _ = fmt.Fprintf(w, "    %s: %s\n", k, formatYAMLVal(row.Values[i]))
					}
				}
			}
		} else if len(rows) > 0 {
			_, _ = fmt.Fprintf(w, "%s:\n", tableName)
			for i, k := range rows[0].Keys {
				_, _ = fmt.Fprintf(w, "  %s: %s\n", k, formatYAMLVal(rows[0].Values[i]))
			}
		}

	case domain.FormatJSON:
		fallthrough
	default:
		if isSlice {
			var rowJSONs []string
			for _, row := range rows {
				var fieldParts []string
				for i, k := range row.Keys {
					fieldParts = append(fieldParts, fmt.Sprintf("      %q: %s", k, formatJSONVal(row.Values[i])))
				}
				rowJSONs = append(rowJSONs, fmt.Sprintf("    {\n%s\n    }", strings.Join(fieldParts, ",\n")))
			}
			if len(rowJSONs) == 0 {
				_, _ = fmt.Fprintf(w, "{\n  %q: []\n}\n", tableName)
			} else {
				_, _ = fmt.Fprintf(w, "{\n  %q: [\n%s\n  ]\n}\n", tableName, strings.Join(rowJSONs, ",\n"))
			}
		} else if len(rows) > 0 {
			var fieldParts []string
			for i, k := range rows[0].Keys {
				fieldParts = append(fieldParts, fmt.Sprintf("    %q: %s", k, formatJSONVal(rows[0].Values[i])))
			}
			_, _ = fmt.Fprintf(w, "{\n  %q: {\n%s\n  }\n}\n", tableName, strings.Join(fieldParts, ",\n"))
		}
	}
}
