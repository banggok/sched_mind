package application

import (
	"context"
	"errors"
	"github.com/banggok/sched_mind/backend/internal/publicholidays/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"testing"
	"time"
)

type fakeRepository struct {
	values map[string]domain.PublicHoliday
	query  ListQuery
	err    error
}

func (f *fakeRepository) List(_ context.Context, q ListQuery) (listing.Page[domain.PublicHoliday], error) {
	f.query = q
	return listing.Page[domain.PublicHoliday]{Page: q.Page, PageSize: q.PageSize}, f.err
}
func (f *fakeRepository) Find(_ context.Context, id string) (*domain.PublicHoliday, error) {
	if f.err != nil {
		return nil, f.err
	}
	value, ok := f.values[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &value, nil
}
func (f *fakeRepository) Create(_ context.Context, v domain.PublicHoliday) error {
	if f.err != nil {
		return f.err
	}
	f.values[v.ID] = v
	return nil
}
func (f *fakeRepository) Update(_ context.Context, v domain.PublicHoliday) error {
	if f.err != nil {
		return f.err
	}
	f.values[v.ID] = v
	return nil
}
func (f *fakeRepository) Delete(_ context.Context, id string) error {
	if _, ok := f.values[id]; !ok {
		return domain.ErrNotFound
	}
	delete(f.values, id)
	return nil
}
func (f *fakeRepository) ExistsOn(_ context.Context, date time.Time) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for _, v := range f.values {
		for _, holidayDate := range v.Dates {
			if holidayDate.Equal(date) {
				return true, nil
			}
		}
	}
	return false, nil
}
func (f *fakeRepository) DatesBetween(_ context.Context, start, end time.Time) ([]time.Time, error) {
	if f.err != nil {
		return nil, f.err
	}
	var dates []time.Time
	for _, value := range f.values {
		for _, date := range value.Dates {
			if !date.Before(start) && !date.After(end) {
				dates = append(dates, date)
			}
		}
	}
	return dates, nil
}
func TestServiceCRUDListAndResolutionConsumer(t *testing.T) {
	repo := &fakeRepository{values: map[string]domain.PublicHoliday{}}
	now := time.Date(2026, 8, 17, 0, 0, 0, 0, time.UTC)
	service := NewServiceWithDependencies(repo, func() time.Time { return now }, func() (string, error) { return "id", nil })
	value, err := service.Create(context.Background(), WriteInput{StartDate: now, EndDate: now, Description: " Holiday "})
	if err != nil || value == nil || value.Description != "Holiday" {
		t.Fatalf("value=%+v err=%v", value, err)
	}
	date := now
	_, err = service.List(context.Background(), ListQuery{Query: listing.Query{Page: 2, PageSize: 5}, HolidayDate: &date})
	if err != nil || repo.query.HolidayDate == nil {
		t.Fatal(err)
	}
	isHoliday, err := service.IsHoliday(context.Background(), now)
	if err != nil || !isHoliday {
		t.Fatalf("holiday=%v err=%v", isHoliday, err)
	}
	value, err = service.Update(context.Background(), "id", WriteInput{StartDate: now, EndDate: now, Description: "Updated"})
	if err != nil || value.Description != "Updated" {
		t.Fatal(err)
	}
	if err := service.Delete(context.Background(), "id"); err != nil {
		t.Fatal(err)
	}
}
func TestServicePreservesDependencyFailure(t *testing.T) {
	sentinel := errors.New("database unavailable")
	service := NewService(&fakeRepository{err: sentinel})
	_, err := service.List(context.Background(), ListQuery{})
	if !errors.Is(err, sentinel) {
		t.Fatalf("cause=%v", err)
	}
}

type scheduleRepository struct {
	fakeRepository
	scheduleCalls int
}

func (repository *scheduleRepository) CreateWithSchedule(ctx context.Context, value domain.PublicHoliday, schedule func(context.Context) error) error {
	if err := repository.Create(ctx, value); err != nil {
		return err
	}
	repository.scheduleCalls++
	return schedule(ctx)
}
func (repository *scheduleRepository) UpdateWithSchedule(ctx context.Context, value domain.PublicHoliday, dateRangeChanged bool, schedule func(context.Context) error) error {
	if err := repository.Update(ctx, value); err != nil {
		return err
	}
	if !dateRangeChanged {
		return nil
	}
	repository.scheduleCalls++
	return schedule(ctx)
}
func (repository *scheduleRepository) DeleteWithSchedule(ctx context.Context, id string, schedule func(context.Context) error) error {
	if err := repository.Delete(ctx, id); err != nil {
		return err
	}
	repository.scheduleCalls++
	return schedule(ctx)
}

type holidayScheduler struct{ calls int }

func (scheduler *holidayScheduler) RecalculateActiveProjects(context.Context) error {
	scheduler.calls++
	return nil
}

func TestScheduleAwareWritesSkipDescriptionOnlyUpdate(t *testing.T) {
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	repository := &scheduleRepository{fakeRepository: fakeRepository{values: map[string]domain.PublicHoliday{}}}
	scheduler := &holidayScheduler{}
	service := NewServiceWithScheduler(repository, scheduler, func() time.Time { return now })
	service.newID = func() (string, error) { return "holiday", nil }

	if _, err := service.Create(context.Background(), WriteInput{StartDate: now, EndDate: now, Description: "Holiday"}); err != nil {
		t.Fatal(err)
	}
	if scheduler.calls != 1 || repository.scheduleCalls != 1 {
		t.Fatalf("create calls: scheduler=%d repository=%d", scheduler.calls, repository.scheduleCalls)
	}

	if _, err := service.Update(context.Background(), "holiday", WriteInput{StartDate: now, EndDate: now, Description: "Renamed"}); err != nil {
		t.Fatal(err)
	}
	if scheduler.calls != 1 || repository.scheduleCalls != 1 {
		t.Fatalf("description-only update scheduled: scheduler=%d repository=%d", scheduler.calls, repository.scheduleCalls)
	}

	next := now.AddDate(0, 0, 1)
	if _, err := service.Update(context.Background(), "holiday", WriteInput{StartDate: next, EndDate: next, Description: "Moved"}); err != nil {
		t.Fatal(err)
	}
	if scheduler.calls != 2 || repository.scheduleCalls != 2 {
		t.Fatalf("date update calls: scheduler=%d repository=%d", scheduler.calls, repository.scheduleCalls)
	}

	if err := service.Delete(context.Background(), "holiday"); err != nil {
		t.Fatal(err)
	}
	if scheduler.calls != 3 || repository.scheduleCalls != 3 {
		t.Fatalf("delete calls: scheduler=%d repository=%d", scheduler.calls, repository.scheduleCalls)
	}
}
