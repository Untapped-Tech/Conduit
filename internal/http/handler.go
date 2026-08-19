package http

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/untappedtech/conduit/internal/domain"
	"github.com/untappedtech/conduit/internal/service"
)

type APIHandler struct {
	apiService      *service.APIService
	serverConfig    *domain.ServerConfig
	responseEncoder *ResponseEncoder
}

func NewAPIHandler(apiService *service.APIService, serverConfig *domain.ServerConfig, responseEncoder *ResponseEncoder) *APIHandler {
	return &APIHandler{
		apiService:      apiService,
		serverConfig:    serverConfig,
		responseEncoder: responseEncoder,
	}
}

func (handler *APIHandler) RegisterRoutes(serveMux *http.ServeMux) {
	serveMux.HandleFunc("/v1/schema", handler.handleSchema)
	serveMux.HandleFunc("/v1/schema/", handler.handleSchema)
	serveMux.HandleFunc("/v1/", handler.handleCRUD)
}

func (handler *APIHandler) handleSchema(writer http.ResponseWriter, request *http.Request) {
	log.Printf("[HTTP] %s %s from %s", request.Method, request.URL.Path, request.RemoteAddr)

	requestPath := strings.TrimPrefix(request.URL.Path, "/v1/schema")
	tableName := strings.Trim(requestPath, "/")

	if tableName == "" && request.Method == http.MethodGet {
		tablesList, err := handler.apiService.ListTables(request.Context())
		if err != nil {
			log.Printf("[ERROR] Failed to list tables: %v", err)
			handler.responseEncoder.EncodeError(writer, request, http.StatusInternalServerError, "failed to list tables")
			return
		}
		handler.responseEncoder.EncodeResponse(writer, request, http.StatusOK, tablesList, domain.FormatJSON, "tables", nil)
		return
	}

	if tableName == "" {
		handler.responseEncoder.EncodeError(writer, request, http.StatusBadRequest, "invalid table identifier")
		return
	}

	switch request.Method {
	case http.MethodGet:
		columnDefinitions, err := handler.apiService.GetSchema(request.Context(), tableName)
		if err != nil {
			handler.responseEncoder.EncodeError(writer, request, http.StatusNotFound, "schema not found")
			return
		}
		handler.responseEncoder.EncodeResponse(writer, request, http.StatusOK, columnDefinitions, domain.FormatJSON, "columns", nil)

	case http.MethodPost:
		var schemaPayload struct {
			Columns []domain.ColumnDef `json:"columns" yaml:"columns" xml:"column" toml:"columns"`
		}
		inputFormat, decodeError := DecodeInputPayload(request, &schemaPayload)
		if decodeError != nil || len(schemaPayload.Columns) == 0 {
			log.Printf("[ERROR] Invalid schema payload for table %s: %v", tableName, decodeError)
			handler.responseEncoder.EncodeError(writer, request, http.StatusBadRequest, "invalid schema payload")
			return
		}

		if err := handler.apiService.CreateTable(request.Context(), tableName, schemaPayload.Columns); err != nil {
			log.Printf("[ERROR] Failed to create table %s: %v", tableName, err)
			handler.responseEncoder.EncodeError(writer, request, http.StatusInternalServerError, err.Error())
			return
		}

		log.Printf("[INFO] Table successfully created: %s", tableName)
		columnDefinitions, _ := handler.apiService.GetSchema(request.Context(), tableName)
		handler.responseEncoder.EncodeResponse(writer, request, http.StatusCreated, columnDefinitions, inputFormat, "columns", nil)

	case http.MethodDelete:
		if err := handler.apiService.DropTable(request.Context(), tableName); err != nil {
			log.Printf("[ERROR] Failed to drop table %s: %v", tableName, err)
			handler.responseEncoder.EncodeError(writer, request, http.StatusInternalServerError, err.Error())
			return
		}
		log.Printf("[INFO] Table successfully dropped: %s", tableName)
		writer.WriteHeader(http.StatusNoContent)

	default:
		handler.responseEncoder.EncodeError(writer, request, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (handler *APIHandler) handleCRUD(writer http.ResponseWriter, request *http.Request) {
	log.Printf("[HTTP] %s %s from %s", request.Method, request.URL.Path, request.RemoteAddr)

	requestPath := strings.TrimPrefix(request.URL.Path, "/v1/")
	pathParts := strings.Split(strings.Trim(requestPath, "/"), "/")

	if len(pathParts) == 0 || pathParts[0] == "" {
		handler.responseEncoder.EncodeError(writer, request, http.StatusNotFound, "not found")
		return
	}

	tableName := pathParts[0]
	requestContext := request.Context()

	if len(pathParts) == 1 && request.Method == http.MethodGet {
		queryLimit, _ := strconv.Atoi(request.URL.Query().Get("limit"))
		queryOffset, _ := strconv.Atoi(request.URL.Query().Get("offset"))

		recordSlice, err := handler.apiService.List(requestContext, tableName, queryLimit, queryOffset)
		if err != nil {
			log.Printf("[ERROR] Failed to list records for table %s: %v", tableName, err)
			handler.responseEncoder.EncodeError(writer, request, http.StatusInternalServerError, err.Error())
			return
		}

		schema, _ := handler.apiService.GetSchema(requestContext, tableName)
		handler.responseEncoder.EncodeResponse(writer, request, http.StatusOK, recordSlice, domain.FormatJSON, tableName, schema)
		return
	}

	if len(pathParts) == 2 {
		recordID := pathParts[1]
		switch request.Method {
		case http.MethodGet:
			singleRecord, err := handler.apiService.GetByID(requestContext, tableName, recordID)
			if err != nil {
				handler.responseEncoder.EncodeError(writer, request, http.StatusNotFound, "not found")
				return
			}
			schema, _ := handler.apiService.GetSchema(requestContext, tableName)
			handler.responseEncoder.EncodeResponse(writer, request, http.StatusOK, singleRecord, domain.FormatJSON, tableName, schema)

		case http.MethodPut:
			var recordPayload map[string]any
			inputFormat, decodeError := DecodeInputPayload(request, &recordPayload)
			if decodeError != nil {
				handler.responseEncoder.EncodeError(writer, request, http.StatusBadRequest, "invalid payload")
				return
			}
			updatedRecord, err := handler.apiService.Update(requestContext, tableName, recordID, recordPayload)
			if err != nil {
				log.Printf("[ERROR] Failed to update record %s in table %s: %v", recordID, tableName, err)
				handler.responseEncoder.EncodeError(writer, request, http.StatusInternalServerError, err.Error())
				return
			}
			schema, _ := handler.apiService.GetSchema(requestContext, tableName)
			handler.responseEncoder.EncodeResponse(writer, request, http.StatusOK, updatedRecord, inputFormat, tableName, schema)

		case http.MethodDelete:
			if err := handler.apiService.Delete(requestContext, tableName, recordID); err != nil {
				log.Printf("[ERROR] Failed to delete record %s in table %s: %v", recordID, tableName, err)
				handler.responseEncoder.EncodeError(writer, request, http.StatusInternalServerError, err.Error())
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		default:
			handler.responseEncoder.EncodeError(writer, request, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if len(pathParts) == 1 && request.Method == http.MethodPost {
		var recordPayload map[string]any
		inputFormat, decodeError := DecodeInputPayload(request, &recordPayload)
		if decodeError != nil {
			handler.responseEncoder.EncodeError(writer, request, http.StatusBadRequest, "invalid payload")
			return
		}
		insertedRecord, err := handler.apiService.Insert(requestContext, tableName, recordPayload)
		if err != nil {
			log.Printf("[ERROR] Failed to insert record into table %s: %v", tableName, err)
			handler.responseEncoder.EncodeError(writer, request, http.StatusInternalServerError, err.Error())
			return
		}
		schema, _ := handler.apiService.GetSchema(requestContext, tableName)
		handler.responseEncoder.EncodeResponse(writer, request, http.StatusCreated, insertedRecord, inputFormat, tableName, schema)
		return
	}

	handler.responseEncoder.EncodeError(writer, request, http.StatusNotFound, "not found")
}
