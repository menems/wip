// Package audit implements read-only access to the immutable audit log.
// It follows hexagonal architecture: domain types and port interfaces live here.
// Audit log entries are append-only; no UPDATE or DELETE is permitted at the
// application layer.
package audit

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Actor is the slim user snapshot embedded in each audit log entry.
type Actor struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
}

// LogEntry is a single immutable audit record.
type LogEntry struct {
	ID           uuid.UUID       `json:"id"`
	Actor        Actor           `json:"actor"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *uuid.UUID      `json:"resource_id"`
	BeforeState  json.RawMessage `json:"before_state"`
	AfterState   json.RawMessage `json:"after_state"`
	IPAddress    string          `json:"ip_address"`
	CreatedAt    time.Time       `json:"created_at"`
}

// Filter carries the query parameters shared by the list and export endpoints.
type Filter struct {
	ActorID      *uuid.UUID // optional; exact match on actor_id
	ResourceType string     // optional; exact match on resource_type
	Action       string     // optional; exact match on action
	From         *time.Time // optional; lower bound on created_at (inclusive)
	To           *time.Time // optional; upper bound on created_at (inclusive)
	SortDir      string     // "asc" or "desc"; defaults to "desc" in the repository
	Page         int        // 1-based; used by List only
	PerPage      int        // max 100; used by List only
}

// Repository is the storage port the audit service depends on.
// Implementations live in repository.go.
type Repository interface {
	// List returns a page of audit log entries matching the filter and the total count.
	List(ctx context.Context, filter Filter) ([]*LogEntry, int, error)

	// Export returns all entries matching the filter without pagination.
	// Intended for CSV streaming — callers must not load the entire result into memory
	// when the result set is large; for v1 this is acceptable given expected log volumes.
	Export(ctx context.Context, filter Filter) ([]*LogEntry, error)
}
