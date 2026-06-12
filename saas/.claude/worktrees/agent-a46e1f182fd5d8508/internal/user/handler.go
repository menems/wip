package user

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	"github.com/menems/saas/gen/api/v1/pb"
	"github.com/menems/saas/gen/api/v1/pb/pbconnect"
	"github.com/menems/saas/pkg/authz"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UserService defines the domain operations the handler depends on.
type UserService interface {
	Create(ctx context.Context, u User, password string) (User, error)
	List(ctx context.Context) ([]User, error)
	Get(ctx context.Context, id string) (User, error)
	Update(ctx context.Context, u User) (User, error)
	Delete(ctx context.Context, id string) error
}

// Handler implements the ConnectRPC UserService.
type Handler struct {
	pbconnect.UnimplementedUserServiceHandler
	svc UserService
}

// NewHandler creates a new user ConnectRPC handler.
func NewHandler(svc UserService) *Handler {
	return &Handler{svc: svc}
}

// Register mounts the handler onto mux with the given ConnectRPC options.
func (h *Handler) Register(mux *http.ServeMux, opts ...connect.HandlerOption) {
	path, handler := pbconnect.NewUserServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

// toConnectError maps domain sentinel errors to ConnectRPC error codes.
func toConnectError(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return connect.NewError(connect.CodeNotFound, err)
	case errors.Is(err, ErrConflict):
		return connect.NewError(connect.CodeAlreadyExists, err)
	case errors.Is(err, ErrValidation):
		return connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, ErrPermissionDenied):
		return connect.NewError(connect.CodePermissionDenied, err)
	default:
		return connect.NewError(connect.CodeInternal, err)
	}
}

// requireAdmin returns ErrPermissionDenied if the caller is not an admin.
func requireAdmin(ctx context.Context) error {
	claims, err := authz.ClaimsFromContext(ctx)
	if err != nil || claims.Role != string(RoleAdmin) {
		return fmt.Errorf("admin access required: %w", ErrPermissionDenied)
	}
	return nil
}

// domainToProto converts a domain User to its proto representation.
func domainToProto(u User) *pb.User {
	return &pb.User{
		Id:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      roleToProto(u.Role),
		CreatedAt: timestamppb.New(u.CreatedAt),
		UpdatedAt: timestamppb.New(u.UpdatedAt),
	}
}

// roleToProto converts a domain Role to a proto Role.
func roleToProto(r Role) pb.Role {
	switch r {
	case RoleAdmin:
		return pb.Role_ROLE_ADMIN
	case RoleMember:
		return pb.Role_ROLE_MEMBER
	default:
		return pb.Role_ROLE_UNSPECIFIED
	}
}

// roleFromProto converts a proto Role to a domain Role.
func roleFromProto(r pb.Role) Role {
	switch r {
	case pb.Role_ROLE_ADMIN:
		return RoleAdmin
	default:
		return RoleMember
	}
}

// CreateUser creates a new user (admin only).
func (h *Handler) CreateUser(ctx context.Context, req *connect.Request[pb.CreateUserRequest]) (*connect.Response[pb.CreateUserResponse], error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, toConnectError(err)
	}

	u := User{
		Email: req.Msg.GetEmail(),
		Name:  req.Msg.GetName(),
		Role:  roleFromProto(req.Msg.GetRole()),
	}

	created, err := h.svc.Create(ctx, u, req.Msg.GetPassword())
	if err != nil {
		return nil, toConnectError(err)
	}

	return connect.NewResponse(&pb.CreateUserResponse{
		User: domainToProto(created),
	}), nil
}

// ListUsers returns all users (admin only).
func (h *Handler) ListUsers(ctx context.Context, _ *connect.Request[pb.ListUsersRequest]) (*connect.Response[pb.ListUsersResponse], error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, toConnectError(err)
	}

	users, err := h.svc.List(ctx)
	if err != nil {
		return nil, toConnectError(err)
	}

	pbUsers := make([]*pb.User, len(users))
	for i, u := range users {
		pbUsers[i] = domainToProto(u)
	}

	return connect.NewResponse(&pb.ListUsersResponse{
		Users: pbUsers,
	}), nil
}

// GetUser returns a user by ID (admin only).
func (h *Handler) GetUser(ctx context.Context, req *connect.Request[pb.GetUserRequest]) (*connect.Response[pb.GetUserResponse], error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, toConnectError(err)
	}

	u, err := h.svc.Get(ctx, req.Msg.GetId())
	if err != nil {
		return nil, toConnectError(err)
	}

	return connect.NewResponse(&pb.GetUserResponse{
		User: domainToProto(u),
	}), nil
}

// UpdateUser modifies an existing user (admin only).
func (h *Handler) UpdateUser(ctx context.Context, req *connect.Request[pb.UpdateUserRequest]) (*connect.Response[pb.UpdateUserResponse], error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, toConnectError(err)
	}

	u := User{
		ID:    req.Msg.GetId(),
		Email: req.Msg.GetEmail(),
		Name:  req.Msg.GetName(),
		Role:  roleFromProto(req.Msg.GetRole()),
	}

	updated, err := h.svc.Update(ctx, u)
	if err != nil {
		return nil, toConnectError(err)
	}

	return connect.NewResponse(&pb.UpdateUserResponse{
		User: domainToProto(updated),
	}), nil
}

// DeleteUser removes a user by ID (admin only).
func (h *Handler) DeleteUser(ctx context.Context, req *connect.Request[pb.DeleteUserRequest]) (*connect.Response[pb.DeleteUserResponse], error) {
	if err := requireAdmin(ctx); err != nil {
		return nil, toConnectError(err)
	}

	if err := h.svc.Delete(ctx, req.Msg.GetId()); err != nil {
		return nil, toConnectError(err)
	}

	return connect.NewResponse(&pb.DeleteUserResponse{}), nil
}
