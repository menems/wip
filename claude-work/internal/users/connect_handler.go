package users

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	usersv1 "github.com/blaz/serve/gen/users/v1"
	"github.com/blaz/serve/gen/users/v1/usersv1connect"
)

// ConnectHandler implements the UserService Connect/gRPC handler.
type ConnectHandler struct {
	svc Service
}

// NewConnectHandler returns a ConnectHandler backed by the given Service.
func NewConnectHandler(s Service) *ConnectHandler {
	return &ConnectHandler{svc: s}
}

// RegisterRoutes mounts the Connect/gRPC handler onto mux.
func (h *ConnectHandler) RegisterRoutes(mux *http.ServeMux) {
	path, handler := usersv1connect.NewUserServiceHandler(h)
	mux.Handle(path, handler)
}

func (h *ConnectHandler) Register(ctx context.Context, req *connect.Request[usersv1.RegisterRequest]) (*connect.Response[usersv1.RegisterResponse], error) {
	m := req.Msg
	if m.Name == "" || m.Email == "" || m.Password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name, email and password are required"))
	}
	u, err := h.svc.Register(ctx, m.Name, m.Email, m.Password)
	if err != nil {
		if errors.Is(err, ErrConflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&usersv1.RegisterResponse{
		Id: u.ID.String(), Name: u.Name, Email: u.Email,
	}), nil
}

func (h *ConnectHandler) SignIn(ctx context.Context, req *connect.Request[usersv1.SignInRequest]) (*connect.Response[usersv1.SignInResponse], error) {
	m := req.Msg
	if m.Email == "" || m.Password == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and password are required"))
	}
	token, err := h.svc.SignIn(ctx, m.Email, m.Password)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, ErrBadCredentials) {
			return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid credentials"))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&usersv1.SignInResponse{Token: token}), nil
}
