package application

import (
	"context"
	"time"

	"github.com/banggok/sched_mind/backend/internal/portfolio/domain"
)

type PortfolioQuery struct {
	ProjectIDs []string
	Projection domain.Projection
	From       time.Time
	To         time.Time
}

type Repository interface {
	ActiveProjects(context.Context) ([]domain.ProjectOption, error)
	Portfolio(context.Context, PortfolioQuery) (*domain.Portfolio, error)
	ListSavedFilters(context.Context) ([]domain.SavedFilter, error)
	CreateSavedFilter(context.Context, domain.SavedFilter) (*domain.SavedFilter, error)
	UpdateSavedFilter(context.Context, string, int64, []string, time.Time) (*domain.SavedFilter, error)
	DeleteSavedFilter(context.Context, string, int64) error
}
