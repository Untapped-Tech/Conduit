package impl

import (
	"context"
	"github.com/untappedtech/conduit/internal/domain"
)

type NoopAuth struct{}

func NewNoopAuth() *NoopAuth {
	return &NoopAuth{}
}

func (noopAuth *NoopAuth) Authorize(ctx context.Context, request domain.AuthRequest) (bool, bool, error) {
	return true, true, nil
}

func (noopAuth *NoopAuth) IsNoop() bool {
	return true
}

func (noopAuth *NoopAuth) Close() error {
	return nil
}
