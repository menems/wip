package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockRepo struct {
	users         map[string]*User     // keyed by email
	usersByID     map[uuid.UUID]*User  // keyed by id
	refreshTokens map[string]*RefreshToken // keyed by token_hash
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		users:         make(map[string]*User),
		usersByID:     make(map[uuid.UUID]*User),
		refreshTokens: make(map[string]*RefreshToken),
	}
}

func (m *mockRepo) addUser(u *User) {
	m.users[u.Email] = u
	m.usersByID[u.ID] = u
}

func (m *mockRepo) FindUserByEmail(_ context.Context, email string) (*User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (m *mockRepo) FindUserByID(_ context.Context, id uuid.UUID) (*User, error) {
	u, ok := m.usersByID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (m *mockRepo) SaveRefreshToken(_ context.Context, token *RefreshToken) error {
	m.refreshTokens[token.TokenHash] = token
	return nil
}

func (m *mockRepo) FindRefreshToken(_ context.Context, tokenHash string) (*RefreshToken, error) {
	t, ok := m.refreshTokens[tokenHash]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (m *mockRepo) RevokeRefreshToken(_ context.Context, id uuid.UUID) error {
	for _, t := range m.refreshTokens {
		if t.ID == id {
			now := time.Now()
			t.RevokedAt = &now
			return nil
		}
	}
	return ErrNotFound
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testService(t *testing.T, repo Repository) *Service {
	t.Helper()
	return NewService(repo, TokenConfig{
		Secret:     []byte("test-secret-that-is-long-enough-32c"),
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 720 * time.Hour,
	})
}

func hashPassword(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

func activeUser(t *testing.T) *User {
	t.Helper()
	return &User{
		ID:           uuid.New(),
		Email:        "alice@example.com",
		Name:         "Alice",
		PasswordHash: hashPassword(t, "secret123"),
		IsActive:     true,
		Roles:        []Role{{ID: uuid.New(), Name: "admin"}},
	}
}

// ---------------------------------------------------------------------------
// Login tests
// ---------------------------------------------------------------------------

func TestService_Login(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*mockRepo) *User
		email    string
		password string
		wantErr  error
	}{
		{
			name: "succeeds with correct credentials",
			setup: func(r *mockRepo) *User {
				u := activeUser(t)
				r.addUser(u)
				return u
			},
			email:    "alice@example.com",
			password: "secret123",
		},
		{
			name: "returns ErrInvalidCredentials for unknown email",
			setup: func(r *mockRepo) *User {
				return nil
			},
			email:    "nobody@example.com",
			password: "anything",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name: "returns ErrInvalidCredentials for wrong password",
			setup: func(r *mockRepo) *User {
				u := activeUser(t)
				r.addUser(u)
				return u
			},
			email:    "alice@example.com",
			password: "wrongpassword",
			wantErr:  ErrInvalidCredentials,
		},
		{
			name: "returns ErrAccountDeactivated for inactive user",
			setup: func(r *mockRepo) *User {
				u := activeUser(t)
				u.IsActive = false
				r.addUser(u)
				return u
			},
			email:    "alice@example.com",
			password: "secret123",
			wantErr:  ErrAccountDeactivated,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			tt.setup(repo)
			svc := testService(t, repo)

			result, err := svc.Login(context.Background(), tt.email, tt.password)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, result)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, result)
			assert.NotEmpty(t, result.AccessToken)
			assert.NotEmpty(t, result.RefreshToken)
			assert.Equal(t, tt.email, result.User.Email)

			// Refresh token must be stored (hashed) in the repo.
			hash := hashToken(result.RefreshToken)
			_, ok := repo.refreshTokens[hash]
			assert.True(t, ok, "refresh token hash should be saved in repo")
		})
	}
}

// ---------------------------------------------------------------------------
// Refresh tests
// ---------------------------------------------------------------------------

func TestService_Refresh(t *testing.T) {
	t.Run("rotates token and returns new token pair", func(t *testing.T) {
		repo := newMockRepo()
		u := activeUser(t)
		repo.addUser(u)
		svc := testService(t, repo)

		// Get an initial token pair via Login.
		first, err := svc.Login(context.Background(), u.Email, "secret123")
		require.NoError(t, err)

		// Refresh using the first raw refresh token.
		second, err := svc.Refresh(context.Background(), first.RefreshToken)
		require.NoError(t, err)
		require.NotNil(t, second)

		// New tokens must differ from old ones.
		assert.NotEqual(t, first.AccessToken, second.AccessToken)
		assert.NotEqual(t, first.RefreshToken, second.RefreshToken)

		// Old refresh token must be revoked.
		oldHash := hashToken(first.RefreshToken)
		assert.NotNil(t, repo.refreshTokens[oldHash].RevokedAt, "old token should be revoked")
	})

	t.Run("returns ErrTokenInvalid for unknown token", func(t *testing.T) {
		repo := newMockRepo()
		svc := testService(t, repo)

		_, err := svc.Refresh(context.Background(), "nonexistent-token")
		require.ErrorIs(t, err, ErrTokenInvalid)
	})

	t.Run("returns ErrTokenInvalid for revoked token", func(t *testing.T) {
		repo := newMockRepo()
		u := activeUser(t)
		repo.addUser(u)
		svc := testService(t, repo)

		first, err := svc.Login(context.Background(), u.Email, "secret123")
		require.NoError(t, err)

		// Revoke by refreshing once (rotation revokes the old token).
		_, err = svc.Refresh(context.Background(), first.RefreshToken)
		require.NoError(t, err)

		// Attempting to use the first token again must fail.
		_, err = svc.Refresh(context.Background(), first.RefreshToken)
		require.ErrorIs(t, err, ErrTokenInvalid)
	})

	t.Run("returns ErrTokenInvalid for expired token", func(t *testing.T) {
		repo := newMockRepo()
		u := activeUser(t)
		repo.addUser(u)
		svc := testService(t, repo)

		first, err := svc.Login(context.Background(), u.Email, "secret123")
		require.NoError(t, err)

		// Manually expire the stored token.
		hash := hashToken(first.RefreshToken)
		repo.refreshTokens[hash].ExpiresAt = time.Now().Add(-time.Hour)

		_, err = svc.Refresh(context.Background(), first.RefreshToken)
		require.ErrorIs(t, err, ErrTokenInvalid)
	})
}

// ---------------------------------------------------------------------------
// Logout tests
// ---------------------------------------------------------------------------

func TestService_Logout(t *testing.T) {
	t.Run("revokes the refresh token", func(t *testing.T) {
		repo := newMockRepo()
		u := activeUser(t)
		repo.addUser(u)
		svc := testService(t, repo)

		result, err := svc.Login(context.Background(), u.Email, "secret123")
		require.NoError(t, err)

		err = svc.Logout(context.Background(), result.RefreshToken)
		require.NoError(t, err)

		hash := hashToken(result.RefreshToken)
		assert.NotNil(t, repo.refreshTokens[hash].RevokedAt, "token should be revoked after logout")
	})

	t.Run("is idempotent for unknown token", func(t *testing.T) {
		repo := newMockRepo()
		svc := testService(t, repo)

		err := svc.Logout(context.Background(), "ghost-token")
		require.NoError(t, err)
	})
}

// ---------------------------------------------------------------------------
// VerifyAccessToken tests
// ---------------------------------------------------------------------------

func TestService_VerifyAccessToken(t *testing.T) {
	t.Run("parses a valid access token", func(t *testing.T) {
		repo := newMockRepo()
		u := activeUser(t)
		repo.addUser(u)
		svc := testService(t, repo)

		result, err := svc.Login(context.Background(), u.Email, "secret123")
		require.NoError(t, err)

		id, err := svc.VerifyAccessToken(result.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, u.ID, id)
	})

	t.Run("rejects a tampered token", func(t *testing.T) {
		repo := newMockRepo()
		svc := testService(t, repo)

		_, err := svc.VerifyAccessToken("this.is.not.a.jwt")
		require.Error(t, err)
	})
}
