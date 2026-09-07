package httpdecoder_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/untappedtech/conduit/internal/domain"
	httpPkg "github.com/untappedtech/conduit/internal/http"
)

func TestDecoder_CBOR(t *testing.T) {
	data, err := cbor.Marshal(map[string]any{"name": "alpha"})
	if err != nil {
		t.Fatalf("failed to marshal cbor: %v", err)
	}

	body := bytes.NewReader(data)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/cbor")

	var sample Sample
	format, err := httpPkg.DecodeInputPayload(req, &sample)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if sample.Name != "alpha" {
		t.Fatalf("expected name=alpha, got %s", sample.Name)
	}
	if format != domain.FormatCBOR {
		t.Fatalf("expected CBOR format")
	}
}
