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

func TestReopenCompletedExecutablePreservesPlanningAndIdentity_AC1_AC2_AC4_AC5(t *testing.T) {
	created := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 7, 2, 8, 0, 0, 0, time.UTC)
	reopened := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	parent, role, assignee, effort := "parent", "role", "member", 390
	executionStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	executionEnd := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	commitmentStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	commitmentEnd := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)
	actualEnd := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	node := Node{
		ID: "task", ProjectID: "project", ParentID: &parent, Name: "Build API", Position: 3,
		Executable: ExecutableFields{
			RoleID: &role, AssigneeID: &assignee, EffortMinutes: &effort,
			ExecutionTimeline:  Timeline{Start: &executionStart, End: &executionEnd},
			CommitmentTimeline: Timeline{Start: &commitmentStart, End: &commitmentEnd},
			ActualEnd:          &actualEnd,
		},
		Children: []Node{}, CreatedAt: created, UpdatedAt: updated,
	}
	before := node
	if err := node.Reopen(reopened); err != nil {
		t.Fatal(err)
	}
	if node.Executable.ActualEnd != nil {
		t.Fatalf("actual end was not cleared: %v", node.Executable.ActualEnd)
	}
	if node.UpdatedAt != reopened {
		t.Fatalf("updated at=%v want %v", node.UpdatedAt, reopened)
	}
	if node.ID != before.ID || node.ProjectID != before.ProjectID || node.ParentID == nil || *node.ParentID != *before.ParentID || node.Name != before.Name || node.Position != before.Position || node.HasChildren != before.HasChildren || node.CreatedAt != before.CreatedAt {
		t.Fatalf("identity or structural data changed: before=%#v after=%#v", before, node)
	}
	if *node.Executable.RoleID != *before.Executable.RoleID || *node.Executable.AssigneeID != *before.Executable.AssigneeID || *node.Executable.EffortMinutes != *before.Executable.EffortMinutes || *node.Executable.ExecutionTimeline.Start != *before.Executable.ExecutionTimeline.Start || *node.Executable.ExecutionTimeline.End != *before.Executable.ExecutionTimeline.End || *node.Executable.CommitmentTimeline.Start != *before.Executable.CommitmentTimeline.Start || *node.Executable.CommitmentTimeline.End != *before.Executable.CommitmentTimeline.End {
		t.Fatalf("planning data changed: before=%#v after=%#v", before.Executable, node.Executable)
	}
}

func TestReopenRejectsUnfinishedAndGrouping_AC2_AC13(t *testing.T) {
	now := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	unfinished, _ := New("task", "project", nil, "Task", 1, now)
	if err := unfinished.Reopen(now.Add(time.Hour)); !errors.Is(err, ErrTaskNotCompleted) {
		t.Fatalf("unfinished error=%v", err)
	}
	actualEnd := now
	group := Node{ID: "group", ProjectID: "project", Name: "Group", HasChildren: true, Executable: ExecutableFields{ActualEnd: &actualEnd}, UpdatedAt: now}
	if err := group.Reopen(now.Add(time.Hour)); !errors.Is(err, ErrExecutableOnly) {
		t.Fatalf("group error=%v", err)
	}
}

func TestDedicatedReopenDoesNotWeakenCompletedGenericMutations_AC6(t *testing.T) {
	now := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	actualEnd := now
	node := Node{ID: "task", ProjectID: "project", Name: "Task", Executable: ExecutableFields{ActualEnd: &actualEnd}, UpdatedAt: now}
	if err := node.Rename("Changed", now.Add(time.Hour)); !errors.Is(err, ErrCompletedReadOnly) {
		t.Fatalf("rename error=%v", err)
	}
	if err := node.UpdateExecutable(ExecutableFields{}, false, true, now.Add(time.Hour)); !errors.Is(err, ErrCompletedReadOnly) {
		t.Fatalf("generic update error=%v", err)
	}
	if node.Name != "Task" || node.Executable.ActualEnd == nil {
		t.Fatalf("generic mutation changed completed task: %#v", node)
	}
}
