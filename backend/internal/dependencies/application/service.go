package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/banggok/sched_mind/backend/internal/dependencies/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
)

type Store interface {
	List(context.Context, string) (*domain.Detail, error)
	Candidates(context.Context, string, domain.Direction, string, int, int) (*domain.CandidatePage, error)
	Create(context.Context, domain.Dependency, func(context.Context, []string) error) (*domain.Dependency, error)
	Delete(context.Context, string, time.Time, func(context.Context, []string) error) error
}

type Scheduler interface {
	InvalidatePortfolio(context.Context, []string) error
}
type NoopScheduler struct{}

func (NoopScheduler) InvalidatePortfolio(context.Context, []string) error { return nil }

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
	if scheduler == nil {
		scheduler = NoopScheduler{}
	}
	return &Service{store: store, scheduler: scheduler, now: now, newID: newID}
}

func (s *Service) List(ctx context.Context, taskID string) (*domain.Detail, error) {
	v, err := s.store.List(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("list dependencies: %w", err)
	}
	if v == nil {
		return nil, errors.New("list dependencies: store returned nil")
	}
	return v, nil
}

func (s *Service) Candidates(ctx context.Context, taskID string, direction domain.Direction, search string, page, pageSize int) (*domain.CandidatePage, error) {
	if !direction.Valid() {
		return nil, domain.ErrInvalidDirection
	}
	if page < 1 || pageSize < 1 || pageSize > 100 {
		return nil, errors.New("invalid pagination")
	}
	v, err := s.store.Candidates(ctx, taskID, direction, search, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list dependency candidates: %w", err)
	}
	if v == nil {
		return nil, errors.New("list dependency candidates: store returned nil")
	}
	return v, nil
}

func (s *Service) Create(ctx context.Context, blockingTaskID, blockedTaskID string) (*domain.Dependency, error) {
	ctx = schedulingimpact.WithOperation(ctx, "", schedulingimpact.ModeOrdinary)
	id, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("generate dependency ID: %w", err)
	}
	d, err := domain.New(id, blockingTaskID, blockedTaskID, s.now())
	if err != nil {
		return nil, err
	}
	v, err := s.store.Create(ctx, *d, s.scheduler.InvalidatePortfolio)
	if err != nil {
		return nil, fmt.Errorf("create dependency: %w", err)
	}
	if v == nil {
		return nil, errors.New("create dependency: store returned nil")
	}
	return v, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	ctx = schedulingimpact.WithOperation(ctx, "", schedulingimpact.ModeOrdinary)
	if err := s.store.Delete(ctx, id, s.now(), s.scheduler.InvalidatePortfolio); err != nil {
		return fmt.Errorf("delete dependency: %w", err)
	}
	return nil
}
