package gormrepo

import (
	"testing"
	"time"
)

func TestComposeRowsBuildsDeterministicHierarchyAndRecursiveSummary(t *testing.T) {
	start := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	actualStart := start
	actualEnd := end
	rootID := "group"
	effort := 480
	roleID := "development"
	roleName := "Development"
	projects := []projectRow{
		{ID: "alpha", Name: "Alpha", Status: "open", Priority: 1},
		{ID: "beta", Name: "Beta", Status: "locked", Priority: 2},
	}
	nodes := []nodeRow{
		{ID: rootID, ProjectID: "alpha", Name: "Delivery", Position: 1},
		{ID: "unscheduled", ProjectID: "alpha", ParentID: &rootID, Name: "Unscheduled", Position: 2},
		{ID: "completed", ProjectID: "alpha", ParentID: &rootID, Name: "Completed", Position: 1, RoleID: &roleID, RoleName: &roleName, EffortMinutes: &effort, Start: &start, End: &end, ActualStart: &actualStart, ActualEnd: &actualEnd},
	}

	rows, anchor, err := composeRows(projects, nodes)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 5 {
		t.Fatalf("row count = %d: %#v", len(rows), rows)
	}
	if rows[0].ID != "alpha" || rows[1].ID != "group" || rows[2].ID != "completed" || rows[3].ID != "unscheduled" || rows[4].ID != "beta" {
		t.Fatalf("row order = %#v", []string{rows[0].ID, rows[1].ID, rows[2].ID, rows[3].ID, rows[4].ID})
	}
	if rows[1].Kind != "group" || rows[1].WBSNumber != "1" || rows[2].WBSNumber != "1.1" || rows[3].WBSNumber != "1.2" {
		t.Fatalf("hierarchy = %#v", rows[1:4])
	}
	if rows[0].EffortMinutes == nil || *rows[0].EffortMinutes != effort || !rows[0].IncompleteEffort || !rows[0].IncompleteSchedule {
		t.Fatalf("project summary = %#v", rows[0])
	}
	if !rows[2].Completed || rows[3].Completed {
		t.Fatalf("completion flags = completed:%v unscheduled:%v", rows[2].Completed, rows[3].Completed)
	}
	if rows[1].EffectiveLifecycle != "open" || rows[2].EffectiveLifecycle != "open" || rows[4].EffectiveLifecycle != "locked" {
		t.Fatalf("effective lifecycle projection = group:%q task:%q beta:%q", rows[1].EffectiveLifecycle, rows[2].EffectiveLifecycle, rows[4].EffectiveLifecycle)
	}
	if rows[2].RoleID == nil || *rows[2].RoleID != roleID || rows[2].RoleName == nil || *rows[2].RoleName != roleName {
		t.Fatalf("task role = %#v", rows[2])
	}
	if anchor == nil || !anchor.Equal(start) {
		t.Fatalf("working-day anchor = %v", anchor)
	}
}
