package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DBRepository implements Repository against PostgreSQL via pgx.
type DBRepository struct {
	pool *pgxpool.Pool
}

// NewDBRepository constructs a DBRepository backed by the given pool.
func NewDBRepository(pool *pgxpool.Pool) *DBRepository {
	return &DBRepository{pool: pool}
}

// List returns a page of audit log entries matching the filter and the total count.
func (r *DBRepository) List(ctx context.Context, filter Filter) ([]*LogEntry, int, error) {
	where, args := buildWhere(filter)

	// Count query — no JOIN needed since all filter columns are on audit_logs.
	var total int
	err := r.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM audit_logs al "+where, args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("audit: repo: count: %w", err)
	}

	sortDir := "DESC"
	if strings.ToLower(filter.SortDir) == "asc" {
		sortDir = "ASC"
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	perPage := filter.PerPage
	if perPage < 1 || perPage > 100 {
		perPage = 25
	}
	offset := (page - 1) * perPage

	dataArgs := append(args, perPage, offset)
	limitPlaceholder := fmt.Sprintf("$%d", len(dataArgs)-1)
	offsetPlaceholder := fmt.Sprintf("$%d", len(dataArgs))

	dataQuery := fmt.Sprintf(`
		SELECT
			al.id, al.actor_id, u.name, u.email,
			al.action, al.resource_type, al.resource_id,
			al.before_state, al.after_state,
			al.ip_address::text,
			al.created_at
		FROM audit_logs al
		JOIN users u ON u.id = al.actor_id
		%s
		ORDER BY al.created_at %s
		LIMIT %s OFFSET %s
	`, where, sortDir, limitPlaceholder, offsetPlaceholder)

	rows, err := r.pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("audit: repo: list query: %w", err)
	}
	defer rows.Close()

	var entries []*LogEntry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("audit: repo: list scan: %w", err)
		}
		entries = append(entries, entry)
	}
	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("audit: repo: list rows: %w", err)
	}

	return entries, total, nil
}

// Export returns all audit log entries matching the filter without pagination.
func (r *DBRepository) Export(ctx context.Context, filter Filter) ([]*LogEntry, error) {
	where, args := buildWhere(filter)

	sortDir := "DESC"
	if strings.ToLower(filter.SortDir) == "asc" {
		sortDir = "ASC"
	}

	query := fmt.Sprintf(`
		SELECT
			al.id, al.actor_id, u.name, u.email,
			al.action, al.resource_type, al.resource_id,
			al.before_state, al.after_state,
			al.ip_address::text,
			al.created_at
		FROM audit_logs al
		JOIN users u ON u.id = al.actor_id
		%s
		ORDER BY al.created_at %s
	`, where, sortDir)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("audit: repo: export query: %w", err)
	}
	defer rows.Close()

	var entries []*LogEntry
	for rows.Next() {
		entry, err := scanEntry(rows)
		if err != nil {
			return nil, fmt.Errorf("audit: repo: export scan: %w", err)
		}
		entries = append(entries, entry)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("audit: repo: export rows: %w", err)
	}

	return entries, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// buildWhere constructs a SQL WHERE clause and positional argument list from the filter.
// All column references use the "al" alias (audit_logs).
func buildWhere(filter Filter) (string, []any) {
	var clauses []string
	var args []any

	if filter.ActorID != nil {
		args = append(args, *filter.ActorID)
		clauses = append(clauses, fmt.Sprintf("al.actor_id = $%d", len(args)))
	}
	if filter.ResourceType != "" {
		args = append(args, filter.ResourceType)
		clauses = append(clauses, fmt.Sprintf("al.resource_type = $%d", len(args)))
	}
	if filter.Action != "" {
		args = append(args, filter.Action)
		clauses = append(clauses, fmt.Sprintf("al.action = $%d", len(args)))
	}
	if filter.From != nil {
		args = append(args, *filter.From)
		clauses = append(clauses, fmt.Sprintf("al.created_at >= $%d", len(args)))
	}
	if filter.To != nil {
		args = append(args, *filter.To)
		clauses = append(clauses, fmt.Sprintf("al.created_at <= $%d", len(args)))
	}

	if len(clauses) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(clauses, " AND "), args
}

// rowScanner is satisfied by pgx.Row and pgx.Rows, letting scanEntry work for both.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanEntry reads a single audit log row into a LogEntry.
// The SELECT must project columns in this exact order:
//
//	al.id, al.actor_id, u.name, u.email,
//	al.action, al.resource_type, al.resource_id,
//	al.before_state, al.after_state, al.ip_address::text, al.created_at
func scanEntry(row rowScanner) (*LogEntry, error) {
	var e LogEntry
	var resourceID pgtype.UUID
	var beforeState, afterState []byte
	var ipAddress *string
	var createdAt time.Time

	err := row.Scan(
		&e.ID,
		&e.Actor.ID,
		&e.Actor.Name,
		&e.Actor.Email,
		&e.Action,
		&e.ResourceType,
		&resourceID,
		&beforeState,
		&afterState,
		&ipAddress,
		&createdAt,
	)
	if err != nil {
		return nil, err
	}

	if resourceID.Valid {
		uid := uuid.UUID(resourceID.Bytes)
		e.ResourceID = &uid
	}

	// json.RawMessage(nil) serialises as JSON null, which is the correct representation
	// for absent before/after state snapshots.
	if len(beforeState) > 0 {
		e.BeforeState = json.RawMessage(beforeState)
	}
	if len(afterState) > 0 {
		e.AfterState = json.RawMessage(afterState)
	}

	if ipAddress != nil {
		e.IPAddress = *ipAddress
	}

	e.CreatedAt = createdAt
	return &e, nil
}
