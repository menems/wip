package audit

import (
	"context"
	"fmt"
)

// Service implements the application use cases for audit log access.
// It depends only on the Repository port — no HTTP or DB types.
type Service struct {
	repo Repository
}

// NewService constructs an audit Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// List returns a paginated, filtered slice of audit log entries and the total count.
func (s *Service) List(ctx context.Context, filter Filter) ([]*LogEntry, int, error) {
	entries, total, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("audit: list: %w", err)
	}
	return entries, total, nil
}

// Export returns all audit log entries matching the filter without pagination.
// Results are intended for CSV streaming from the handler layer.
func (s *Service) Export(ctx context.Context, filter Filter) ([]*LogEntry, error) {
	entries, err := s.repo.Export(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("audit: export: %w", err)
	}
	return entries, nil
}
