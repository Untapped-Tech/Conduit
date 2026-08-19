package http

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/untappedtech/conduit/internal/domain"
	"gopkg.in/yaml.v3"
)

func DecodeInputPayload[T any](request *http.Request, targetObject *T) (domain.FormatType, error) {
	contentTypeHeader := strings.ToLower(request.Header.Get("Content-Type"))

	if strings.Contains(contentTypeHeader, "csv") {
		return domain.FormatCSV, fmt.Errorf("CSV input is not supported")
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
