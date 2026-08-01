package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestProjectLifecycleAndValidation(t *testing.T) {
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	project, err := NewProject("id", "  Alpha  ", 1, now)
	if err != nil || project == nil {
		t.Fatalf("create: %#v %v", project, err)
	}
	if project.Name != "Alpha" || project.Status != StatusOpen || !project.AutoCalculateDate || !project.AutomaticScheduling || project.ProjectBuffer != 20 {
		t.Fatalf("unexpected defaults: %#v", project)
	}
	if _, err := NewProject("id", " ", 1, now); !errors.Is(err, ErrNameRequired) {
		t.Fatalf("required: %v", err)
	}
	if _, err := NewProject("id", strings.Repeat("x", 101), 1, now); !errors.Is(err, ErrNameTooLong) {
		t.Fatalf("length: %v", err)
	}
	if _, err := NewProject("id", "Alpha", 0, now); !errors.Is(err, ErrPriorityInvalid) {
		t.Fatalf("priority: %v", err)
	}
	if err := project.ChangeStatus(StatusLocked, false, false, nil, nil, now.Add(time.Hour)); !errors.Is(err, ErrCannotLockWithoutTasks) {
		t.Fatalf("zero leaf lock: %v", err)
	}
	if err := project.ChangeStatus(StatusLocked, true, false, nil, nil, now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := project.ChangeStatus(StatusOpen, false, false, nil, nil, now.Add(2*time.Hour)); err != nil {
		t.Fatalf("locked reopen: %v", err)
	}
	if project.Status != StatusOpen || project.LockedExecutionSnapshot != nil || project.LockedCommitmentSnapshot != nil {
		t.Fatalf("locked reopen state: %#v", project)
	}
	if err := project.ChangeStatus(StatusClosed, false, false, nil, nil, now.Add(3*time.Hour)); !errors.Is(err, ErrCannotCloseWithoutTasks) {
		t.Fatalf("zero leaf close: %v", err)
	}
	if err := project.ChangeStatus(StatusClosed, true, true, nil, nil, now.Add(4*time.Hour)); !errors.Is(err, ErrCannotCloseWithActiveTasks) {
		t.Fatalf("active close: %v", err)
	}
	if err := project.ChangeStatus(StatusClosed, true, false, nil, nil, now.Add(5*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := project.Rename("Beta", now); !errors.Is(err, ErrClosedReadOnly) {
		t.Fatalf("closed edit: %v", err)
	}
	if err := project.ChangeStatus(StatusOpen, false, false, nil, nil, now.Add(6*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if project.LockedExecutionSnapshot != nil || project.LockedCommitmentSnapshot != nil {
		t.Fatal("reopen must clear locked snapshots")
	}
}

func TestProjectSettingsValidationAndStatusRules(t *testing.T) {
	project, _ := NewProject("id", "Alpha", 1, time.Now())
	anchor := time.Date(2026, 8, 3, 15, 30, 0, 0, time.FixedZone("test", 7*60*60))
	if err := project.UpdateSettings(false, &anchor, 0, time.Now()); err != nil || project.AutomaticScheduling || project.ProjectBuffer != 0 {
		t.Fatalf("valid settings: %#v %v", project, err)
	}
	if project.SchedulingStartDate == nil || project.SchedulingStartDate.Format("2006-01-02T15:04:05Z07:00") != "2026-08-03T00:00:00Z" {
		t.Fatalf("date-only scheduling anchor: %#v", project.SchedulingStartDate)
	}
	if err := project.UpdateSettings(true, nil, 101, time.Now()); !errors.Is(err, ErrProjectBufferInvalid) {
		t.Fatalf("buffer: %v", err)
	}
	project.Status = StatusLocked
	if err := project.UpdateSettings(true, nil, 20, time.Now()); !errors.Is(err, ErrSettingsReadOnly) {
		t.Fatalf("locked: %v", err)
	}
	project.Status = StatusClosed
	if err := project.UpdateSettings(true, nil, 20, time.Now()); !errors.Is(err, ErrSettingsReadOnly) {
		t.Fatalf("closed: %v", err)
	}
}

func TestProjectDeleteAndPriorityDirection(t *testing.T) {
	project, _ := NewProject("id", "Alpha", 1, time.Now())
	if err := project.CanDelete(true); !errors.Is(err, ErrHasChildren) {
		t.Fatalf("children: %v", err)
	}
	if err := project.CanDelete(false); err != nil {
		t.Fatal(err)
	}
	if direction, err := ParsePriorityDirection("up"); err != nil || direction != PriorityUp {
		t.Fatalf("direction: %v %v", direction, err)
	}
	if _, err := ParsePriorityDirection("left"); !errors.Is(err, ErrPriorityDirectionInvalid) {
		t.Fatalf("invalid direction: %v", err)
	}
}
