package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// Repo is the persistence interface required by the service.
type Repo interface {
	CreateUser(ctx context.Context, u User, passwordHash string) error
	FindByEmail(ctx context.Context, email string) (User, string, error)
	CreateSession(ctx context.Context, userID uuid.UUID, token string) error
	FindUserByToken(ctx context.Context, token string) (uuid.UUID, error)
}

type svc struct{ repo Repo }

// NewService returns a Service backed by the provided Repo.
func NewService(r Repo) Service { return &svc{repo: r} }

func (s *svc) Register(ctx context.Context, name, email, password string) (User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}
	u := User{ID: uuid.New(), Name: name, Email: email}
	if err := s.repo.CreateUser(ctx, u, string(hash)); err != nil {
		return User{}, err
	}
	return u, nil
}

func (s *svc) SignIn(ctx context.Context, email, password string) (string, error) {
	u, hash, err := s.repo.FindByEmail(ctx, email)
	if err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", ErrBadCredentials
	}
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	if err := s.repo.CreateSession(ctx, u.ID, token); err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	return token, nil
}

func (s *svc) FindUserByToken(ctx context.Context, token string) (uuid.UUID, error) {
	return s.repo.FindUserByToken(ctx, token)
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
