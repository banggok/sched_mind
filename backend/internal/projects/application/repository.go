package application

import (
	"context"
	"time"

	"github.com/banggok/sched_mind/backend/internal/projects/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

// Store keeps transaction-shaped persistence operations together so application
// workflows can demand atomicity without importing GORM.
type Store interface {
	List(context.Context, listing.Query) (listing.Page[domain.Project], error)
	Find(context.Context, string) (*domain.Project, error)
	CreateNext(context.Context, string, string, time.Time) (*domain.Project, error)
	Update(context.Context, domain.Project) error
	DeleteChildless(context.Context, string) error
	ChangeStatus(context.Context, string, domain.Status, time.Time) (*domain.Project, error)
	MovePriority(context.Context, string, domain.PriorityDirection, time.Time, func(context.Context) error) (*domain.Project, error)
}

type Scheduler interface{ RecalculateActiveProjects(context.Context) error }

type NoopScheduler struct{}

func (NoopScheduler) RecalculateActiveProjects(context.Context) error { return nil }
