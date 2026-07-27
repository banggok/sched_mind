package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNodeStateAndExecutableValidation(t *testing.T) {
	node, err := New("w", "p", nil, " Build ", 1, time.Now())
	if err != nil || node == nil || !node.IsExecutable() || node.Name != "Build" {
		t.Fatalf("unexpected new node: %#v %v", node, err)
	}
	effort := 30
	if err := node.UpdateExecutable(ExecutableFields{EffortMinutes: &effort}, false, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	node.HasChildren = true
	if err := node.UpdateExecutable(ExecutableFields{}, false, true, time.Now()); !errors.Is(err, ErrExecutableOnly) {
		t.Fatalf("expected executable-only, got %v", err)
	}
}

func TestManualTimelineAndCompletion(t *testing.T) {
	node, _ := New("w", "p", nil, "Build", 1, time.Now())
	start := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)
	fields := ExecutableFields{ExecutionTimeline: Timeline{Start: &start, End: &end}}
	if err := node.UpdateExecutable(fields, true, true, time.Now()); !errors.Is(err, ErrManualTimeline) {
		t.Fatalf("expected manual restriction, got %v", err)
	}
	if err := node.UpdateExecutable(fields, false, true, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := node.Complete(end, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := node.Rename("Other", time.Now()); !errors.Is(err, ErrCompletedReadOnly) {
		t.Fatalf("expected completed restriction, got %v", err)
	}
}
