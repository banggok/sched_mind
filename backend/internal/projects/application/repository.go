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
	CreateNext(context.Context, string, string, bool, *time.Time, int, time.Time) (*domain.Project, error)
	UpdateDetails(context.Context, string, string, bool, *time.Time, int, time.Time, func(context.Context, string) error, func(context.Context, string, string) error) (*domain.Project, error)
	DeleteChildless(context.Context, string) error
	ChangeStatus(context.Context, string, domain.Status, time.Time, func(context.Context) error) (*domain.Project, error)
	BulkReopen(context.Context, string, string, time.Time, func(context.Context) error) ([]domain.Project, error)
	MovePriority(context.Context, string, domain.PriorityDirection, time.Time, func(context.Context) error) (*domain.Project, error)
	UpdateSettings(context.Context, string, bool, *time.Time, int, time.Time, func(context.Context, string) error, func(context.Context, string, string) error) (*domain.Project, error)
}

type Scheduler interface {
	RecalculateActiveProjects(context.Context) error
	RecalculateProjectSchedule(context.Context, string) error
	MarkProjectUnscheduled(context.Context, string, string) error
}

type NoopScheduler struct{}

func (NoopScheduler) RecalculateActiveProjects(context.Context) error              { return nil }
func (NoopScheduler) RecalculateProjectSchedule(context.Context, string) error     { return nil }
func (NoopScheduler) MarkProjectUnscheduled(context.Context, string, string) error { return nil }
