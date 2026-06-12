package users

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
	users              map[uuid.UUID]*User
	emailIndex         map[string]uuid.UUID
	activeAdminCount   int
	errOnCreate        error
	errOnUpdate        error
	errOnSetActive     error
	errOnUpdatePwd     error
	errOnCountAdmins   error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		users:      make(map[uuid.UUID]*User),
		emailIndex: make(map[string]uuid.UUID),
	}
}

func (m *mockRepo) addUser(u *User) {
	m.users[u.ID] = u
	m.emailIndex[u.Email] = u.ID
}

func (m *mockRepo) List(_ context.Context, _ UserFilter) ([]*User, int, error) {
	var out []*User
	for _, u := range m.users {
		cp := *u
		out = append(out, &cp)
	}
	return out, len(out), nil
}

func (m *mockRepo) FindByID(_ context.Context, id uuid.UUID) (*User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *mockRepo) FindByEmail(_ context.Context, email string) (*User, error) {
	id, ok := m.emailIndex[email]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *m.users[id]
	return &cp, nil
}

func (m *mockRepo) Create(_ context.Context, user *User, _ uuid.UUID) error {
	if m.errOnCreate != nil {
		return m.errOnCreate
	}
	if _, exists := m.emailIndex[user.Email]; exists {
		return ErrEmailConflict
	}
	cp := *user
	m.users[user.ID] = &cp
	m.emailIndex[user.Email] = user.ID
	return nil
}

func (m *mockRepo) Update(_ context.Context, user *User, _ uuid.UUID) error {
	if m.errOnUpdate != nil {
		return m.errOnUpdate
	}
	existing := m.users[user.ID]
	if existing == nil {
		return ErrNotFound
	}
	// Check email conflict against other users
	if id, exists := m.emailIndex[user.Email]; exists && id != user.ID {
		return ErrEmailConflict
	}
	// Remove old email index entry if email changed
	delete(m.emailIndex, existing.Email)
	cp := *user
	m.users[user.ID] = &cp
	m.emailIndex[user.Email] = user.ID
	return nil
}

func (m *mockRepo) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	if m.errOnUpdatePwd != nil {
		return m.errOnUpdatePwd
	}
	u, ok := m.users[id]
	if !ok {
		return ErrNotFound
	}
	u.PasswordHash = hash
	return nil
}

func (m *mockRepo) SetActive(_ context.Context, id uuid.UUID, active bool) (*User, error) {
	if m.errOnSetActive != nil {
		return nil, m.errOnSetActive
	}
	u, ok := m.users[id]
	if !ok {
		return nil, ErrNotFound
	}
	u.IsActive = active
	cp := *u
	return &cp, nil
}

func (m *mockRepo) CountActiveSystemRoleUsers(_ context.Context) (int, error) {
	if m.errOnCountAdmins != nil {
		return 0, m.errOnCountAdmins
	}
	return m.activeAdminCount, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func testService(t *testing.T, repo Repository) *Service {
	t.Helper()
	return NewService(repo, bcrypt.MinCost)
}

func adminUser() *User {
	return &User{
		ID:       uuid.New(),
		Email:    "admin@example.com",
		Name:     "Admin",
		IsActive: true,
		Roles:    []Role{{ID: uuid.New(), Name: "admin", IsSystem: true}},
	}
}

func regularUser() *User {
	return &User{
		ID:        uuid.New(),
		Email:     "user@example.com",
		Name:      "Regular",
		IsActive:  true,
		Roles:     []Role{{ID: uuid.New(), Name: "viewer", IsSystem: false}},
		CreatedAt: time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestService_Create(t *testing.T) {
	roleID := uuid.New()

	tests := []struct {
		name    string
		setup   func(*mockRepo)
		req     CreateParams
		wantErr error
	}{
		{
			name:  "creates user successfully",
			setup: func(_ *mockRepo) {},
			req:   CreateParams{Email: "new@example.com", Name: "New", Password: "password1", RoleID: roleID},
		},
		{
			name: "returns ErrEmailConflict for duplicate email",
			setup: func(r *mockRepo) {
				u := regularUser()
				u.Email = "dup@example.com"
				r.addUser(u)
			},
			req:     CreateParams{Email: "dup@example.com", Name: "Dup", Password: "password1", RoleID: roleID},
			wantErr: ErrEmailConflict,
		},
		{
			name:    "returns validation error for short password",
			setup:   func(_ *mockRepo) {},
			req:     CreateParams{Email: "new@example.com", Name: "New", Password: "short", RoleID: roleID},
			wantErr: ErrValidation,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			tt.setup(repo)
			svc := testService(t, repo)

			user, err := svc.Create(context.Background(), tt.req)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, user)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, user)
			assert.Equal(t, tt.req.Email, user.Email)
			assert.Equal(t, tt.req.Name, user.Name)
			assert.True(t, user.IsActive)
			assert.NotEmpty(t, user.ID)
		})
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestService_Update(t *testing.T) {
	t.Run("updates name and email successfully", func(t *testing.T) {
		repo := newMockRepo()
		u := regularUser()
		repo.addUser(u)
		svc := testService(t, repo)

		updated, err := svc.Update(context.Background(), u.ID, UpdateParams{
			Email:  "updated@example.com",
			Name:   "Updated Name",
			RoleID: uuid.New(),
		})

		require.NoError(t, err)
		assert.Equal(t, "updated@example.com", updated.Email)
		assert.Equal(t, "Updated Name", updated.Name)
	})

	t.Run("returns ErrNotFound for unknown user", func(t *testing.T) {
		repo := newMockRepo()
		svc := testService(t, repo)

		_, err := svc.Update(context.Background(), uuid.New(), UpdateParams{
			Email:  "x@example.com",
			Name:   "X",
			RoleID: uuid.New(),
		})

		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns ErrEmailConflict when new email is taken", func(t *testing.T) {
		repo := newMockRepo()
		u1 := regularUser()
		u1.Email = "alice@example.com"
		u2 := regularUser()
		u2.Email = "bob@example.com"
		repo.addUser(u1)
		repo.addUser(u2)
		svc := testService(t, repo)

		_, err := svc.Update(context.Background(), u1.ID, UpdateParams{
			Email:  "bob@example.com", // already taken by u2
			Name:   "Alice",
			RoleID: uuid.New(),
		})

		require.ErrorIs(t, err, ErrEmailConflict)
	})
}

// ---------------------------------------------------------------------------
// Deactivate
// ---------------------------------------------------------------------------

func TestService_Deactivate(t *testing.T) {
	t.Run("deactivates a non-admin user", func(t *testing.T) {
		repo := newMockRepo()
		u := regularUser()
		repo.addUser(u)
		svc := testService(t, repo)

		updated, err := svc.Deactivate(context.Background(), u.ID)

		require.NoError(t, err)
		assert.False(t, updated.IsActive)
	})

	t.Run("deactivates an admin when another admin exists", func(t *testing.T) {
		repo := newMockRepo()
		u := adminUser()
		repo.addUser(u)
		repo.activeAdminCount = 2 // another admin still active
		svc := testService(t, repo)

		updated, err := svc.Deactivate(context.Background(), u.ID)

		require.NoError(t, err)
		assert.False(t, updated.IsActive)
	})

	t.Run("returns ErrLastAdmin when user is sole active admin", func(t *testing.T) {
		repo := newMockRepo()
		u := adminUser()
		repo.addUser(u)
		repo.activeAdminCount = 1 // this is the only admin
		svc := testService(t, repo)

		_, err := svc.Deactivate(context.Background(), u.ID)

		require.ErrorIs(t, err, ErrLastAdmin)
	})

	t.Run("returns ErrNotFound for unknown user", func(t *testing.T) {
		repo := newMockRepo()
		svc := testService(t, repo)

		_, err := svc.Deactivate(context.Background(), uuid.New())

		require.ErrorIs(t, err, ErrNotFound)
	})
}

// ---------------------------------------------------------------------------
// Reactivate
// ---------------------------------------------------------------------------

func TestService_Reactivate(t *testing.T) {
	t.Run("reactivates an inactive user", func(t *testing.T) {
		repo := newMockRepo()
		u := regularUser()
		u.IsActive = false
		repo.addUser(u)
		svc := testService(t, repo)

		updated, err := svc.Reactivate(context.Background(), u.ID)

		require.NoError(t, err)
		assert.True(t, updated.IsActive)
	})

	t.Run("returns ErrNotFound for unknown user", func(t *testing.T) {
		repo := newMockRepo()
		svc := testService(t, repo)

		_, err := svc.Reactivate(context.Background(), uuid.New())

		require.ErrorIs(t, err, ErrNotFound)
	})
}

// ---------------------------------------------------------------------------
// ResetPassword
// ---------------------------------------------------------------------------

func TestService_ResetPassword(t *testing.T) {
	t.Run("updates password hash", func(t *testing.T) {
		repo := newMockRepo()
		u := regularUser()
		repo.addUser(u)
		svc := testService(t, repo)

		err := svc.ResetPassword(context.Background(), u.ID, "newpassword123")

		require.NoError(t, err)
		// Verify hash was stored and is valid bcrypt
		stored := repo.users[u.ID].PasswordHash
		require.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored), []byte("newpassword123")))
	})

	t.Run("returns validation error for short password", func(t *testing.T) {
		repo := newMockRepo()
		u := regularUser()
		repo.addUser(u)
		svc := testService(t, repo)

		err := svc.ResetPassword(context.Background(), u.ID, "short")

		require.ErrorIs(t, err, ErrValidation)
	})

	t.Run("returns ErrNotFound for unknown user", func(t *testing.T) {
		repo := newMockRepo()
		svc := testService(t, repo)

		err := svc.ResetPassword(context.Background(), uuid.New(), "longpassword123")

		require.ErrorIs(t, err, ErrNotFound)
	})
}
