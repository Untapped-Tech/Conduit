package httpdecoder_test

import (
    "bytes"
    "net/http"
    "net/http/httptest"
    "testing"

    httpPkg "github.com/untappedtech/conduit/internal/http"
    "github.com/untappedtech/conduit/internal/domain"
)

func TestDecoder_CSVUnsupported(t *testing.T) {
    body := bytes.NewBufferString(\"name,alpha\")
    req := httptest.NewRequest(http.MethodPost, "/", body)
    req.Header.Set("Content-Type", "text/csv")

    var s Sample
    format, err := httpPkg.DecodeInputPayload(req, &s)
    if err == nil {
        t.Fatalf("expected CSV error")
    }
    if format != domain.FormatCSV {
        t.Fatalf("expected CSV format")
    }
}
