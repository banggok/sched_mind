package application

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/portfolio/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/identity"
)

const (
	maximumSelectedProjects = 100
	maximumVisibleDays      = 730
)

type Service struct {
	repository Repository
	now        func() time.Time
	newID      func() (string, error)
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: func() time.Time { return time.Now().UTC() }, newID: identity.NewUUID}
}

func NewServiceWithDependencies(repository Repository, now func() time.Time, newID func() (string, error)) *Service {
	return &Service{repository: repository, now: now, newID: newID}
}

func (service *Service) ActiveProjects(ctx context.Context) ([]domain.ProjectOption, error) {
	values, err := service.repository.ActiveProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active portfolio projects: %w", err)
	}
	return values, nil
}

func (service *Service) Portfolio(ctx context.Context, query PortfolioQuery) (*domain.Portfolio, error) {
	query.ProjectIDs = normalizeIDs(query.ProjectIDs)
	if len(query.ProjectIDs) > maximumSelectedProjects {
		return nil, domain.ErrProjectLimit
	}
	if query.Projection != domain.Execution && query.Projection != domain.Commitment {
		return nil, domain.ErrProjectionInvalid
	}
	query.From = dateOnly(query.From)
	query.To = dateOnly(query.To)
	if query.To.Before(query.From) || int(query.To.Sub(query.From).Hours()/24)+1 > maximumVisibleDays {
		return nil, domain.ErrDateRangeInvalid
	}
	value, err := service.repository.Portfolio(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("load portfolio projection: %w", err)
	}
	if value == nil {
		return nil, errors.New("load portfolio projection: repository returned nil")
	}
	return value, nil
}

func (service *Service) ListSavedFilters(ctx context.Context) ([]domain.SavedFilter, error) {
	values, err := service.repository.ListSavedFilters(ctx)
	if err != nil {
		return nil, fmt.Errorf("list saved filters: %w", err)
	}
	return values, nil
}

func (service *Service) CreateSavedFilter(ctx context.Context, name string, projectIDs []string) (*domain.SavedFilter, error) {
	name, _, err := domain.NormalizeFilterName(name)
	if err != nil {
		return nil, err
	}
	id, err := service.newID()
	if err != nil {
		return nil, fmt.Errorf("generate saved filter ID: %w", err)
	}
	now := service.now()
	value := domain.SavedFilter{ID: id, Name: name, ProjectIDs: normalizeIDs(projectIDs), Version: 1, CreatedAt: now, UpdatedAt: now}
	created, err := service.repository.CreateSavedFilter(ctx, value)
	if err != nil {
		return nil, fmt.Errorf("create saved filter: %w", err)
	}
	return created, nil
}

func (service *Service) UpdateSavedFilter(ctx context.Context, id string, version int64, projectIDs []string) (*domain.SavedFilter, error) {
	if version < 1 {
		return nil, domain.ErrFilterConflict
	}
	value, err := service.repository.UpdateSavedFilter(ctx, strings.TrimSpace(id), version, normalizeIDs(projectIDs), service.now())
	if err != nil {
		return nil, fmt.Errorf("update saved filter: %w", err)
	}
	return value, nil
}

func (service *Service) DeleteSavedFilter(ctx context.Context, id string, version int64) error {
	if version < 1 {
		return domain.ErrFilterConflict
	}
	if err := service.repository.DeleteSavedFilter(ctx, strings.TrimSpace(id), version); err != nil {
		return fmt.Errorf("delete saved filter: %w", err)
	}
	return nil
}

func normalizeIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
