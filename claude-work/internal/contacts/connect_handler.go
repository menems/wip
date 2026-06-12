package contacts

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"
	addressbookv1 "github.com/blaz/serve/gen/addressbook/v1"
	"github.com/blaz/serve/gen/addressbook/v1/addressbookv1connect"
	"github.com/blaz/serve/platform/auth"
)

// ConnectHandler implements the ContactService Connect/gRPC handler.
type ConnectHandler struct {
	svc          Service
	interceptors []connect.Interceptor
}

// NewConnectHandler returns a ConnectHandler backed by the given Service.
// Optional interceptors (e.g. auth) are applied to every RPC.
func NewConnectHandler(s Service, interceptors ...connect.Interceptor) *ConnectHandler {
	return &ConnectHandler{svc: s, interceptors: interceptors}
}

// RegisterRoutes mounts the Connect/gRPC handler onto mux.
func (h *ConnectHandler) RegisterRoutes(mux *http.ServeMux) {
	opts := []connect.HandlerOption{}
	if len(h.interceptors) > 0 {
		opts = append(opts, connect.WithInterceptors(h.interceptors...))
	}
	path, handler := addressbookv1connect.NewContactServiceHandler(h, opts...)
	mux.Handle(path, handler)
}

func protoToAddress(a *addressbookv1.Address) Address {
	if a == nil {
		return Address{}
	}
	return Address{Street: a.Street, City: a.City, State: a.State, Zip: a.Zip, Country: a.Country}
}

func addressToProto(a Address) *addressbookv1.Address {
	if a == (Address{}) {
		return nil
	}
	return &addressbookv1.Address{Street: a.Street, City: a.City, State: a.State, Zip: a.Zip, Country: a.Country}
}

func (h *ConnectHandler) Add(ctx context.Context, req *connect.Request[addressbookv1.AddRequest]) (*connect.Response[addressbookv1.AddResponse], error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
	}
	m := req.Msg
	if m.Name == "" || m.Phone == "" || m.Email == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name, phone and email are required"))
	}
	if err := h.svc.Add(ctx, userID, Contact{Name: m.Name, Phone: m.Phone, Email: m.Email, Address: protoToAddress(m.Address)}); err != nil {
		if errors.Is(err, ErrConflict) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&addressbookv1.AddResponse{}), nil
}

func (h *ConnectHandler) List(ctx context.Context, req *connect.Request[addressbookv1.ListRequest]) (*connect.Response[addressbookv1.ListResponse], error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
	}
	cs, err := h.svc.List(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pb := make([]*addressbookv1.Contact, len(cs))
	for i, c := range cs {
		pb[i] = &addressbookv1.Contact{Name: c.Name, Phone: c.Phone, Email: c.Email, Address: addressToProto(c.Address)}
	}
	return connect.NewResponse(&addressbookv1.ListResponse{Contacts: pb}), nil
}

func (h *ConnectHandler) GetByName(ctx context.Context, req *connect.Request[addressbookv1.GetByNameRequest]) (*connect.Response[addressbookv1.GetByNameResponse], error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
	}
	c, err := h.svc.GetByName(ctx, userID, req.Msg.Name)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&addressbookv1.GetByNameResponse{
		Contact: &addressbookv1.Contact{Name: c.Name, Phone: c.Phone, Email: c.Email, Address: addressToProto(c.Address)},
	}), nil
}

func (h *ConnectHandler) Delete(ctx context.Context, req *connect.Request[addressbookv1.DeleteRequest]) (*connect.Response[addressbookv1.DeleteResponse], error) {
	userID, ok := auth.UserIDFromContext(ctx)
	if !ok {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("unauthenticated"))
	}
	if err := h.svc.Delete(ctx, userID, req.Msg.Name); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&addressbookv1.DeleteResponse{}), nil
}
