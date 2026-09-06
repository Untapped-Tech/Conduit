package httpdecoder_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/untappedtech/conduit/internal/domain"
	httpPkg "github.com/untappedtech/conduit/internal/http"
)

func TestDecoder_XML(t *testing.T) {
	body := bytes.NewBufferString("<Sample><name>alpha</name></Sample>")
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/xml")

	var sample Sample
	format, err := httpPkg.DecodeInputPayload(req, &sample)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if sample.Name != "alpha" {
		t.Fatalf("expected name=alpha")
	}
	if format != domain.FormatXML {
		t.Fatalf("expected XML format")
	}
}
