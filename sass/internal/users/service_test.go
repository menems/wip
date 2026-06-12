// Tests live in package users (not users_test) because the mock types
// implement unexported repository interfaces defined in service.go.
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
// In-memory test doubles
// ---------------------------------------------------------------------------

type mockStore struct {
	byID    map[uuid.UUID]*User
	byEmail map[string]*User
	admins  int // simulated count of active system-role users
}

func newMockStore() *mockStore {
	return &mockStore{
		byID:    make(map[uuid.UUID]*User),
		byEmail: make(map[string]*User),
		admins:  0,
	}
}

func (m *mockStore) add(u *User) {
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	for _, r := range u.Roles {
		if r.IsSystem {
			m.admins++
			break
		}
	}
}

// userReader implementation

func (m *mockStore) FindByID(_ context.Context, id uuid.UUID) (*User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (m *mockStore) FindByEmail(_ context.Context, email string) (*User, error) {
	u, ok := m.byEmail[email]
	if !ok {
		return nil, ErrNotFound
	}
	return u, nil
}

func (m *mockStore) List(_ context.Context, _ UserFilter) ([]*User, int, error) {
	out := make([]*User, 0, len(m.byID))
	for _, u := range m.byID {
		out = append(out, u)
	}
	return out, len(out), nil
}

// userWriter implementation

func (m *mockStore) Create(_ context.Context, u *User, _ uuid.UUID) error {
	if _, exists := m.byEmail[u.Email]; exists {
		return ErrEmailConflict
	}
	m.byID[u.ID] = u
	m.byEmail[u.Email] = u
	return nil
}

func (m *mockStore) Update(_ context.Context, u *User, _ uuid.UUID) error {
	existing, ok := m.byID[u.ID]
	if !ok {
		return ErrNotFound
	}
	if existing.Email != u.Email {
		if _, conflict := m.byEmail[u.Email]; conflict {
			return ErrEmailConflict
		}
		delete(m.byEmail, existing.Email)
		m.byEmail[u.Email] = u
	}
	m.byID[u.ID] = u
	return nil
}

// userLifecycle implementation

func (m *mockStore) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	u, ok := m.byID[id]
	if !ok {
		return ErrNotFound
	}
	u.PasswordHash = hash
	return nil
}

func (m *mockStore) SetActive(_ context.Context, id uuid.UUID, active bool) (*User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, ErrNotFound
	}
	u.IsActive = active
	return u, nil
}

func (m *mockStore) CountActiveSystemRoleUsers(_ context.Context) (int, error) {
	return m.admins, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func hashPwd(t *testing.T, plain string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.MinCost)
	require.NoError(t, err)
	return string(h)
}

func adminUser(t *testing.T) *User {
	t.Helper()
	return &User{
		ID:           uuid.New(),
		Email:        "alice@example.com",
		Name:         "Alice",
		PasswordHash: hashPwd(t, "secret123"),
		IsActive:     true,
		Roles:        []Role{{ID: uuid.New(), Name: "admin", IsSystem: true}},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func regularUser(t *testing.T) *User {
	t.Helper()
	return &User{
		ID:           uuid.New(),
		Email:        "bob@example.com",
		Name:         "Bob",
		PasswordHash: hashPwd(t, "pass456"),
		IsActive:     true,
		Roles:        []Role{{ID: uuid.New(), Name: "viewer", IsSystem: false}},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func newService(store *mockStore) *UserService {
	return NewUserService(store, store, store)
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestUserService_List(t *testing.T) {
	t.Parallel()
	t.Run("returns all users", func(t *testing.T) {
		store := newMockStore()
		store.add(adminUser(t))
		store.add(regularUser(t))
		svc := newService(store)

		users, total, err := svc.List(context.Background(), UserFilter{})
		require.NoError(t, err)
		assert.Equal(t, 2, total)
		assert.Len(t, users, 2)
	})
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestUserService_Get(t *testing.T) {
	t.Parallel()
	t.Run("returns user by ID", func(t *testing.T) {
		store := newMockStore()
		u := adminUser(t)
		store.add(u)
		svc := newService(store)

		got, err := svc.Get(context.Background(), u.ID)
		require.NoError(t, err)
		assert.Equal(t, u.Email, got.Email)
	})

	t.Run("returns ErrNotFound for unknown ID", func(t *testing.T) {
		svc := newService(newMockStore())
		_, err := svc.Get(context.Background(), uuid.New())
		require.ErrorIs(t, err, ErrNotFound)
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestUserService_Create(t *testing.T) {
	t.Parallel()
	t.Run("creates user and hashes password", func(t *testing.T) {
		svc := newService(newMockStore())
		req := CreateRequest{
			Email:    "new@example.com",
			Name:     "New",
			Password: "plaintext",
			RoleID:   uuid.New(),
		}

		user, err := svc.Create(context.Background(), req)
		require.NoError(t, err)
		assert.Equal(t, req.Email, user.Email)
		assert.True(t, user.IsActive)
		assert.NotEmpty(t, user.PasswordHash)
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)))
	})

	t.Run("returns ErrEmailConflict for duplicate email", func(t *testing.T) {
		store := newMockStore()
		u := regularUser(t)
		store.add(u)
		svc := newService(store)

		_, err := svc.Create(context.Background(), CreateRequest{
			Email:    u.Email,
			Name:     "Dupe",
			Password: "pass",
			RoleID:   uuid.New(),
		})
		require.ErrorIs(t, err, ErrEmailConflict)
	})
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestUserService_Update(t *testing.T) {
	t.Parallel()
	t.Run("updates name and email", func(t *testing.T) {
		store := newMockStore()
		u := regularUser(t)
		store.add(u)
		svc := newService(store)

		updated, err := svc.Update(context.Background(), u.ID, UpdateRequest{
			Name:   "Bobby",
			Email:  "bobby@example.com",
			RoleID: u.Roles[0].ID,
		})
		require.NoError(t, err)
		assert.Equal(t, "Bobby", updated.Name)
		assert.Equal(t, "bobby@example.com", updated.Email)
	})

	t.Run("returns ErrNotFound for unknown ID", func(t *testing.T) {
		svc := newService(newMockStore())
		_, err := svc.Update(context.Background(), uuid.New(), UpdateRequest{Name: "X", Email: "x@x.com"})
		require.ErrorIs(t, err, ErrNotFound)
	})
}

// ---------------------------------------------------------------------------
// SetActive
// ---------------------------------------------------------------------------

func TestUserService_SetActive(t *testing.T) {
	t.Parallel()
	t.Run("deactivates a regular user", func(t *testing.T) {
		store := newMockStore()
		u := regularUser(t)
		store.add(u)
		svc := newService(store)

		got, err := svc.SetActive(context.Background(), u.ID, false)
		require.NoError(t, err)
		assert.False(t, got.IsActive)
	})

	t.Run("returns ErrLastAdmin when deactivating sole admin", func(t *testing.T) {
		store := newMockStore()
		u := adminUser(t)
		store.add(u)
		svc := newService(store)

		_, err := svc.SetActive(context.Background(), u.ID, false)
		require.ErrorIs(t, err, ErrLastAdmin)
	})

	t.Run("allows deactivation when other admins exist", func(t *testing.T) {
		store := newMockStore()
		u1 := adminUser(t)
		u2 := &User{
			ID:       uuid.New(),
			Email:    "carol@example.com",
			Name:     "Carol",
			IsActive: true,
			Roles:    []Role{{ID: uuid.New(), Name: "admin", IsSystem: true}},
		}
		store.add(u1)
		store.add(u2)
		svc := newService(store)

		got, err := svc.SetActive(context.Background(), u1.ID, false)
		require.NoError(t, err)
		assert.False(t, got.IsActive)
	})
}

// ---------------------------------------------------------------------------
// ChangePassword
// ---------------------------------------------------------------------------

func TestUserService_ChangePassword(t *testing.T) {
	t.Parallel()
	t.Run("changes password with correct old password", func(t *testing.T) {
		store := newMockStore()
		u := regularUser(t)
		store.add(u)
		svc := newService(store)

		err := svc.ChangePassword(context.Background(), u.ID, "pass456", "newpass")
		require.NoError(t, err)

		stored := store.byID[u.ID]
		assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("newpass")))
	})

	t.Run("returns ErrInvalidPassword for wrong old password", func(t *testing.T) {
		store := newMockStore()
		u := regularUser(t)
		store.add(u)
		svc := newService(store)

		err := svc.ChangePassword(context.Background(), u.ID, "wrong", "newpass")
		require.ErrorIs(t, err, ErrInvalidPassword)
	})

	t.Run("returns ErrNotFound for unknown user", func(t *testing.T) {
		svc := newService(newMockStore())
		err := svc.ChangePassword(context.Background(), uuid.New(), "old", "new")
		require.ErrorIs(t, err, ErrNotFound)
	})
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestUserService_Delete(t *testing.T) {
	t.Parallel()
	t.Run("soft-deletes a regular user", func(t *testing.T) {
		store := newMockStore()
		u := regularUser(t)
		store.add(u)
		svc := newService(store)

		err := svc.Delete(context.Background(), u.ID)
		require.NoError(t, err)
		assert.False(t, store.byID[u.ID].IsActive)
	})

	t.Run("returns ErrLastAdmin for sole admin", func(t *testing.T) {
		store := newMockStore()
		u := adminUser(t)
		store.add(u)
		svc := newService(store)

		err := svc.Delete(context.Background(), u.ID)
		require.ErrorIs(t, err, ErrLastAdmin)
	})

	t.Run("returns ErrNotFound for unknown user", func(t *testing.T) {
		svc := newService(newMockStore())
		err := svc.Delete(context.Background(), uuid.New())
		require.ErrorIs(t, err, ErrNotFound)
	})
}
