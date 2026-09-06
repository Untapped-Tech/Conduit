package http

import (
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/untappedtech/conduit/internal/domain"
	"gopkg.in/yaml.v3"
)

func DecodeInputPayload[T any](request *http.Request, targetObject *T) (domain.FormatType, error) {
	contentTypeHeader := strings.ToLower(request.Header.Get("Content-Type"))

	if strings.Contains(contentTypeHeader, "csv") {
		return domain.FormatCSV, decodeCSVPayload(request.Body, targetObject)
	}

	var parsedFormat domain.FormatType
	var decodeError error

	switch {
	case strings.Contains(contentTypeHeader, "yaml") || strings.Contains(contentTypeHeader, "yml"):
		parsedFormat = domain.FormatYAML
		decodeError = yaml.NewDecoder(request.Body).Decode(targetObject)
	case strings.Contains(contentTypeHeader, "toml"):
		parsedFormat = domain.FormatTOML
		decodeError = toml.NewDecoder(request.Body).Decode(targetObject)
	case strings.Contains(contentTypeHeader, "xml"):
		parsedFormat = domain.FormatXML
		decodeError = xml.NewDecoder(request.Body).Decode(targetObject)
	default:
		parsedFormat = domain.FormatJSON
		decodeError = json.NewDecoder(request.Body).Decode(targetObject)
	}

	return parsedFormat, decodeError
}

func decodeCSVPayload[T any](body io.Reader, targetObject *T) error {
	csvReader := csv.NewReader(body)
	csvReader.TrimLeadingSpace = true
	records, err := csvReader.ReadAll()
	if err != nil {
		return err
	}
	if len(records) < 2 {
		return fmt.Errorf("csv payload requires a header row and at least one data row")
	}

	headers := records[0]
	rows := make([]map[string]any, 0, len(records)-1)
	for _, record := range records[1:] {
		if csvRecordEmpty(record) {
			continue
		}
		row := make(map[string]any, len(headers))
		for index, header := range headers {
			trimmedHeader := strings.TrimSpace(header)
			if trimmedHeader == "" {
				continue
			}
			cell := ""
			if index < len(record) {
				cell = record[index]
			}
			row[trimmedHeader] = coerceCSVCell(cell)
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		return fmt.Errorf("csv payload requires a header row and at least one data row")
	}

	return assignCSVRows(rows, targetObject)
}

func assignCSVRows[T any](rows []map[string]any, targetObject *T) error {
	switch dest := any(targetObject).(type) {
	case *map[string]any:
		if len(rows) != 1 {
			return fmt.Errorf("csv payload must contain exactly one data row")
		}
		*dest = rows[0]
		return nil
	case *[]map[string]any:
		*dest = rows
		return nil
	}

	payload, err := csvPayloadForTarget(any(targetObject), rows)
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, targetObject)
}

func csvPayloadForTarget(target any, rows []map[string]any) (any, error) {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, fmt.Errorf("csv decode target must be a pointer")
	}

	elem := value.Elem()
	switch elem.Kind() {
	case reflect.Slice:
		return rows, nil
	case reflect.Map:
		if len(rows) != 1 {
			return nil, fmt.Errorf("csv payload must contain exactly one data row")
		}
		return rows[0], nil
	case reflect.Struct:
		if sliceField := singleSliceJSONField(elem.Type()); sliceField != "" {
			return map[string]any{sliceField: rows}, nil
		}
		if len(rows) != 1 {
			return nil, fmt.Errorf("csv payload must contain exactly one data row")
		}
		return rows[0], nil
	default:
		if len(rows) != 1 {
			return nil, fmt.Errorf("csv payload must contain exactly one data row")
		}
		return rows[0], nil
	}
}

func singleSliceJSONField(structType reflect.Type) string {
	sliceCount := 0
	fieldName := ""
	for index := 0; index < structType.NumField(); index++ {
		field := structType.Field(index)
		if !field.IsExported() || field.Type.Kind() != reflect.Slice {
			continue
		}
		sliceCount++
		fieldName = field.Name
		if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
			fieldName = strings.Split(tag, ",")[0]
		}
	}
	if sliceCount == 1 {
		return fieldName
	}
	return ""
}

func csvRecordEmpty(record []string) bool {
	for _, cell := range record {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func coerceCSVCell(cell string) any {
	trimmed := strings.TrimSpace(cell)
	if trimmed == "" {
		return nil
	}
	if strings.EqualFold(trimmed, "true") {
		return true
	}
	if strings.EqualFold(trimmed, "false") {
		return false
	}
	if intValue, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return intValue
	}
	if floatValue, err := strconv.ParseFloat(trimmed, 64); err == nil {
		return floatValue
	}
	return trimmed
}
