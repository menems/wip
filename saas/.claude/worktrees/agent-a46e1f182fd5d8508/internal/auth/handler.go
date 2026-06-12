package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/menems/saas/gen/api/v1/pb"
	"github.com/menems/saas/gen/api/v1/pb/pbconnect"
)

// Handler implements the ConnectRPC AuthService.
type Handler struct {
	pbconnect.UnimplementedAuthServiceHandler
	svc AuthService
}

// NewHandler creates a new auth ConnectRPC handler.
func NewHandler(svc AuthService) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the handler onto mux with the given ConnectRPC options.
func (h *Handler) Register(mux *http.ServeMux, opts ...connect.HandlerOption) {
	path, handler := pbconnect.NewAuthServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// toConnectError maps domain sentinel errors to ConnectRPC error codes.
func toConnectError(err error) error {
	switch {
	case errors.Is(err, ErrUnauthenticated):
		return connect.NewError(connect.CodeUnauthenticated, err)
	case errors.Is(err, ErrValidation):
		return connect.NewError(connect.CodeInvalidArgument, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// Login authenticates a user by email and password, returning a JWT token.
func (h *Handler) Login(ctx context.Context, req *connect.Request[pb.LoginRequest]) (*connect.Response[pb.LoginResponse], error) {
	if req.Msg.GetEmail() == "" {
		return nil, toConnectError(fmt.Errorf("email is required: %w", ErrValidation))
	}
	if req.Msg.GetPassword() == "" {
		return nil, toConnectError(fmt.Errorf("password is required: %w", ErrValidation))
	}

	token, err := h.svc.Login(ctx, req.Msg.GetEmail(), req.Msg.GetPassword())
	if err != nil {
		return nil, toConnectError(fmt.Errorf("invalid credentials: %w", ErrUnauthenticated))
	}

	return connect.NewResponse(&pb.LoginResponse{
		Token: token,
	}), nil
}

// GetCurrentUser returns info about the currently authenticated user from context claims.
func (h *Handler) GetCurrentUser(ctx context.Context, _ *connect.Request[pb.GetCurrentUserRequest]) (*connect.Response[pb.GetCurrentUserResponse], error) {
	claims, err := ClaimsFromContext(ctx)
	if err != nil {
		return nil, toConnectError(fmt.Errorf("not authenticated: %w", ErrUnauthenticated))
	}

	return connect.NewResponse(&pb.GetCurrentUserResponse{
		UserId: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
	}), nil
}
