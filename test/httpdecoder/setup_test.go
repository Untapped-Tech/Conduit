package httpdecoder_test

import (
    "bytes"
    "net/http"
    "net/http/httptest"

    httpPkg "github.com/untappedtech/conduit/internal/http"
    "github.com/untappedtech/conduit/internal/domain"
)

type Sample struct {
    Name string json:"name" yaml:"name" toml:"name" xml:"name"
}
