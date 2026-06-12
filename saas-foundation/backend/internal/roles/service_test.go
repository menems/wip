package roles

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Mock repository
// ---------------------------------------------------------------------------

type mockRepo struct {
	roles          map[uuid.UUID]*Role
	nameIndex      map[string]uuid.UUID
	userCounts     map[uuid.UUID]int // role_id → user count
	errOnCreate    error
	errOnUpdate    error
	errOnDelete    error
	errOnUserCount error
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		roles:      make(map[uuid.UUID]*Role),
		nameIndex:  make(map[string]uuid.UUID),
		userCounts: make(map[uuid.UUID]int),
	}
}

func (m *mockRepo) addRole(r *Role) {
	cp := *r
	if cp.Permissions == nil {
		cp.Permissions = []Permission{}
	}
	m.roles[r.ID] = &cp
	m.nameIndex[r.Name] = r.ID
}

func (m *mockRepo) List(_ context.Context) ([]*Role, error) {
	out := make([]*Role, 0, len(m.roles))
	for _, r := range m.roles {
		cp := *r
		out = append(out, &cp)
	}
	return out, nil
}

func (m *mockRepo) FindByID(_ context.Context, id uuid.UUID) (*Role, error) {
	r, ok := m.roles[id]
	if !ok {
		return nil, ErrNotFound
	}
	cp := *r
	return &cp, nil
}

func (m *mockRepo) Create(_ context.Context, role *Role) error {
	if m.errOnCreate != nil {
		return m.errOnCreate
	}
	if _, exists := m.nameIndex[role.Name]; exists {
		return ErrNameConflict
	}
	cp := *role
	if cp.Permissions == nil {
		cp.Permissions = []Permission{}
	}
	m.roles[role.ID] = &cp
	m.nameIndex[role.Name] = role.ID
	return nil
}

func (m *mockRepo) Update(_ context.Context, role *Role) error {
	if m.errOnUpdate != nil {
		return m.errOnUpdate
	}
	existing, ok := m.roles[role.ID]
	if !ok {
		return ErrNotFound
	}
	if id, exists := m.nameIndex[role.Name]; exists && id != role.ID {
		return ErrNameConflict
	}
	delete(m.nameIndex, existing.Name)
	cp := *role
	m.roles[role.ID] = &cp
	m.nameIndex[role.Name] = role.ID
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id uuid.UUID) error {
	if m.errOnDelete != nil {
		return m.errOnDelete
	}
	r, ok := m.roles[id]
	if !ok {
		return ErrNotFound
	}
	delete(m.nameIndex, r.Name)
	delete(m.roles, id)
	return nil
}

func (m *mockRepo) CountUsersWithRole(_ context.Context, roleID uuid.UUID) (int, error) {
	if m.errOnUserCount != nil {
		return 0, m.errOnUserCount
	}
	return m.userCounts[roleID], nil
}

// ---------------------------------------------------------------------------
// Fixtures
// ---------------------------------------------------------------------------

func testService(t *testing.T, repo Repository) *Service {
	t.Helper()
	return NewService(repo)
}

func adminRole() *Role {
	return &Role{
		ID:       uuid.New(),
		Name:     "admin",
		IsSystem: true,
		Permissions: []Permission{
			{Resource: "users", Action: "read"},
			{Resource: "users", Action: "write"},
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func viewerRole() *Role {
	return &Role{
		ID:          uuid.New(),
		Name:        "viewer",
		IsSystem:    false,
		Permissions: []Permission{{Resource: "users", Action: "read"}},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestService_List(t *testing.T) {
	t.Run("returns all roles", func(t *testing.T) {
		repo := newMockRepo()
		repo.addRole(adminRole())
		repo.addRole(viewerRole())
		svc := testService(t, repo)

		roles, err := svc.List(context.Background())
		require.NoError(t, err)
		assert.Len(t, roles, 2)
	})
}

// ---------------------------------------------------------------------------
// Get
// ---------------------------------------------------------------------------

func TestService_Get(t *testing.T) {
	t.Run("returns role by ID", func(t *testing.T) {
		repo := newMockRepo()
		r := viewerRole()
		repo.addRole(r)
		svc := testService(t, repo)

		got, err := svc.Get(context.Background(), r.ID)
		require.NoError(t, err)
		assert.Equal(t, r.Name, got.Name)
	})

	t.Run("returns ErrNotFound for unknown ID", func(t *testing.T) {
		repo := newMockRepo()
		svc := testService(t, repo)

		_, err := svc.Get(context.Background(), uuid.New())
		require.ErrorIs(t, err, ErrNotFound)
	})
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestService_Create(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*mockRepo)
		req     CreateRequest
		wantErr error
	}{
		{
			name:  "creates role with valid permissions",
			setup: func(_ *mockRepo) {},
			req: CreateRequest{
				Name:        "editor",
				Description: "Can edit users",
				Permissions: []Permission{
					{Resource: "users", Action: "read"},
					{Resource: "users", Action: "write"},
				},
			},
		},
		{
			name:  "creates role with empty permissions",
			setup: func(_ *mockRepo) {},
			req:   CreateRequest{Name: "readonly"},
		},
		{
			name: "returns ErrNameConflict for duplicate name",
			setup: func(r *mockRepo) {
				r.addRole(&Role{ID: uuid.New(), Name: "viewer"})
			},
			req:     CreateRequest{Name: "viewer"},
			wantErr: ErrNameConflict,
		},
		{
			name:    "returns ErrValidation for unknown resource",
			setup:   func(_ *mockRepo) {},
			req:     CreateRequest{Name: "bad", Permissions: []Permission{{Resource: "invoices", Action: "read"}}},
			wantErr: ErrValidation,
		},
		{
			name:    "returns ErrValidation for invalid action on known resource",
			setup:   func(_ *mockRepo) {},
			req:     CreateRequest{Name: "bad", Permissions: []Permission{{Resource: "audit_logs", Action: "write"}}},
			wantErr: ErrValidation,
		},
		{
			name:  "deduplicates repeated permissions",
			setup: func(_ *mockRepo) {},
			req: CreateRequest{
				Name: "dup",
				Permissions: []Permission{
					{Resource: "users", Action: "read"},
					{Resource: "users", Action: "read"}, // duplicate
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockRepo()
			tt.setup(repo)
			svc := testService(t, repo)

			role, err := svc.Create(context.Background(), tt.req)

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, role)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, role)
			assert.Equal(t, tt.req.Name, role.Name)
			assert.False(t, role.IsSystem)

			// Deduplication check
			if tt.name == "deduplicates repeated permissions" {
				assert.Len(t, role.Permissions, 1)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestService_Update(t *testing.T) {
	t.Run("replaces name and permissions", func(t *testing.T) {
		repo := newMockRepo()
		r := viewerRole()
		repo.addRole(r)
		svc := testService(t, repo)

		updated, err := svc.Update(context.Background(), r.ID, UpdateRequest{
			Name:        "super-viewer",
			Description: "Updated",
			Permissions: []Permission{
				{Resource: "users", Action: "read"},
				{Resource: "roles", Action: "read"},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, "super-viewer", updated.Name)
		assert.Len(t, updated.Permissions, 2)
	})

	t.Run("returns ErrNotFound for unknown role", func(t *testing.T) {
		repo := newMockRepo()
		svc := testService(t, repo)

		_, err := svc.Update(context.Background(), uuid.New(), UpdateRequest{Name: "x"})
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("returns ErrNameConflict when name is taken by another role", func(t *testing.T) {
		repo := newMockRepo()
		r1 := viewerRole()
		r2 := &Role{ID: uuid.New(), Name: "editor"}
		repo.addRole(r1)
		repo.addRole(r2)
		svc := testService(t, repo)

		_, err := svc.Update(context.Background(), r1.ID, UpdateRequest{Name: "editor"})
		require.ErrorIs(t, err, ErrNameConflict)
	})

	t.Run("returns ErrValidation for invalid permission", func(t *testing.T) {
		repo := newMockRepo()
		r := viewerRole()
		repo.addRole(r)
		svc := testService(t, repo)

		_, err := svc.Update(context.Background(), r.ID, UpdateRequest{
			Name:        "viewer",
			Permissions: []Permission{{Resource: "audit_logs", Action: "delete"}},
		})
		require.ErrorIs(t, err, ErrValidation)
	})
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestService_Delete(t *testing.T) {
	t.Run("deletes an unused non-system role", func(t *testing.T) {
		repo := newMockRepo()
		r := viewerRole()
		repo.addRole(r)
		repo.userCounts[r.ID] = 0
		svc := testService(t, repo)

		err := svc.Delete(context.Background(), r.ID)
		require.NoError(t, err)

		_, notFound := repo.roles[r.ID]
		assert.False(t, notFound)
	})

	t.Run("returns ErrSystemRole for system roles", func(t *testing.T) {
		repo := newMockRepo()
		r := adminRole()
		repo.addRole(r)
		svc := testService(t, repo)

		err := svc.Delete(context.Background(), r.ID)
		require.ErrorIs(t, err, ErrSystemRole)
	})

	t.Run("returns ErrRoleInUse when role has assigned users", func(t *testing.T) {
		repo := newMockRepo()
		r := viewerRole()
		repo.addRole(r)
		repo.userCounts[r.ID] = 3
		svc := testService(t, repo)

		err := svc.Delete(context.Background(), r.ID)
		require.ErrorIs(t, err, ErrRoleInUse)
	})

	t.Run("returns ErrNotFound for unknown role", func(t *testing.T) {
		repo := newMockRepo()
		svc := testService(t, repo)

		err := svc.Delete(context.Background(), uuid.New())
		require.ErrorIs(t, err, ErrNotFound)
	})
}
