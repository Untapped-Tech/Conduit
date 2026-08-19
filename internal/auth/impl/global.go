package impl

import (
	"context"
	"github.com/untappedtech/conduit/internal/domain"
)

type GlobalPolicyAuth struct {
	publicReads    bool
	publicWrites   bool
	publicMutation bool
}

func NewGlobalPolicyAuth(policyConfig *domain.PolicyConfig) *GlobalPolicyAuth {
	return &GlobalPolicyAuth{
		publicReads:    policyConfig.PublicReads,
		publicWrites:   policyConfig.PublicWrites,
		publicMutation: policyConfig.PublicMutation,
	}
}

func (globalPolicy *GlobalPolicyAuth) Authorize(ctx context.Context, request domain.AuthRequest) (bool, bool, error) {
	switch request.Action {
	case domain.ActionReadTable:
		if globalPolicy.publicReads {
			return true, true, nil
		}
	case domain.ActionWriteTable:
		if globalPolicy.publicWrites {
			return true, true, nil
		}
	case domain.ActionMutateSchema:
		if globalPolicy.publicMutation {
			return true, true, nil
		}
	}
	return false, false, nil
}

func (globalPolicy *GlobalPolicyAuth) IsNoop() bool {
	return globalPolicy.publicReads && globalPolicy.publicWrites && globalPolicy.publicMutation
}

func (globalPolicy *GlobalPolicyAuth) Close() error {
	return nil
}
