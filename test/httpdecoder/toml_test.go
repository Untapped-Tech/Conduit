package httpdecoder_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
	httpPkg "github.com/untappedtech/conduit/internal/http"
)

func TestDecoder_TOML(t *testing.T) {
	body := bytes.NewBufferString(`name = "alpha"`)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/toml")

	var sample Sample
	format, err := httpPkg.DecodeInputPayload(req, &sample)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if sample.Name != "alpha" {
		t.Fatalf("expected name=alpha")
	}
	if format != domain.FormatTOML {
		t.Fatalf("expected TOML format")
	}
}
