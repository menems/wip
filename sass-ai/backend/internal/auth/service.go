package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"sassai/backend/internal/user"

	"golang.org/x/crypto/bcrypt"
)

// RegisterParams is the input value-object for the Register use case.
type RegisterParams struct {
	Email    string
	Password string
	Name     string
}

// LoginParams is the input value-object for the Login use case.
type LoginParams struct {
	Email    string
	Password string
}

// AuthResult is the output of a successful auth operation.
type AuthResult struct {
	Token string
	User  *user.User
}

// Service is the inbound port for authentication use cases.
// The HTTP adapter depends on this interface, not the concrete type.
type Service interface {
	Register(ctx context.Context, p RegisterParams) (*AuthResult, error)
	Login(ctx context.Context, p LoginParams) (*AuthResult, error)
}

// authService is the application-layer implementation of Service.
type authService struct {
	users  user.Repository
	tokens TokenService
}

// NewService constructs an AuthService backed by the given ports.
func NewService(users user.Repository, tokens TokenService) Service {
	return &authService{users: users, tokens: tokens}
}

func (s *authService) Register(ctx context.Context, p RegisterParams) (*AuthResult, error) {
	p.Email = strings.TrimSpace(strings.ToLower(p.Email))
	if p.Email == "" {
		return nil, fmt.Errorf("email is required")
	}
	if len(p.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(p.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u, err := s.users.Create(ctx, p.Email, string(hash), p.Name)
	if err != nil {
		return nil, err // propagates user.ErrEmailTaken
	}

	token, err := s.tokens.Generate(u.ID, u.Email)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{Token: token, User: u}, nil
}

func (s *authService) Login(ctx context.Context, p LoginParams) (*AuthResult, error) {
	p.Email = strings.TrimSpace(strings.ToLower(p.Email))

	u, hash, err := s.users.FindByEmail(ctx, p.Email)
	if err != nil {
		if errors.Is(err, user.ErrNotFound) {
			// Return a generic error to avoid account enumeration.
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("find user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(p.Password)); err != nil {
		return nil, user.ErrNotFound // same generic error on bad password
	}

	token, err := s.tokens.Generate(u.ID, u.Email)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	return &AuthResult{Token: token, User: u}, nil
}
