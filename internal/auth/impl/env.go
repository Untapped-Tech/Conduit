package impl

import (
	"context"
	"os"

	"github.com/untappedtech/conduit/internal/domain"
)

type EnvAuthenticator struct {
	adminToken     string
	readWriteToken string
	readOnlyToken  string
}

func NewEnvAuthenticator() *EnvAuthenticator {
	return &EnvAuthenticator{
		adminToken:     os.Getenv("AUTH_TOKEN_ADMIN"),
		readWriteToken: os.Getenv("AUTH_TOKEN_READ_WRITE"),
		readOnlyToken:  os.Getenv("AUTH_TOKEN_READ_ONLY"),
	}
}

func (authenticator *EnvAuthenticator) Authorize(ctx context.Context, request domain.AuthRequest) (bool, bool, error) {
	if request.Token == "" {
		return false, false, nil
	}

	var userRole domain.Role
	if authenticator.adminToken != "" && request.Token == authenticator.adminToken {
		userRole = domain.RoleAdmin
	} else if authenticator.readWriteToken != "" && request.Token == authenticator.readWriteToken {
		userRole = domain.RoleReadWrite
	} else if authenticator.readOnlyToken != "" && request.Token == authenticator.readOnlyToken {
		userRole = domain.RoleReadOnly
	} else {
		return false, true, domain.ErrUnauthorized
	}

	switch userRole {
	case domain.RoleAdmin:
		return true, true, nil
	case domain.RoleReadWrite:
		return request.Action != domain.ActionMutateSchema, true, nil
	case domain.RoleReadOnly:
		return request.Action == domain.ActionReadTable, true, nil
	default:
		return false, true, domain.ErrUnauthorized
	}
}

func (authenticator *EnvAuthenticator) IsNoop() bool {
	return authenticator.adminToken == "" &&
		authenticator.readWriteToken == "" &&
		authenticator.readOnlyToken == ""
}

func (authenticator *EnvAuthenticator) Close() error {
	return nil
}
