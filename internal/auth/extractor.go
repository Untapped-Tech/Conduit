package auth

import (
	"net/http"
	"strings"
)

type TokenExtractor interface {
	ExtractToken(request *http.Request) string
}

type DefaultTokenExtractor struct{}

func NewDefaultTokenExtractor() TokenExtractor {
	return &DefaultTokenExtractor{}
}

func (extractor *DefaultTokenExtractor) ExtractToken(request *http.Request) string {
	authorizationHeader := request.Header.Get("Authorization")
	if strings.HasPrefix(authorizationHeader, "Bearer ") {
		return strings.TrimPrefix(authorizationHeader, "Bearer ")
	}

	if apiKeyHeader := request.Header.Get("X-API-Key"); apiKeyHeader != "" {
		return apiKeyHeader
	}

	return request.URL.Query().Get("api_key")
}
