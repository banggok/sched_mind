package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/portfolio/domain"
)

type repositoryStub struct {
	query      PortfolioQuery
	portfolio  *domain.Portfolio
	created    domain.SavedFilter
	updatedIDs []string
	deleteID   string
	deleteVer  int64
}

func (repository *repositoryStub) ActiveProjects(context.Context) ([]domain.ProjectOption, error) {
	return []domain.ProjectOption{{ID: "project", Name: "Project", Status: "open"}}, nil
}
func (repository *repositoryStub) Portfolio(_ context.Context, query PortfolioQuery) (*domain.Portfolio, error) {
	repository.query = query
	if repository.portfolio == nil {
		return &domain.Portfolio{Projection: query.Projection}, nil
	}
	return repository.portfolio, nil
}
func (repository *repositoryStub) ListSavedFilters(context.Context) ([]domain.SavedFilter, error) {
	return nil, nil
}
func (repository *repositoryStub) CreateSavedFilter(_ context.Context, value domain.SavedFilter) (*domain.SavedFilter, error) {
	repository.created = value
	return &value, nil
}
func (repository *repositoryStub) UpdateSavedFilter(_ context.Context, id string, version int64, projectIDs []string, now time.Time) (*domain.SavedFilter, error) {
	repository.updatedIDs = append([]string{}, projectIDs...)
	return &domain.SavedFilter{ID: id, ProjectIDs: projectIDs, Version: version + 1, UpdatedAt: now}, nil
}
func (repository *repositoryStub) DeleteSavedFilter(_ context.Context, id string, version int64) error {
	repository.deleteID, repository.deleteVer = id, version
	return nil
}

func TestPortfolioNormalizesSelectionAndDateOnlyRange(t *testing.T) {
	repository := &repositoryStub{}
	service := NewService(repository)
	from := time.Date(2026, 8, 1, 13, 30, 0, 0, time.FixedZone("WIB", 7*60*60))
	to := time.Date(2026, 8, 3, 23, 0, 0, 0, time.FixedZone("WIB", 7*60*60))

	if _, err := service.Portfolio(context.Background(), PortfolioQuery{
		ProjectIDs: []string{" beta ", "alpha", "beta", ""},
		Projection: domain.Execution,
		From:       from,
		To:         to,
	}); err != nil {
		t.Fatal(err)
	}
	if len(repository.query.ProjectIDs) != 2 || repository.query.ProjectIDs[0] != "alpha" || repository.query.ProjectIDs[1] != "beta" {
		t.Fatalf("project IDs = %#v", repository.query.ProjectIDs)
	}
	if repository.query.From.Location() != time.UTC || repository.query.From.Hour() != 0 || repository.query.To.Hour() != 0 {
		t.Fatalf("normalized range = %s through %s", repository.query.From, repository.query.To)
	}
}

func TestPortfolioRejectsUnboundedRequests(t *testing.T) {
	service := NewService(&repositoryStub{})
	projects := make([]string, 101)
	for index := range projects {
		projects[index] = fmt.Sprintf("project-%03d", index)
	}
	_, err := service.Portfolio(context.Background(), PortfolioQuery{
		ProjectIDs: projects,
		Projection: domain.Execution,
		From:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:         time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, domain.ErrProjectLimit) {
		t.Fatalf("project limit error = %v", err)
	}

	_, err = service.Portfolio(context.Background(), PortfolioQuery{
		Projection: domain.Execution,
		From:       time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:         time.Date(2028, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	if !errors.Is(err, domain.ErrDateRangeInvalid) {
		t.Fatalf("date range error = %v", err)
	}
}

func TestSavedFilterWritesNormalizeIdentityAndOptimisticVersion(t *testing.T) {
	now := time.Date(2026, 8, 1, 4, 0, 0, 0, time.UTC)
	repository := &repositoryStub{}
	service := NewServiceWithDependencies(repository, func() time.Time { return now }, func() (string, error) { return "filter", nil })

	created, err := service.CreateSavedFilter(context.Background(), "  Delivery  ", []string{"beta", "alpha", "beta"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Delivery" || created.Version != 1 || len(created.ProjectIDs) != 2 || created.ProjectIDs[0] != "alpha" {
		t.Fatalf("created = %#v", created)
	}

	updated, err := service.UpdateSavedFilter(context.Background(), " filter ", 1, []string{"beta", "beta", "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != "filter" || updated.Version != 2 || len(repository.updatedIDs) != 2 {
		t.Fatalf("updated = %#v IDs=%#v", updated, repository.updatedIDs)
	}
	if err := service.DeleteSavedFilter(context.Background(), " filter ", 2); err != nil {
		t.Fatal(err)
	}
	if repository.deleteID != "filter" || repository.deleteVer != 2 {
		t.Fatalf("delete = %q v%d", repository.deleteID, repository.deleteVer)
	}
}
