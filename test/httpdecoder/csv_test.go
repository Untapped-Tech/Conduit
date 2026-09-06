package httpdecoder_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
	httpPkg "github.com/untappedtech/conduit/internal/http"
)

func TestDecoder_CSV(t *testing.T) {
	body := bytes.NewBufferString("name\nalpha")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "text/csv")

	var sample Sample
	format, err := httpPkg.DecodeInputPayload(req, &sample)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if sample.Name != "alpha" {
		t.Fatalf("expected name=alpha")
	}
	if format != domain.FormatCSV {
		t.Fatalf("expected CSV format")
	}
}

func TestDecoder_CSVMap(t *testing.T) {
	body := bytes.NewBufferString("name,players\nGolf,4")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "text/csv")

	var record map[string]any
	format, err := httpPkg.DecodeInputPayload(req, &record)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if format != domain.FormatCSV {
		t.Fatalf("expected CSV format")
	}
	if record["name"] != "Golf" {
		t.Fatalf("expected name=Golf, got %#v", record["name"])
	}
	if record["players"] != int64(4) {
		t.Fatalf("expected players=4, got %#v", record["players"])
	}
}

func TestDecoder_CSVSchemaColumns(t *testing.T) {
	body := bytes.NewBufferString("name,type,pk,autoincrement\nid,integer,true,true\nname,text,false,false")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "text/csv")

	var schemaPayload struct {
		Columns []domain.ColumnDef `json:"columns"`
	}
	_, err := httpPkg.DecodeInputPayload(req, &schemaPayload)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(schemaPayload.Columns) != 2 {
		t.Fatalf("expected 2 columns, got %d", len(schemaPayload.Columns))
	}
	if schemaPayload.Columns[0].Name != "id" || schemaPayload.Columns[1].Name != "name" {
		t.Fatalf("unexpected columns: %#v", schemaPayload.Columns)
	}
}

func TestDecoder_CSVRequiresDataRow(t *testing.T) {
	body := bytes.NewBufferString("name")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "text/csv")

	var sample Sample
	_, err := httpPkg.DecodeInputPayload(req, &sample)
	if err == nil {
		t.Fatalf("expected CSV error for header-only payload")
	}
}
