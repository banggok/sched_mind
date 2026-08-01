package gormrepo

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCreateEmptyTaskSkipsSchedulerEvenWhenProjectHasCompletedTask_US6_AC29_US4_AC23(t *testing.T) {
	repository, database := dependencyTestDB(t)
	now := time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC)
	actualEnd := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	effort := 480
	assigneeID := "member"

	if err := database.Create(&projectModel{
		ID:                  "project",
		Status:              "open",
		AutomaticScheduling: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&nodeModel{
		ID:              "completed",
		ProjectID:       "project",
		ParentKey:       "",
		Name:            "Completed Task",
		NameKey:         "completed task",
		Position:        1,
		AssigneeID:      &assigneeID,
		EffortMinutes:   &effort,
		ExecutionStart:  &actualEnd,
		ExecutionEnd:    &actualEnd,
		CommitmentStart: &actualEnd,
		CommitmentEnd:   &actualEnd,
		ActualStart:     &actualEnd,
		ActualEnd:       &actualEnd,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	scheduleCalls := 0
	invalidateCalls := 0
	created, err := repository.Create(
		context.Background(),
		"new-task",
		"project",
		nil,
		"New Task",
		false,
		now,
		func(context.Context, string) error {
			scheduleCalls++
			return errors.New("scheduler must not run for empty create")
		},
		func(context.Context, []string) error {
			invalidateCalls++
			return errors.New("portfolio invalidation must not run for empty create")
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if created == nil || created.ID != "new-task" {
		t.Fatalf("created=%#v, want new-task", created)
	}
	if scheduleCalls != 0 || invalidateCalls != 0 {
		t.Fatalf("schedule calls=%d invalidate calls=%d, want both 0", scheduleCalls, invalidateCalls)
	}

	var stored nodeModel
	if err := database.First(&stored, "id = ?", "new-task").Error; err != nil {
		t.Fatal(err)
	}
	if stored.AssigneeID != nil || stored.EffortMinutes != nil || stored.ExecutionStart != nil || stored.CommitmentStart != nil || stored.ActualEnd != nil {
		t.Fatalf("new Task must remain empty and unscheduled: %#v", stored)
	}

	var completed nodeModel
	if err := database.First(&completed, "id = ?", "completed").Error; err != nil {
		t.Fatal(err)
	}
	if completed.ActualEnd == nil || !completed.ActualEnd.Equal(actualEnd) {
		t.Fatalf("completed Task changed: %#v", completed)
	}
}

func TestCreateChildConversionStillRunsSchedulerAndRollsBackOnFailure_US6_AC29_US4_AC23_AC25(t *testing.T) {
	repository, database := dependencyTestDB(t)
	now := time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC)
	effort := 480
	assigneeID := "member"

	if err := database.Create(&projectModel{
		ID:                  "project",
		Status:              "open",
		AutomaticScheduling: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&nodeModel{
		ID:            "parent",
		ProjectID:     "project",
		ParentKey:     "",
		Name:          "Parent Task",
		NameKey:       "parent task",
		Position:      1,
		AssigneeID:    &assigneeID,
		EffortMinutes: &effort,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	scheduleFailure := errors.New("schedule failed")
	scheduleCalls := 0
	parentID := "parent"
	created, err := repository.Create(
		context.Background(),
		"child",
		"project",
		&parentID,
		"Child Task",
		true,
		now,
		func(context.Context, string) error {
			scheduleCalls++
			return scheduleFailure
		},
		func(context.Context, []string) error { return nil },
	)
	if created != nil || !errors.Is(err, scheduleFailure) {
		t.Fatalf("created=%#v err=%v, want scheduler failure", created, err)
	}
	if scheduleCalls != 1 {
		t.Fatalf("schedule calls=%d, want 1", scheduleCalls)
	}

	var childCount int64
	if err := database.Model(&nodeModel{}).Where("id = ?", "child").Count(&childCount).Error; err != nil {
		t.Fatal(err)
	}
	if childCount != 0 {
		t.Fatalf("child count=%d, want rollback", childCount)
	}
	var parent nodeModel
	if err := database.First(&parent, "id = ?", "parent").Error; err != nil {
		t.Fatal(err)
	}
	if parent.AssigneeID == nil || *parent.AssigneeID != assigneeID || parent.EffortMinutes == nil || *parent.EffortMinutes != effort {
		t.Fatalf("parent executable state changed after rollback: %#v", parent)
	}
}
