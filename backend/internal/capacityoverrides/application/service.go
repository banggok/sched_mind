package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
)

type WriteInput struct {
	Description        string
	StartDate, EndDate time.Time
	Capacity           *float64
}
type Scheduler interface {
	RecalculateMemberSchedule(context.Context, string) error
}

type NoopScheduler struct{}

func (NoopScheduler) RecalculateMemberSchedule(context.Context, string) error { return nil }

type ConditionalScheduleRepository interface {
	CreateWithSchedule(context.Context, domain.CapacityOverride, func(context.Context, string) error) error
	UpdateWithSchedule(context.Context, domain.CapacityOverride, func(context.Context, string) error) error
	DeleteWithSchedule(context.Context, string, string, func(context.Context, string) error) error
}

type Service struct {
	repository Repository
	scheduler  Scheduler
	now        func() time.Time
	newID      func() (string, error)
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, scheduler: NoopScheduler{}, now: time.Now, newID: identity.NewUUID}
}
func NewServiceWithScheduler(repository Repository, scheduler Scheduler) *Service {
	if scheduler == nil {
		scheduler = NoopScheduler{}
	}
	return &Service{repository: repository, scheduler: scheduler, now: time.Now, newID: identity.NewUUID}
}
func NewServiceWithDependencies(repository Repository, now func() time.Time, newID func() (string, error)) *Service {
	return &Service{repository: repository, scheduler: NoopScheduler{}, now: now, newID: newID}
}

func (s *Service) List(ctx context.Context, memberID string, query ListQuery) (listing.Page[domain.CapacityOverride], error) {
	result, err := s.repository.List(ctx, memberID, query)
	if err != nil {
		return listing.Page[domain.CapacityOverride]{}, fmt.Errorf("list capacity overrides: %w", err)
	}
	return result, nil
}
func (s *Service) Get(ctx context.Context, memberID, id string) (*domain.CapacityOverride, error) {
	result, err := s.repository.Find(ctx, memberID, id)
	if err != nil {
		return nil, fmt.Errorf("get capacity override: %w", err)
	}
	if result == nil {
		return nil, errors.New("get capacity override: repository returned nil without error")
	}
	return result, nil
}
func (s *Service) Create(ctx context.Context, memberID string, input WriteInput) (*domain.CapacityOverride, error) {
	ctx = schedulingimpact.WithOperation(ctx, "", schedulingimpact.ModeOrdinary)
	capacity, err := capacityFrom(input.Capacity)
	if err != nil {
		return nil, err
	}
	id, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("create capacity override ID: %w", err)
	}
	override, err := domain.New(id, memberID, input.Description, input.StartDate, input.EndDate, capacity, s.now())
	if err != nil {
		return nil, err
	}
	if override == nil {
		return nil, errors.New("create capacity override: domain returned nil without error")
	}
	if repository, ok := s.repository.(ConditionalScheduleRepository); ok {
		if err := repository.CreateWithSchedule(ctx, *override, s.scheduler.RecalculateMemberSchedule); err != nil {
			return nil, fmt.Errorf("create capacity override: %w", err)
		}
	} else if err := s.repository.Create(ctx, *override); err != nil {
		return nil, fmt.Errorf("create capacity override: %w", err)
	}
	return s.Get(ctx, memberID, id)
}
func (s *Service) Update(ctx context.Context, memberID, id string, input WriteInput) (*domain.CapacityOverride, error) {
	ctx = schedulingimpact.WithOperation(ctx, "", schedulingimpact.ModeOrdinary)
	capacity, err := capacityFrom(input.Capacity)
	if err != nil {
		return nil, err
	}
	override, err := s.repository.Find(ctx, memberID, id)
	if err != nil {
		return nil, fmt.Errorf("find capacity override for update: %w", err)
	}
	if override == nil {
		return nil, errors.New("update capacity override: repository returned nil without error")
	}
	if err := override.Update(input.Description, input.StartDate, input.EndDate, capacity, s.now()); err != nil {
		return nil, err
	}
	if repository, ok := s.repository.(ConditionalScheduleRepository); ok {
		if err := repository.UpdateWithSchedule(ctx, *override, s.scheduler.RecalculateMemberSchedule); err != nil {
			return nil, fmt.Errorf("update capacity override: %w", err)
		}
	} else if err := s.repository.Update(ctx, *override); err != nil {
		return nil, fmt.Errorf("update capacity override: %w", err)
	}
	return s.Get(ctx, memberID, id)
}
func (s *Service) Delete(ctx context.Context, memberID, id string) error {
	ctx = schedulingimpact.WithOperation(ctx, "", schedulingimpact.ModeOrdinary)
	if repository, ok := s.repository.(ConditionalScheduleRepository); ok {
		if err := repository.DeleteWithSchedule(ctx, memberID, id, s.scheduler.RecalculateMemberSchedule); err != nil {
			return fmt.Errorf("delete capacity override: %w", err)
		}
		return nil
	}
	if err := s.repository.Delete(ctx, memberID, id); err != nil {
		return fmt.Errorf("delete capacity override: %w", err)
	}
	return nil
}
func capacityFrom(value *float64) (domain.Capacity, error) {
	if value == nil {
		return domain.Capacity{}, domain.ErrCapacityRequired
	}
	return domain.NewCapacity(*value)
}
