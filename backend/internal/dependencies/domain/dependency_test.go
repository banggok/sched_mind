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
