package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/untappedtech/conduit/internal/domain"
)

type ErrorResponder interface {
	EncodeError(responseWriter http.ResponseWriter, httpRequest *http.Request, statusCode int, errorMessage string)
}

func AuthMiddleware(authChain []domain.AuthProvider, tokenExtractor TokenExtractor, errorResponder ErrorResponder) func(http.Handler) http.Handler {
	return func(nextHandler http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			extractedToken := tokenExtractor.ExtractToken(request)

			requestPath := strings.TrimPrefix(request.URL.Path, "/v1/")
			pathParts := strings.Split(strings.Trim(requestPath, "/"), "/")
			tableName := ""
			var targetAction domain.ActionType = domain.ActionReadTable

			if len(pathParts) > 0 && pathParts[0] == "schema" {
				targetAction = domain.ActionReadTable
				if request.Method == http.MethodPost || request.Method == http.MethodDelete {
					targetAction = domain.ActionMutateSchema
				}
				if len(pathParts) > 1 {
					tableName = pathParts[1]
				}
			} else if len(pathParts) > 0 {
				tableName = pathParts[0]
				if request.Method == http.MethodPost || request.Method == http.MethodPut || request.Method == http.MethodDelete {
					targetAction = domain.ActionWriteTable
				}
			}

			authReq := domain.AuthRequest{
				Token:  extractedToken,
				Action: targetAction,
				Table:  tableName,
			}

			authorized := false
			for _, provider := range authChain {
				allowed, handled, err := provider.Authorize(request.Context(), authReq)
				if err != nil {
					errorResponder.EncodeError(writer, request, http.StatusUnauthorized, "unauthorized access")
					return
				}
				if handled {
					authorized = allowed
					break
				}
			}

			if !authorized {
				errorResponder.EncodeError(writer, request, http.StatusUnauthorized, "unauthorized access")
				return
			}

			ctx := context.WithValue(request.Context(), domain.RoleContextKey, domain.RoleAdmin)
			nextHandler.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}
