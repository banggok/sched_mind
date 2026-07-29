package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
)

type reopenStoreStub struct {
	Store
	reopen func(context.Context, string, string, time.Time, func(context.Context, string) error) (*domain.Node, error)
}

func (s reopenStoreStub) Reopen(ctx context.Context, projectID, id string, now time.Time, forecast func(context.Context, string) error) (*domain.Node, error) {
	return s.reopen(ctx, projectID, id, now, forecast)
}

type schedulerSpy struct {
	scheduleCalls   int
	forecastCalls   int
	invalidateCalls int
	forecastErr     error
}

func (s *schedulerSpy) RecalculateProjectSchedule(context.Context, string) error {
	s.scheduleCalls++
	return nil
}
func (s *schedulerSpy) RecalculateProjectForecast(context.Context, string) error {
	s.forecastCalls++
	return s.forecastErr
}
func (s *schedulerSpy) InvalidatePortfolio(context.Context, []string) error {
	s.invalidateCalls++
	return nil
}

func TestReopenUsesDedicatedStoreAndForecastExactlyOnce_AC1_AC7_AC8(t *testing.T) {
	now := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	scheduler := &schedulerSpy{}
	storeCalls := 0
	store := reopenStoreStub{reopen: func(ctx context.Context, projectID, id string, gotNow time.Time, forecast func(context.Context, string) error) (*domain.Node, error) {
		storeCalls++
		if projectID != "project" || id != "task" || gotNow != now {
			t.Fatalf("unexpected store args: %q %q %v", projectID, id, gotNow)
		}
		if err := forecast(ctx, projectID); err != nil {
			return nil, err
		}
		return &domain.Node{ID: id, ProjectID: projectID, Name: "Task", Executable: domain.ExecutableFields{}, Children: []domain.Node{}}, nil
	}}
	service := NewServiceWithDependencies(store, scheduler, func() time.Time { return now }, func() (string, error) { return "unused", nil })
	value, err := service.Reopen(context.Background(), "project", "task")
	if err != nil {
		t.Fatal(err)
	}
	if value == nil || value.ID != "task" || value.Executable.ActualEnd != nil {
		t.Fatalf("unexpected confirmed state: %#v", value)
	}
	if storeCalls != 1 || scheduler.forecastCalls != 1 || scheduler.scheduleCalls != 0 || scheduler.invalidateCalls != 0 {
		t.Fatalf("calls store=%d forecast=%d schedule=%d invalidate=%d", storeCalls, scheduler.forecastCalls, scheduler.scheduleCalls, scheduler.invalidateCalls)
	}
}

func TestReopenAllowsConfirmedOpenAndLockedStoreTransitions_AC7_AC8(t *testing.T) {
	for _, status := range []string{"open", "locked"} {
		t.Run(status, func(t *testing.T) {
			scheduler := &schedulerSpy{}
			store := reopenStoreStub{reopen: func(ctx context.Context, projectID, id string, _ time.Time, forecast func(context.Context, string) error) (*domain.Node, error) {
				if err := forecast(ctx, projectID); err != nil {
					return nil, err
				}
				return &domain.Node{ID: id, ProjectID: projectID, Name: status + " task", Executable: domain.ExecutableFields{}, Children: []domain.Node{}}, nil
			}}
			service := NewServiceWithDependencies(store, scheduler, time.Now, func() (string, error) { return "unused", nil })
			value, err := service.Reopen(context.Background(), "project", "task")
			if err != nil || value == nil || value.Name != status+" task" {
				t.Fatalf("value=%#v err=%v", value, err)
			}
			if scheduler.forecastCalls != 1 || scheduler.scheduleCalls != 0 {
				t.Fatalf("forecast=%d schedule=%d", scheduler.forecastCalls, scheduler.scheduleCalls)
			}
		})
	}
}

func TestReopenPreservesStoreAndForecastErrorsWithoutExtraCoordination_AC2_AC7_AC9_AC11_AC12_AC13(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{"unfinished", domain.ErrTaskNotCompleted},
		{"group", domain.ErrExecutableOnly},
		{"closed", domain.ErrProjectClosedReadOnly},
		{"conflict", domain.ErrTaskReopenConflict},
		{"not found", domain.ErrNotFound},
		{"persistence", errors.New("persistence unavailable")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			scheduler := &schedulerSpy{}
			store := reopenStoreStub{reopen: func(context.Context, string, string, time.Time, func(context.Context, string) error) (*domain.Node, error) {
				return nil, tc.err
			}}
			service := NewServiceWithDependencies(store, scheduler, time.Now, func() (string, error) { return "unused", nil })
			value, err := service.Reopen(context.Background(), "project", "task")
			if value != nil || !errors.Is(err, tc.err) {
				t.Fatalf("value=%#v err=%v", value, err)
			}
			if scheduler.forecastCalls != 0 || scheduler.scheduleCalls != 0 {
				t.Fatalf("unexpected coordination: forecast=%d schedule=%d", scheduler.forecastCalls, scheduler.scheduleCalls)
			}
		})
	}

	forecastFailure := errors.New("forecast failed")
	scheduler := &schedulerSpy{forecastErr: forecastFailure}
	store := reopenStoreStub{reopen: func(ctx context.Context, projectID, _ string, _ time.Time, forecast func(context.Context, string) error) (*domain.Node, error) {
		if err := forecast(ctx, projectID); err != nil {
			return nil, err
		}
		return &domain.Node{}, nil
	}}
	service := NewServiceWithDependencies(store, scheduler, time.Now, func() (string, error) { return "unused", nil })
	_, err := service.Reopen(context.Background(), "project", "task")
	if !errors.Is(err, forecastFailure) || scheduler.forecastCalls != 1 || scheduler.scheduleCalls != 0 {
		t.Fatalf("forecast failure err=%v calls=%d schedule=%d", err, scheduler.forecastCalls, scheduler.scheduleCalls)
	}
}

func TestReopenRejectsNilStoreResultAsContractViolation_AC11(t *testing.T) {
	store := reopenStoreStub{reopen: func(context.Context, string, string, time.Time, func(context.Context, string) error) (*domain.Node, error) {
		return nil, nil
	}}
	service := NewServiceWithDependencies(store, &schedulerSpy{}, time.Now, func() (string, error) { return "unused", nil })
	value, err := service.Reopen(context.Background(), "project", "task")
	if value != nil || err == nil || !strings.Contains(err.Error(), "store returned nil") {
		t.Fatalf("value=%#v err=%v", value, err)
	}
}
