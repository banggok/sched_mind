package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewDependency(t *testing.T) {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	value, err := New("dependency", "task-a", "task-b", now)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if value == nil || value.BlockingTaskID != "task-a" || value.BlockedTaskID != "task-b" {
		t.Fatalf("New() = %#v", value)
	}
}

func TestNewDependencyRejectsSelfReference(t *testing.T) {
	value, err := New("dependency", "task-a", "task-a", time.Now())
	if value != nil {
		t.Fatalf("New() value = %#v, want nil", value)
	}
	if !errors.Is(err, ErrSelfReference) {
		t.Fatalf("New() error = %v", err)
	}
}

func TestDirectionValidation(t *testing.T) {
	if !BlockedBy.Valid() || !Blocks.Valid() {
		t.Fatal("approved directions must be valid")
	}
	if Direction("other").Valid() {
		t.Fatal("unknown direction must be invalid")
	}
}

func TestDependencyOwnershipTransitions_AC23_AC24(t *testing.T) {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	automatic, err := NewAutomatic("dependency", "task-a", "task-b", now)
	if err != nil {
		t.Fatal(err)
	}
	if automatic.Source() != SourceAutomatic || automatic.ManualRemovable() {
		t.Fatalf("automatic source=%q removable=%v", automatic.Source(), automatic.ManualRemovable())
	}

	automatic.KeepAsManual(now.Add(time.Hour))
	if automatic.Source() != SourceShared || !automatic.ManualRemovable() {
		t.Fatalf("shared source=%q removable=%v", automatic.Source(), automatic.ManualRemovable())
	}
	deleted, err := automatic.RemoveManual(now.Add(2 * time.Hour))
	if err != nil || deleted {
		t.Fatalf("RemoveManual()=(delete:%v,error:%v), want preserved automatic relation", deleted, err)
	}
	if automatic.Source() != SourceAutomatic {
		t.Fatalf("source after manual removal = %q", automatic.Source())
	}
}

func TestAutomaticOnlyDependencyRejectsGenericManualRemoval_AC24(t *testing.T) {
	value, err := NewAutomatic("dependency", "task-a", "task-b", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := value.RemoveManual(time.Now())
	if deleted || !errors.Is(err, ErrAutomaticOnlyReadOnly) {
		t.Fatalf("RemoveManual()=(delete:%v,error:%v)", deleted, err)
	}
}
