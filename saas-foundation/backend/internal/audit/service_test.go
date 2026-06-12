package audit

import (
	"context"
	"errors"
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
	entries     []*LogEntry
	total       int
	errOnList   error
	errOnExport error
}

func (m *mockRepo) List(_ context.Context, _ Filter) ([]*LogEntry, int, error) {
	if m.errOnList != nil {
		return nil, 0, m.errOnList
	}
	return m.entries, m.total, nil
}

func (m *mockRepo) Export(_ context.Context, _ Filter) ([]*LogEntry, error) {
	if m.errOnExport != nil {
		return nil, m.errOnExport
	}
	return m.entries, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func sampleEntry() *LogEntry {
	return &LogEntry{
		ID: uuid.New(),
		Actor: Actor{
			ID:    uuid.New(),
			Name:  "Jane Smith",
			Email: "jane@example.com",
		},
		Action:       "user.create",
		ResourceType: "user",
		BeforeState:  nil,
		AfterState:   []byte(`{"email":"new@example.com"}`),
		IPAddress:    "127.0.0.1",
		CreatedAt:    time.Now(),
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestService_List(t *testing.T) {
	tests := []struct {
		name      string
		repo      *mockRepo
		filter    Filter
		wantLen   int
		wantTotal int
		wantErr   bool
	}{
		{
			name:      "returns entries and total",
			repo:      &mockRepo{entries: []*LogEntry{sampleEntry(), sampleEntry()}, total: 2},
			filter:    Filter{Page: 1, PerPage: 25},
			wantLen:   2,
			wantTotal: 2,
		},
		{
			name:   "returns empty list when no entries exist",
			repo:   &mockRepo{entries: nil, total: 0},
			filter: Filter{Page: 1, PerPage: 25},
		},
		{
			name:    "propagates repository error",
			repo:    &mockRepo{errOnList: errors.New("db down")},
			filter:  Filter{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)
			entries, total, err := svc.List(context.Background(), tt.filter)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, entries, tt.wantLen)
			assert.Equal(t, tt.wantTotal, total)
		})
	}
}

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------

func TestService_Export(t *testing.T) {
	tests := []struct {
		name    string
		repo    *mockRepo
		wantLen int
		wantErr bool
	}{
		{
			name:    "returns all entries for export",
			repo:    &mockRepo{entries: []*LogEntry{sampleEntry(), sampleEntry()}, total: 2},
			wantLen: 2,
		},
		{
			name: "returns empty slice when no entries exist",
			repo: &mockRepo{entries: nil},
		},
		{
			name:    "propagates repository error",
			repo:    &mockRepo{errOnExport: errors.New("db error")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.repo)
			entries, err := svc.Export(context.Background(), Filter{})

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, entries, tt.wantLen)
		})
	}
}
