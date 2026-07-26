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
	if project.Name != "Alpha" || project.Status != StatusOpen || !project.AutoCalculateDate || !project.AutoDependencyByAssignee {
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
	if err := project.ChangeStatus(StatusOpen, false, false, nil, nil, now); !errors.Is(err, ErrStatusTransitionNotAllowed) {
		t.Fatalf("locked reopen: %v", err)
	}
	if err := project.ChangeStatus(StatusClosed, false, false, nil, nil, now); !errors.Is(err, ErrCannotCloseWithoutTasks) {
		t.Fatalf("zero leaf close: %v", err)
	}
	if err := project.ChangeStatus(StatusClosed, true, true, nil, nil, now); !errors.Is(err, ErrCannotCloseWithActiveTasks) {
		t.Fatalf("active close: %v", err)
	}
	if err := project.ChangeStatus(StatusClosed, true, false, nil, nil, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := project.Rename("Beta", now); !errors.Is(err, ErrClosedReadOnly) {
		t.Fatalf("closed edit: %v", err)
	}
	if err := project.ChangeStatus(StatusOpen, false, false, nil, nil, now.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if project.LockedExecutionSnapshot != nil || project.LockedCommitmentSnapshot != nil {
		t.Fatal("reopen must retain nullable snapshots")
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
