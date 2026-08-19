package domain

import (
	"context"
	"errors"
)

var (
	ErrUnauthorized = errors.New("unauthorized access")
	ErrForbidden    = errors.New("insufficient privileges for operation")
)

type Role string

const (
	RoleAdmin     Role = "Admin"
	RoleReadWrite Role = "Read_Write"
	RoleReadOnly  Role = "Read_Only"
)

type ActionType int

const (
	ActionReadTable ActionType = iota
	ActionWriteTable
	ActionMutateSchema
)

type AuthRequest struct {
	Token  string
	Action ActionType
	Table  string
}

type AuthProvider interface {
	Authorize(ctx context.Context, request AuthRequest) (allowed bool, handled bool, err error)
	IsNoop() bool
	Close() error
}

type contextKey string

const RoleContextKey contextKey = "user_role"

func RoleFromContext(ctx context.Context) Role {
	if userRole, ok := ctx.Value(RoleContextKey).(Role); ok {
		return userRole
	}
	return RoleReadOnly
}
