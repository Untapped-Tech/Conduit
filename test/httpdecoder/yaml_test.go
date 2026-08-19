package httpdecoder_test

import (
    "bytes"
    "net/http"
    "net/http/httptest"
    "testing"

    httpPkg "github.com/untappedtech/conduit/internal/http"
    "github.com/untappedtech/conduit/internal/domain"
)

func TestDecoder_YAML(t *testing.T) {
    body := bytes.NewBufferString(\"name: alpha\")
    req := httptest.NewRequest(http.MethodPost, "/", body)
    req.Header.Set("Content-Type", "application/x-yaml")

    var s Sample
    format, err := httpPkg.DecodeInputPayload(req, &s)
    if err != nil {
        t.Fatalf("decode error: %v", err)
    }
    if s.Name != "alpha" {
        t.Fatalf("expected name=alpha")
    }
    if format != domain.FormatYAML {
        t.Fatalf("expected YAML format")
    }
}
