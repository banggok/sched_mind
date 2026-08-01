package gormrepo

import (
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func actualAllocationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&actualAllocationModel{},
		&allocationMemberModel{},
		&allocationOverrideModel{},
		&allocationHolidayModel{},
		&nodeModel{},
		&projectModel{},
	); err != nil {
		t.Fatal(err)
	}
	return database
}

func TestReplaceActualAllocationsUsesDebtWindowMinimumOverrideAndBAUCapacity(t *testing.T) {
	database := actualAllocationTestDB(t)
	memberID, taskID, otherTaskID := "member", "task", "other"
	assignee := memberID
	effort, otherEffort := 900, 300
	baseline := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	actualStart := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	actualEnd := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)
	otherActualStart, otherActualEnd := baseline, baseline
	values := []any{
		&allocationMemberModel{ID: memberID, DailyCapacity: "8", BufferPercentage: "50"},
		&projectModel{ID: "project", Status: "open"},
		&nodeModel{ID: taskID, ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", AssigneeID: &assignee, EffortMinutes: &effort},
		&nodeModel{ID: otherTaskID, ProjectID: "project", ParentKey: "", Name: "Other", NameKey: "other", AssigneeID: &assignee, EffortMinutes: &otherEffort, ActualStart: &otherActualStart, ActualEnd: &otherActualEnd},
		&allocationOverrideModel{TeamMemberID: memberID, StartDate: actualStart, EndDate: actualStart, Capacity: "6"},
		&allocationOverrideModel{TeamMemberID: memberID, StartDate: actualStart, EndDate: actualStart, Capacity: "4"},
		&actualAllocationModel{TaskID: otherTaskID, AssigneeID: memberID, Timeline: "actual", AllocationDate: baseline, AllocatedMinutes: "300", RemainingCapacityMinutes: "180", Sequence: 1},
	}
	for _, value := range values {
		if err := database.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	task := nodeModel{ID: taskID, ProjectID: "project", AssigneeID: &assignee, EffortMinutes: &effort}

	allocations, err := replaceActualAllocations(database, task, &baseline, actualStart, actualEnd)
	if err != nil {
		t.Fatal(err)
	}
	if len(allocations) != 3 {
		t.Fatalf("allocations = %#v", allocations)
	}
	expectedAllocated := []int{180, 240, 480}
	expectedCapacity := []int{480, 240, 480}
	for index, allocation := range allocations {
		if allocation.AllocatedMinutes != expectedAllocated[index] || allocation.BAUCapacityMinutes != expectedCapacity[index] || allocation.OvercapacityMinutes != 0 {
			t.Fatalf("allocation %d = %#v", index, allocation)
		}
	}
	if allocations[0].RemainingMinutes != 0 || allocations[1].RemainingMinutes != 0 || allocations[2].RemainingMinutes != 0 {
		t.Fatalf("remaining capacity = %#v", allocations)
	}
}

func TestReplaceActualAllocationsDistributesExcessDeterministically(t *testing.T) {
	database := actualAllocationTestDB(t)
	memberID, taskID := "member", "task"
	assignee := memberID
	effort := 1200
	actualStart := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	actualEnd := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	for _, value := range []any{
		&allocationMemberModel{ID: memberID, DailyCapacity: "8", BufferPercentage: "25"},
		&projectModel{ID: "project", Status: "open"},
		&nodeModel{ID: taskID, ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", AssigneeID: &assignee, EffortMinutes: &effort},
	} {
		if err := database.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}

	allocations, err := replaceActualAllocations(database, nodeModel{ID: taskID, ProjectID: "project", AssigneeID: &assignee, EffortMinutes: &effort}, nil, actualStart, actualEnd)
	if err != nil {
		t.Fatal(err)
	}
	if len(allocations) != 2 || allocations[0].AllocatedMinutes != 600 || allocations[1].AllocatedMinutes != 600 || allocations[0].OvercapacityMinutes != 120 || allocations[1].OvercapacityMinutes != 120 {
		t.Fatalf("allocations = %#v", allocations)
	}
}

func TestReplaceActualAllocationsFallsBackAfterNonWorkingActualWindow(t *testing.T) {
	database := actualAllocationTestDB(t)
	memberID, taskID := "member", "task"
	assignee := memberID
	effort := 120
	actualStart := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	actualEnd := time.Date(2026, 8, 9, 0, 0, 0, 0, time.UTC)
	for _, value := range []any{
		&allocationMemberModel{ID: memberID, DailyCapacity: "8", BufferPercentage: "0"},
		&projectModel{ID: "project", Status: "open"},
		&nodeModel{ID: taskID, ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", AssigneeID: &assignee, EffortMinutes: &effort},
	} {
		if err := database.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}

	allocations, err := replaceActualAllocations(database, nodeModel{ID: taskID, ProjectID: "project", AssigneeID: &assignee, EffortMinutes: &effort}, nil, actualStart, actualEnd)
	if err != nil {
		t.Fatal(err)
	}
	expected := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	if len(allocations) != 1 || !allocations[0].Date.Equal(expected) || allocations[0].AllocatedMinutes != effort {
		t.Fatalf("allocations = %#v", allocations)
	}
}
