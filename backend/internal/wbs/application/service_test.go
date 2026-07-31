package application

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	dependencydomain "github.com/banggok/sched_mind/backend/internal/dependencies/domain"
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

type previewStoreStub struct {
	Store
	preview func(context.Context, string, string, PreviewExecutableInput, time.Time, func(context.Context, string) error) (*SchedulePreview, error)
}

func (s previewStoreStub) PreviewExecutableSchedule(ctx context.Context, projectID, id string, input PreviewExecutableInput, now time.Time, schedule func(context.Context, string) error) (*SchedulePreview, error) {
	return s.preview(ctx, projectID, id, input, now, schedule)
}

func TestPreviewExecutableScheduleRejectsIncompleteDraftWithoutStoreCall(t *testing.T) {
	roleID, assigneeID, effortMinutes := "role", "member", 480
	cases := []struct {
		name  string
		input PreviewExecutableInput
	}{
		{name: "missing role", input: PreviewExecutableInput{AssigneeID: &assigneeID, EffortMinutes: &effortMinutes}},
		{name: "blank role", input: PreviewExecutableInput{RoleID: ptrString(" "), AssigneeID: &assigneeID, EffortMinutes: &effortMinutes}},
		{name: "missing effort", input: PreviewExecutableInput{RoleID: &roleID, AssigneeID: &assigneeID}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			storeCalls := 0
			store := previewStoreStub{preview: func(context.Context, string, string, PreviewExecutableInput, time.Time, func(context.Context, string) error) (*SchedulePreview, error) {
				storeCalls++
				return nil, errors.New("unexpected store call")
			}}
			service := NewServiceWithDependencies(store, &schedulerSpy{}, time.Now, func() (string, error) { return "unused", nil })

			value, err := service.PreviewExecutableSchedule(context.Background(), "project", "task", tc.input)
			if value != nil || !errors.Is(err, domain.ErrSchedulePreviewIncomplete) || storeCalls != 0 {
				t.Fatalf("value=%#v err=%v store calls=%d", value, err, storeCalls)
			}
		})
	}
}

func TestPreviewExecutableScheduleAllowsClearedAssigneeToReconcileDraft(t *testing.T) {
	now := time.Date(2026, 7, 31, 10, 16, 0, 0, time.UTC)
	roleID, effortMinutes := "role", 480
	reason := "Task requires an Assignee before it can be scheduled."
	scheduler := &schedulerSpy{}
	storeCalls := 0
	store := previewStoreStub{preview: func(ctx context.Context, projectID, id string, input PreviewExecutableInput, gotNow time.Time, schedule func(context.Context, string) error) (*SchedulePreview, error) {
		storeCalls++
		if projectID != "project" || id != "task" || gotNow != now {
			t.Fatalf("unexpected preview scope: %q %q %v", projectID, id, gotNow)
		}
		if input.RoleID == nil || *input.RoleID != roleID || input.AssigneeID != nil || input.EffortMinutes == nil || *input.EffortMinutes != effortMinutes {
			t.Fatalf("cleared-assignee preview input=%#v", input)
		}
		if err := schedule(ctx, projectID); err != nil {
			return nil, err
		}
		return &SchedulePreview{
			Task: &domain.Node{
				ID:        id,
				ProjectID: projectID,
				Name:      "Build API",
				Executable: domain.ExecutableFields{
					RoleID:                     &roleID,
					EffortMinutes:              &effortMinutes,
					ExecutionUnscheduledReason: &reason,
				},
				Children: []domain.Node{},
			},
			Dependencies: dependencydomain.Detail{BlockedBy: []dependencydomain.Item{}, Blocks: []dependencydomain.Item{}},
		}, nil
	}}
	service := NewServiceWithDependencies(store, scheduler, func() time.Time { return now }, func() (string, error) { return "unused", nil })

	value, err := service.PreviewExecutableSchedule(context.Background(), "project", "task", PreviewExecutableInput{
		RoleID:        &roleID,
		AssigneeID:    nil,
		EffortMinutes: &effortMinutes,
		LagDays:       0,
	})
	if err != nil {
		t.Fatal(err)
	}
	if storeCalls != 1 || scheduler.scheduleCalls != 1 {
		t.Fatalf("store calls=%d scheduler calls=%d", storeCalls, scheduler.scheduleCalls)
	}
	if value == nil || value.Task == nil || value.Task.Executable.AssigneeID != nil || value.Task.Executable.ExecutionTimeline.Start != nil || value.Task.Executable.ExecutionUnscheduledReason == nil || *value.Task.Executable.ExecutionUnscheduledReason != reason {
		t.Fatalf("cleared-assignee preview=%#v", value)
	}
	if len(value.Dependencies.BlockedBy) != 0 || len(value.Dependencies.Blocks) != 0 {
		t.Fatalf("cleared-assignee dependencies=%#v", value.Dependencies)
	}
}

func TestPreviewExecutableScheduleUsesDraftWithoutCallingConfirmedMutation(t *testing.T) {
	now := time.Date(2026, 7, 31, 6, 13, 0, 0, time.UTC)
	roleID, assigneeID, effortMinutes := "role", "member", 480
	scheduler := &schedulerSpy{}
	storeCalls := 0
	store := previewStoreStub{preview: func(ctx context.Context, projectID, id string, input PreviewExecutableInput, gotNow time.Time, schedule func(context.Context, string) error) (*SchedulePreview, error) {
		storeCalls++
		if projectID != "project" || id != "task" || gotNow != now {
			t.Fatalf("unexpected preview scope: %q %q %v", projectID, id, gotNow)
		}
		if input.RoleID == nil || *input.RoleID != roleID || input.AssigneeID == nil || *input.AssigneeID != assigneeID || input.EffortMinutes == nil || *input.EffortMinutes != effortMinutes || input.LagDays != 2 {
			t.Fatalf("preview input=%#v", input)
		}
		if err := schedule(ctx, projectID); err != nil {
			return nil, err
		}
		start := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
		return &SchedulePreview{
			Task: &domain.Node{
				ID:        id,
				ProjectID: projectID,
				Name:      "Build API",
				Executable: domain.ExecutableFields{
					RoleID:            &roleID,
					AssigneeID:        &assigneeID,
					EffortMinutes:     &effortMinutes,
					LagDays:           2,
					ExecutionTimeline: domain.Timeline{Start: &start, End: &end},
				},
				Children: []domain.Node{},
			},
			Dependencies: dependencydomain.Detail{
				BlockedBy: []dependencydomain.Item{{
					Dependency: dependencydomain.Dependency{
						ID:             "automatic-1",
						BlockingTaskID: "task-1",
						BlockedTaskID:  id,
						AutomaticOwned: true,
					},
					Task: dependencydomain.Task{ID: "task-1", Name: "Task 1"},
				}},
				Blocks: []dependencydomain.Item{},
			},
		}, nil
	}}
	service := NewServiceWithDependencies(store, scheduler, func() time.Time { return now }, func() (string, error) { return "unused", nil })

	value, err := service.PreviewExecutableSchedule(context.Background(), "project", "task", PreviewExecutableInput{
		RoleID:        &roleID,
		AssigneeID:    &assigneeID,
		EffortMinutes: &effortMinutes,
		LagDays:       2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if storeCalls != 1 || scheduler.scheduleCalls != 1 {
		t.Fatalf("store calls=%d scheduler calls=%d", storeCalls, scheduler.scheduleCalls)
	}
	if value == nil || value.Task == nil || value.Task.Executable.ExecutionTimeline.Start == nil || value.Task.Executable.ExecutionTimeline.End == nil {
		t.Fatalf("preview value=%#v", value)
	}
	if len(value.Dependencies.BlockedBy) != 1 || value.Dependencies.BlockedBy[0].Dependency.ID != "automatic-1" {
		t.Fatalf("preview dependencies=%#v", value.Dependencies)
	}
}

func ptrString(value string) *string { return &value }
