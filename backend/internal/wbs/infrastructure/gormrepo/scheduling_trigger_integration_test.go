package gormrepo

import (
	"context"
	"errors"
	"testing"
	"time"

	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"github.com/banggok/sched_mind/backend/internal/wbs/application"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
)

type schedulePreviewProjectRecord struct {
	ID              string `gorm:"primaryKey"`
	StartDate       *time.Time
	EndDate         *time.Time
	ScheduleVersion int64
}

func (schedulePreviewProjectRecord) TableName() string { return "projects" }

type schedulePreviewDependencyRecord struct {
	ID                            string `gorm:"primaryKey"`
	BlockingTaskID, BlockedTaskID string
	CreatedAt, UpdatedAt          time.Time
}

func (schedulePreviewDependencyRecord) TableName() string { return "task_dependencies" }

type schedulePreviewAllocationRecord struct {
	TaskID, AssigneeID       string
	Timeline                 string
	AllocationDate           time.Time
	AllocatedMinutes         string
	RemainingCapacityMinutes string
	Sequence                 int
}

func (schedulePreviewAllocationRecord) TableName() string {
	return "task_schedule_allocations"
}

func TestUpdateExecutableAndSchedulerShareTransactionRollback_AC27_AC28_AC35(t *testing.T) {
	repository, database := reopenTestDB(t)
	if err := database.AutoMigrate(&memberModel{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	roleID, assigneeID := "role", "member"
	if err := database.Create(&memberModel{ID: assigneeID, RoleID: roleID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&reopenProjectRecord{
		ID: "project", Name: "Alpha", NameKey: "alpha", Status: "open", AutomaticScheduling: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&nodeModel{
		ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task",
		Position: 1, RoleID: &roleID, AssigneeID: &assigneeID, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	schedulerFailure := errors.New("scheduler persistence failed")
	percentage := 20
	_, err := repository.UpdateExecutable(
		context.Background(),
		"project",
		"task",
		application.WriteExecutableInput{RoleID: &roleID, AssigneeID: &assigneeID, LagDays: 2, CapacityAllocationPercentage: &percentage},
		now.Add(time.Hour),
		func(ctx context.Context, projectID string) error {
			if !sharedpersistence.ScheduleMutationSerialized(ctx) {
				t.Fatal("scheduler callback did not inherit schedule-mutation serialization")
			}
			if projectID != "project" {
				t.Fatalf("scheduler project=%q", projectID)
			}
			transaction := sharedpersistence.Transaction(ctx, database)
			var pending nodeModel
			if err := transaction.First(&pending, "id = ?", "task").Error; err != nil {
				t.Fatal(err)
			}
			if pending.LagDays != 2 || pending.CapacityAllocationPercentage != 20 {
				t.Fatalf("scheduler did not observe pending Lag: %#v", pending)
			}
			generatedStart := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)
			if err := transaction.Model(&nodeModel{}).Where("id = ?", "task").Update("execution_start", generatedStart).Error; err != nil {
				t.Fatal(err)
			}
			return schedulerFailure
		},
	)
	if !errors.Is(err, schedulerFailure) {
		t.Fatalf("UpdateExecutable() error=%v, want scheduler failure", err)
	}

	var stored nodeModel
	if err := database.First(&stored, "id = ?", "task").Error; err != nil {
		t.Fatal(err)
	}
	if stored.LagDays != 0 || stored.CapacityAllocationPercentage != 100 || stored.ExecutionStart != nil || stored.UpdatedAt != now {
		t.Fatalf("state after rollback=%#v, want original Task", stored)
	}
}

func TestUpdateExecutableAutomaticOffPreservesManualTimelineAndRebuildsFixedAllocation_US63_AC18_AC19(t *testing.T) {
	repository, database := reopenTestDB(t)
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	if err := database.Create(&reopenProjectRecord{
		ID: "project", Name: "Alpha", NameKey: "alpha", Status: "open", AutomaticScheduling: false,
	}).Error; err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	effort := 480
	if err := database.Create(&nodeModel{
		ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task",
		Position: 1, EffortMinutes: &effort, ExecutionStart: &start, ExecutionEnd: &end,
		CommitmentStart: &start, CommitmentEnd: &end, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	calls := 0
	updated, err := repository.UpdateExecutable(
		context.Background(),
		"project",
		"task",
		application.WriteExecutableInput{
			EffortMinutes: &effort,
			LagDays:       1,
			Execution:     domain.Timeline{Start: &start, End: &end},
			Commitment:    domain.Timeline{Start: &start, End: &end},
		},
		now.Add(time.Hour),
		func(context.Context, string) error {
			calls++
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("scheduler calls=%d, want fixed-allocation portfolio recalculation", calls)
	}
	if updated == nil || updated.Executable.LagDays != 1 {
		t.Fatalf("updated Task=%#v", updated)
	}
	var stored nodeModel
	if err := database.First(&stored, "id = ?", "task").Error; err != nil {
		t.Fatal(err)
	}
	if stored.ExecutionStart == nil || !stored.ExecutionStart.Equal(start) || stored.ExecutionEnd == nil || !stored.ExecutionEnd.Equal(end) {
		t.Fatalf("manual Execution timeline changed: %#v", stored)
	}
	if stored.CommitmentStart == nil || !stored.CommitmentStart.Equal(start) || stored.CommitmentEnd == nil || !stored.CommitmentEnd.Equal(end) {
		t.Fatalf("manual Commitment timeline changed: %#v", stored)
	}
}

func TestUpdateExecutablePreservesPercentageOnFirstSelectionChangeAndClear_US65_AC13_AC14_AC15(t *testing.T) {
	repository, database := reopenTestDB(t)
	if err := database.AutoMigrate(&memberModel{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	if err := database.Create(&reopenProjectRecord{ID: "project", Name: "Alpha", NameKey: "alpha", Status: "open", AutomaticScheduling: true}).Error; err != nil {
		t.Fatal(err)
	}
	roleID, firstID, secondID := "role", "first", "second"
	for _, member := range []memberModel{{ID: firstID, RoleID: roleID}, {ID: secondID, RoleID: roleID}} {
		if err := database.Create(&member).Error; err != nil {
			t.Fatal(err)
		}
	}
	effort, percentage := 480, 20
	if err := database.Create(&nodeModel{ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", Position: 1, RoleID: &roleID, EffortMinutes: &effort, CapacityAllocationPercentage: percentage, CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	noopSchedule := func(context.Context, string) error { return nil }
	value, err := repository.UpdateExecutable(context.Background(), "project", "task", application.WriteExecutableInput{RoleID: &roleID, AssigneeID: &firstID, EffortMinutes: &effort}, now.Add(time.Hour), noopSchedule)
	if err != nil {
		t.Fatal(err)
	}
	if value.Executable.CapacityAllocationPercentage != 20 {
		t.Fatalf("first-assignment percentage=%d", value.Executable.CapacityAllocationPercentage)
	}
	value, err = repository.UpdateExecutable(context.Background(), "project", "task", application.WriteExecutableInput{RoleID: &roleID, AssigneeID: &secondID, EffortMinutes: &effort}, now.Add(2*time.Hour), noopSchedule)
	if err != nil {
		t.Fatal(err)
	}
	if value.Executable.CapacityAllocationPercentage != 20 {
		t.Fatalf("changed-assignee percentage=%d", value.Executable.CapacityAllocationPercentage)
	}
	value, err = repository.UpdateExecutable(context.Background(), "project", "task", application.WriteExecutableInput{RoleID: &roleID, AssigneeID: nil, EffortMinutes: &effort}, now.Add(3*time.Hour), noopSchedule)
	if err != nil {
		t.Fatal(err)
	}
	if value.Executable.CapacityAllocationPercentage != 20 {
		t.Fatalf("cleared-assignee percentage=%d", value.Executable.CapacityAllocationPercentage)
	}
	explicit := 60
	value, err = repository.UpdateExecutable(context.Background(), "project", "task", application.WriteExecutableInput{RoleID: &roleID, AssigneeID: &firstID, EffortMinutes: &effort, CapacityAllocationPercentage: &explicit}, now.Add(4*time.Hour), noopSchedule)
	if err != nil {
		t.Fatal(err)
	}
	if value.Executable.CapacityAllocationPercentage != 60 {
		t.Fatalf("explicit percentage=%d", value.Executable.CapacityAllocationPercentage)
	}
}

func TestPreviewExecutableScheduleUsesDraftAndRollsBackAllPersistence(t *testing.T) {
	repository, database := reopenTestDB(t)
	if err := database.AutoMigrate(
		&memberModel{},
		&schedulePreviewProjectRecord{},
		&schedulePreviewDependencyRecord{},
		&schedulePreviewAllocationRecord{},
	); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 31, 6, 13, 0, 0, time.UTC)
	if err := database.Create(&reopenProjectRecord{
		ID: "project", Name: "Alpha", NameKey: "alpha", Status: "open", AutomaticScheduling: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	roleID, assigneeID, originalAssigneeID := "role", "member", "old-member"
	originalEffort, previewEffort := 240, 480
	originalStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	originalEnd := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	if err := database.Create(&memberModel{ID: assigneeID, RoleID: roleID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&[]nodeModel{
		{
			ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task",
			Position: 1, RoleID: &roleID, AssigneeID: &originalAssigneeID, EffortMinutes: &originalEffort,
			LagDays: 0, ExecutionStart: &originalStart, ExecutionEnd: &originalEnd,
			CommitmentStart: &originalStart, CommitmentEnd: &originalEnd,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "other", ProjectID: "project", ParentKey: "", Name: "Other", NameKey: "other",
			Position: 2, CreatedAt: now, UpdatedAt: now,
		},
	}).Error; err != nil {
		t.Fatal(err)
	}

	previewStart := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	previewEnd := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	scheduleCalls := 0
	preview, err := repository.PreviewExecutableSchedule(
		context.Background(),
		"project",
		"task",
		application.PreviewExecutableInput{
			RoleID:        &roleID,
			AssigneeID:    &assigneeID,
			EffortMinutes: &previewEffort,
			LagDays:       2,
		},
		now.Add(time.Hour),
		func(ctx context.Context, projectID string) error {
			scheduleCalls++
			if projectID != "project" || !sharedpersistence.ScheduleMutationSerialized(ctx) {
				t.Fatalf("schedule context project=%q serialized=%v", projectID, sharedpersistence.ScheduleMutationSerialized(ctx))
			}
			tx := sharedpersistence.Transaction(ctx, database)
			var pending nodeModel
			if err := tx.First(&pending, "id = ?", "task").Error; err != nil {
				return err
			}
			if pending.AssigneeID == nil || *pending.AssigneeID != assigneeID || pending.EffortMinutes == nil || *pending.EffortMinutes != previewEffort || pending.LagDays != 2 {
				t.Fatalf("scheduler pending draft=%#v", pending)
			}
			if pending.ExecutionStart == nil || !pending.ExecutionStart.Equal(originalStart) || pending.ExecutionEnd == nil || !pending.ExecutionEnd.Equal(originalEnd) || pending.CommitmentStart == nil || !pending.CommitmentStart.Equal(originalStart) || pending.CommitmentEnd == nil || !pending.CommitmentEnd.Equal(originalEnd) {
				t.Fatalf("generated dates were not preserved until scheduling: %#v", pending)
			}
			if err := tx.Model(&nodeModel{}).Where("id = ?", "task").Updates(map[string]any{
				"execution_start":  previewStart,
				"execution_end":    previewEnd,
				"commitment_start": previewStart,
				"commitment_end":   previewEnd,
			}).Error; err != nil {
				return err
			}
			if err := tx.Create(&schedulePreviewAllocationRecord{
				TaskID: "task", AssigneeID: assigneeID, Timeline: "execution",
				AllocationDate: previewStart, AllocatedMinutes: "480",
				RemainingCapacityMinutes: "0", Sequence: 1,
			}).Error; err != nil {
				return err
			}
			return tx.Model(&schedulePreviewProjectRecord{}).
				Where("id = ?", "project").
				Updates(map[string]any{
					"start_date":       previewStart,
					"end_date":         previewEnd,
					"schedule_version": 7,
				}).Error
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if scheduleCalls != 1 || preview == nil {
		t.Fatalf("schedule calls=%d preview=%#v", scheduleCalls, preview)
	}
	if preview.Task == nil || preview.Task.Executable.ExecutionTimeline.Start == nil || !preview.Task.Executable.ExecutionTimeline.Start.Equal(previewStart) || preview.Task.Executable.ExecutionTimeline.End == nil || !preview.Task.Executable.ExecutionTimeline.End.Equal(previewEnd) {
		t.Fatalf("preview task=%#v", preview.Task)
	}

	var stored nodeModel
	if err := database.First(&stored, "id = ?", "task").Error; err != nil {
		t.Fatal(err)
	}
	if stored.AssigneeID == nil || *stored.AssigneeID != originalAssigneeID || stored.EffortMinutes == nil || *stored.EffortMinutes != originalEffort || stored.LagDays != 0 {
		t.Fatalf("preview persisted draft fields=%#v", stored)
	}
	if stored.ExecutionStart == nil || !stored.ExecutionStart.Equal(originalStart) || stored.ExecutionEnd == nil || !stored.ExecutionEnd.Equal(originalEnd) || stored.CommitmentStart == nil || !stored.CommitmentStart.Equal(originalStart) || stored.CommitmentEnd == nil || !stored.CommitmentEnd.Equal(originalEnd) {
		t.Fatalf("preview persisted generated dates=%#v", stored)
	}
	if !stored.UpdatedAt.Equal(now) {
		t.Fatalf("preview persisted updated_at=%v", stored.UpdatedAt)
	}
	var dependencyCount int64
	if err := database.Model(&schedulePreviewDependencyRecord{}).
		Where("id = ?", "preview-automatic").Count(&dependencyCount).Error; err != nil {
		t.Fatal(err)
	}
	if dependencyCount != 0 {
		t.Fatalf("preview persisted automatic dependency count=%d", dependencyCount)
	}
	var allocationCount int64
	if err := database.Model(&schedulePreviewAllocationRecord{}).
		Where("task_id = ?", "task").Count(&allocationCount).Error; err != nil {
		t.Fatal(err)
	}
	if allocationCount != 0 {
		t.Fatalf("preview persisted allocation count=%d", allocationCount)
	}
	var projectProjection schedulePreviewProjectRecord
	if err := database.First(&projectProjection, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	if projectProjection.StartDate != nil || projectProjection.EndDate != nil || projectProjection.ScheduleVersion != 0 {
		t.Fatalf("preview persisted Project projection=%#v", projectProjection)
	}
}

func TestPreviewExecutableScheduleRejectsIncompleteDraftWithoutCallingScheduler(t *testing.T) {
	repository, database := reopenTestDB(t)
	now := time.Date(2026, 7, 31, 6, 13, 0, 0, time.UTC)
	if err := database.Create(&reopenProjectRecord{
		ID: "project", Name: "Alpha", NameKey: "alpha", Status: "open", AutomaticScheduling: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&nodeModel{
		ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task",
		Position: 1, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	calls := 0
	value, err := repository.PreviewExecutableSchedule(
		context.Background(),
		"project",
		"task",
		application.PreviewExecutableInput{LagDays: 0},
		now.Add(time.Hour),
		func(context.Context, string) error {
			calls++
			return nil
		},
	)
	if !errors.Is(err, domain.ErrSchedulePreviewIncomplete) || value != nil || calls != 0 {
		t.Fatalf("value=%#v err=%v calls=%d", value, err, calls)
	}
}

func TestPreviewExecutableScheduleRejectsUnavailableStateWithoutCallingScheduler(t *testing.T) {
	cases := []struct {
		name                string
		status              string
		automaticScheduling bool
		completed           bool
	}{
		{name: "automatic scheduling off", status: "open", automaticScheduling: false},
		{name: "locked project", status: "locked", automaticScheduling: true},
		{name: "completed task", status: "open", automaticScheduling: true, completed: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repository, database := reopenTestDB(t)
			now := time.Date(2026, 7, 31, 6, 13, 0, 0, time.UTC)
			if err := database.Create(&reopenProjectRecord{
				ID: "project", Name: "Alpha", NameKey: "alpha", Status: tc.status,
				AutomaticScheduling: tc.automaticScheduling,
			}).Error; err != nil {
				t.Fatal(err)
			}
			var actualStart, actualEnd *time.Time
			if tc.completed {
				value := now.Add(-time.Hour)
				actualStart, actualEnd = &value, &value
			}
			if err := database.Create(&nodeModel{
				ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task",
				Position: 1, ActualStart: actualStart, ActualEnd: actualEnd, CreatedAt: now, UpdatedAt: now,
			}).Error; err != nil {
				t.Fatal(err)
			}
			roleID, assigneeID, effortMinutes := "role", "member", 480
			calls := 0
			value, err := repository.PreviewExecutableSchedule(
				context.Background(),
				"project",
				"task",
				application.PreviewExecutableInput{
					RoleID: &roleID, AssigneeID: &assigneeID, EffortMinutes: &effortMinutes, LagDays: 0,
				},
				now.Add(time.Hour),
				func(context.Context, string) error {
					calls++
					return nil
				},
			)
			if !errors.Is(err, domain.ErrSchedulePreviewUnavailable) || value != nil || calls != 0 {
				t.Fatalf("value=%#v err=%v calls=%d", value, err, calls)
			}
		})
	}
}
