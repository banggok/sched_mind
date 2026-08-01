package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/banggok/sched_mind/backend/internal/projects/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
)

type Service struct {
	store     Store
	scheduler Scheduler
	now       func() time.Time
	newID     func() (string, error)
}

func NewService(store Store, scheduler Scheduler) *Service {
	if scheduler == nil {
		scheduler = NoopScheduler{}
	}
	return &Service{store: store, scheduler: scheduler, now: func() time.Time { return time.Now().UTC() }, newID: identity.NewUUID}
}

func NewServiceWithDependencies(store Store, scheduler Scheduler, now func() time.Time, newID func() (string, error)) *Service {
	return &Service{store: store, scheduler: scheduler, now: now, newID: newID}
}

func (s *Service) List(ctx context.Context, query listing.Query) (listing.Page[domain.Project], error) {
	result, err := s.store.List(ctx, query)
	if err != nil {
		return listing.Page[domain.Project]{}, fmt.Errorf("list projects: %w", err)
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Project, error) {
	value, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if value == nil {
		return nil, errors.New("get project: store returned nil without error")
	}
	return value, nil
}

func (s *Service) Create(ctx context.Context, name string, automaticScheduling bool, schedulingStartDate *time.Time, projectBuffer int) (*domain.Project, error) {
	if _, err := domain.NormalizeName(name); err != nil {
		return nil, err
	}
	if err := domain.ValidateProjectBuffer(projectBuffer); err != nil {
		return nil, err
	}
	id, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("generate project ID: %w", err)
	}
	value, err := s.store.CreateNext(ctx, id, name, automaticScheduling, schedulingStartDate, projectBuffer, s.now())
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	if value == nil {
		return nil, errors.New("create project: store returned nil without error")
	}
	return value, nil
}

func (s *Service) Rename(ctx context.Context, id, name string) (*domain.Project, error) {
	if _, err := domain.NormalizeName(name); err != nil {
		return nil, err
	}
	value, err := s.store.Rename(ctx, id, name, s.now())
	if err != nil {
		return nil, fmt.Errorf("rename project: %w", err)
	}
	if value == nil {
		return nil, errors.New("rename project: store returned nil without error")
	}
	return value, nil
}

func (s *Service) Update(ctx context.Context, id, name string, automaticScheduling bool, schedulingStartDate *time.Time, projectBuffer int) (*domain.Project, error) {
	ctx = schedulingimpact.WithOperation(ctx, id, schedulingimpact.ModeOrdinary)
	if _, err := domain.NormalizeName(name); err != nil {
		return nil, err
	}
	if err := domain.ValidateProjectBuffer(projectBuffer); err != nil {
		return nil, err
	}
	value, err := s.store.UpdateDetails(ctx, id, name, automaticScheduling, schedulingStartDate, projectBuffer, s.now(), s.scheduler.RecalculateProjectSchedule, s.scheduler.MarkProjectUnscheduled)
	if err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	if value == nil {
		return nil, errors.New("update project: store returned nil without error")
	}
	return value, nil
}

func (s *Service) ChangeStatus(ctx context.Context, id string, target domain.Status) (*domain.Project, error) {
	ctx = schedulingimpact.WithOperation(ctx, id, schedulingimpact.ModeOrdinary)
	value, err := s.store.ChangeStatus(ctx, id, target, s.now(), s.scheduler.RecalculateActiveProjects)
	if err != nil {
		return nil, fmt.Errorf("change project status: %w", err)
	}
	if value == nil {
		return nil, errors.New("change project status: store returned nil without error")
	}
	return value, nil
}

func (s *Service) BulkReopen(ctx context.Context, rootProjectID, token string) ([]domain.Project, error) {
	ctx = schedulingimpact.WithOperation(ctx, rootProjectID, schedulingimpact.ModeBulkReopen)
	values, err := s.store.BulkReopen(ctx, rootProjectID, token, s.now(), s.scheduler.RecalculateActiveProjects)
	if err != nil {
		return nil, fmt.Errorf("bulk reopen projects: %w", err)
	}
	return values, nil
}

func (s *Service) MovePriority(ctx context.Context, id string, direction domain.PriorityDirection) (*domain.Project, error) {
	ctx = schedulingimpact.WithOperation(ctx, id, schedulingimpact.ModeOrdinary)
	if direction != domain.PriorityUp && direction != domain.PriorityDown {
		return nil, domain.ErrPriorityDirectionInvalid
	}
	value, err := s.store.MovePriority(ctx, id, direction, s.now(), s.scheduler.RecalculateActiveProjects)
	if err != nil {
		return nil, fmt.Errorf("move project priority: %w", err)
	}
	if value == nil {
		return nil, errors.New("move project priority: store returned nil without error")
	}
	return value, nil
}

func (s *Service) UpdateSettings(ctx context.Context, id string, automaticScheduling bool, schedulingStartDate *time.Time, projectBuffer int) (*domain.Project, error) {
	ctx = schedulingimpact.WithOperation(ctx, id, schedulingimpact.ModeOrdinary)
	if err := domain.ValidateProjectBuffer(projectBuffer); err != nil {
		return nil, err
	}
	value, err := s.store.UpdateSettings(ctx, id, automaticScheduling, schedulingStartDate, projectBuffer, s.now(), s.scheduler.RecalculateProjectSchedule, s.scheduler.MarkProjectUnscheduled)
	if err != nil {
		return nil, fmt.Errorf("update project settings: %w", err)
	}
	if value == nil {
		return nil, errors.New("update project settings: store returned nil without error")
	}
	return value, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.store.DeleteChildless(ctx, id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}
