package auth

import (
	"fmt"

	"github.com/untappedtech/conduit/internal/auth/impl"
	"github.com/untappedtech/conduit/internal/domain"
)

func BuildAuthChain(serverConfig *domain.ServerConfig) ([]domain.AuthProvider, error) {
	var authChain []domain.AuthProvider

	globalPolicyAuth := impl.NewGlobalPolicyAuth(&serverConfig.Policy)
	if !globalPolicyAuth.IsNoop() {
		authChain = append(authChain, globalPolicyAuth)
	}

	if serverConfig.Auth.EnvironmentEnabled {
		envAuthenticator := impl.NewEnvAuthenticator()
		if !envAuthenticator.IsNoop() {
			authChain = append(authChain, envAuthenticator)
		}
	}

	if serverConfig.Auth.DBEnabled {
		dbAuthenticator, err := impl.NewDBAuthenticator(&serverConfig.Auth.DBAuth)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize db auth plugin: %w", err)
		}
		if !dbAuthenticator.IsNoop() {
			authChain = append(authChain, dbAuthenticator)
		}
	}

	if len(authChain) == 0 {
		authChain = append(authChain, impl.NewNoopAuth())
	}

	return authChain, nil
}
