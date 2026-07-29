package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
)

type WriteExecutableInput struct {
	Name                  *string
	RoleID, AssigneeID    *string
	EffortMinutes         *int
	Execution, Commitment domain.Timeline
}

type Store interface {
	Tree(context.Context, string) ([]domain.Node, error)
	Find(context.Context, string, string) (*domain.Node, error)
	Create(context.Context, string, string, *string, string, bool, time.Time, func(context.Context, string) error, func(context.Context, []string) error) (*domain.Node, error)
	Rename(context.Context, string, string, string, time.Time) (*domain.Node, error)
	UpdateExecutable(context.Context, string, string, WriteExecutableInput, time.Time, func(context.Context, string) error) (*domain.Node, error)
	Complete(context.Context, string, string, time.Time, time.Time, func(context.Context, string) error) (*domain.Node, error)
	Reopen(context.Context, string, string, time.Time, func(context.Context, string) error) (*domain.Node, error)
	Reorder(context.Context, string, string, domain.Direction, time.Time, func(context.Context, string) error) error
	Move(context.Context, string, string, string, *string, bool, time.Time, func(context.Context, string) error, func(context.Context, []string) error) error
	Delete(context.Context, string, string, time.Time, func(context.Context, string) error, func(context.Context, []string) error) error
}

type Scheduler interface {
	RecalculateProjectSchedule(context.Context, string) error
	RecalculateProjectForecast(context.Context, string) error
	InvalidatePortfolio(context.Context, []string) error
}
type NoopScheduler struct{}

func (NoopScheduler) RecalculateProjectSchedule(context.Context, string) error { return nil }
func (NoopScheduler) RecalculateProjectForecast(context.Context, string) error { return nil }
func (NoopScheduler) InvalidatePortfolio(context.Context, []string) error      { return nil }

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

func (s *Service) Tree(ctx context.Context, projectID string) ([]domain.Node, error) {
	value, err := s.store.Tree(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("get WBS tree: %w", err)
	}
	return value, nil
}
func (s *Service) Get(ctx context.Context, projectID, id string) (*domain.Node, error) {
	value, err := s.store.Find(ctx, projectID, id)
	if err != nil {
		return nil, fmt.Errorf("get WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("get WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) Create(ctx context.Context, projectID string, parentID *string, name string, confirm bool) (*domain.Node, error) {
	if _, err := domain.NormalizeName(name); err != nil {
		return nil, err
	}
	id, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("generate WBS ID: %w", err)
	}
	value, err := s.store.Create(ctx, id, projectID, parentID, name, confirm, s.now(), s.scheduler.RecalculateProjectSchedule, s.scheduler.InvalidatePortfolio)
	if err != nil {
		return nil, fmt.Errorf("create WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("create WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) Rename(ctx context.Context, p, id, name string) (*domain.Node, error) {
	value, err := s.store.Rename(ctx, p, id, name, s.now())
	if err != nil {
		return nil, fmt.Errorf("rename WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("rename WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) UpdateExecutable(ctx context.Context, p, id string, input WriteExecutableInput) (*domain.Node, error) {
	value, err := s.store.UpdateExecutable(ctx, p, id, input, s.now(), s.scheduler.RecalculateProjectSchedule)
	if err != nil {
		return nil, fmt.Errorf("update executable WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("update executable WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) Complete(ctx context.Context, p, id string, actual time.Time) (*domain.Node, error) {
	value, err := s.store.Complete(ctx, p, id, actual, s.now(), s.scheduler.RecalculateProjectForecast)
	if err != nil {
		return nil, fmt.Errorf("complete WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("complete WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) Reopen(ctx context.Context, p, id string) (*domain.Node, error) {
	value, err := s.store.Reopen(ctx, p, id, s.now(), s.scheduler.RecalculateProjectForecast)
	if err != nil {
		return nil, fmt.Errorf("reopen WBS: %w", err)
	}
	if value == nil {
		return nil, errors.New("reopen WBS: store returned nil")
	}
	return value, nil
}
func (s *Service) Reorder(ctx context.Context, p, id string, d domain.Direction) error {
	if d != domain.MoveUp && d != domain.MoveDown {
		return domain.ErrMoveNotAllowed
	}
	if err := s.store.Reorder(ctx, p, id, d, s.now(), s.scheduler.RecalculateProjectSchedule); err != nil {
		return fmt.Errorf("reorder WBS: %w", err)
	}
	return nil
}
func (s *Service) Move(ctx context.Context, p, id string, parent *string, confirm bool) error {
	conversionID, err := s.newID()
	if err != nil {
		return fmt.Errorf("generate conversion WBS ID: %w", err)
	}
	if err := s.store.Move(ctx, p, id, conversionID, parent, confirm, s.now(), s.scheduler.RecalculateProjectSchedule, s.scheduler.InvalidatePortfolio); err != nil {
		return fmt.Errorf("move WBS: %w", err)
	}
	return nil
}
func (s *Service) Delete(ctx context.Context, p, id string) error {
	if err := s.store.Delete(ctx, p, id, s.now(), s.scheduler.RecalculateProjectSchedule, s.scheduler.InvalidatePortfolio); err != nil {
		return fmt.Errorf("delete WBS: %w", err)
	}
	return nil
}
