package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/banggok/sched_mind/backend/internal/publicholidays/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
)

type WriteInput struct {
	StartDate   time.Time
	EndDate     time.Time
	Description string
}

type Scheduler interface {
	RecalculateActiveProjects(context.Context) error
}

type scheduleAwareRepository interface {
	CreateWithSchedule(context.Context, domain.PublicHoliday, func(context.Context) error) error
	UpdateWithSchedule(context.Context, domain.PublicHoliday, bool, func(context.Context) error) error
	DeleteWithSchedule(context.Context, string, func(context.Context) error) error
}

type Service struct {
	repository Repository
	scheduler  Scheduler
	now        func() time.Time
	newID      func() (string, error)
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now, newID: identity.NewUUID}
}

func NewServiceWithScheduler(repository Repository, scheduler Scheduler, now func() time.Time) *Service {
	service := NewServiceWithClock(repository, now)
	service.scheduler = scheduler
	return service
}

func NewServiceWithClock(repository Repository, now func() time.Time) *Service {
	return &Service{repository: repository, now: now, newID: identity.NewUUID}
}

func NewServiceWithDependencies(repository Repository, now func() time.Time, newID func() (string, error)) *Service {
	return &Service{repository: repository, now: now, newID: newID}
}

func (s *Service) List(ctx context.Context, query ListQuery) (listing.Page[domain.PublicHoliday], error) {
	if query.Today.IsZero() {
		query.Today = dateOnly(s.now())
	}
	result, err := s.repository.List(ctx, query)
	if err != nil {
		return listing.Page[domain.PublicHoliday]{}, fmt.Errorf("list public holidays: %w", err)
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, id string) (*domain.PublicHoliday, error) {
	result, err := s.repository.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get public holiday: %w", err)
	}
	if result == nil {
		return nil, errors.New("get public holiday: repository returned nil without error")
	}
	return result, nil
}

func (s *Service) Create(ctx context.Context, input WriteInput) (*domain.PublicHoliday, error) {
	id, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("create public holiday ID: %w", err)
	}
	holiday, err := domain.New(id, input.StartDate, input.EndDate, input.Description, s.now())
	if err != nil {
		return nil, err
	}
	if holiday == nil {
		return nil, errors.New("create public holiday: domain returned nil without error")
	}
	ctx = schedulingimpact.WithOperation(ctx, "", schedulingimpact.ModeOrdinary)
	if repository, ok := s.repository.(scheduleAwareRepository); ok && s.scheduler != nil {
		err = repository.CreateWithSchedule(ctx, *holiday, s.scheduler.RecalculateActiveProjects)
	} else {
		err = s.repository.Create(ctx, *holiday)
	}
	if err != nil {
		return nil, fmt.Errorf("create public holiday: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, input WriteInput) (*domain.PublicHoliday, error) {
	holiday, err := s.repository.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find public holiday for update: %w", err)
	}
	if holiday == nil {
		return nil, errors.New("update public holiday: repository returned nil without error")
	}
	dateRangeChanged := !sameDate(holiday.StartDate, input.StartDate) || !sameDate(holiday.EndDate, input.EndDate)
	if err := holiday.Update(input.StartDate, input.EndDate, input.Description, s.now()); err != nil {
		return nil, err
	}
	ctx = schedulingimpact.WithOperation(ctx, "", schedulingimpact.ModeOrdinary)
	if repository, ok := s.repository.(scheduleAwareRepository); ok && s.scheduler != nil {
		err = repository.UpdateWithSchedule(ctx, *holiday, dateRangeChanged, s.scheduler.RecalculateActiveProjects)
	} else {
		err = s.repository.Update(ctx, *holiday)
	}
	if err != nil {
		return nil, fmt.Errorf("update public holiday: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	ctx = schedulingimpact.WithOperation(ctx, "", schedulingimpact.ModeOrdinary)
	var err error
	if repository, ok := s.repository.(scheduleAwareRepository); ok && s.scheduler != nil {
		err = repository.DeleteWithSchedule(ctx, id, s.scheduler.RecalculateActiveProjects)
	} else {
		err = s.repository.Delete(ctx, id)
	}
	if err != nil {
		return fmt.Errorf("delete public holiday: %w", err)
	}
	return nil
}

func (s *Service) IsHoliday(ctx context.Context, date time.Time) (bool, error) {
	if domain.IsWeekend(date) {
		return true, nil
	}
	result, err := s.repository.ExistsOn(ctx, date)
	if err != nil {
		return false, fmt.Errorf("resolve public holiday date: %w", err)
	}
	return result, nil
}

func (s *Service) CalendarDates(ctx context.Context, startDate, endDate time.Time) ([]time.Time, error) {
	result, err := s.repository.DatesBetween(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("list public holiday calendar dates: %w", err)
	}
	return result, nil
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, value.Location())
}

func sameDate(left, right time.Time) bool {
	return dateOnly(left).Equal(dateOnly(right))
}
