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

func TestReplaceActualAllocationsUsesActualWindowAndEqualizesUnavoidableOvercapacity_US62_AC7_AC11(t *testing.T) {
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
		&nodeModel{ID: taskID, ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", AssigneeID: &assignee, EffortMinutes: &effort, CapacityAllocationPercentage: 20},
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
	if len(allocations) != 2 {
		t.Fatalf("allocations = %#v", allocations)
	}
	expectedAllocated := []int{330, 570}
	expectedCapacity := []int{240, 480}
	for index, allocation := range allocations {
		if allocation.AllocatedMinutes != expectedAllocated[index] || allocation.BAUCapacityMinutes != expectedCapacity[index] || allocation.OvercapacityMinutes != 90 {
			t.Fatalf("allocation %d = %#v", index, allocation)
		}
	}
	if allocations[0].RemainingMinutes != 0 || allocations[1].RemainingMinutes != 0 {
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

func TestReplaceActualAllocationsFallsBackOnActualEndForNonWorkingWindow_US62_AC7(t *testing.T) {
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
	expected := actualEnd
	if len(allocations) != 1 || !allocations[0].Date.Equal(expected) || allocations[0].AllocatedMinutes != effort {
		t.Fatalf("allocations = %#v", allocations)
	}
}

func TestReplaceActualAllocationsProgressivelyRebalancesAroundExistingActualLoad_US62_AC8_AC9(t *testing.T) {
	database := actualAllocationTestDB(t)
	memberID, taskID, existingID := "member", "task", "existing"
	assignee := memberID
	effort, existingEffort := 600, 360
	start, end := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC), time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)
	for _, value := range []any{
		&allocationMemberModel{ID: memberID, DailyCapacity: "8", BufferPercentage: "0"},
		&projectModel{ID: "project", Status: "open"},
		&nodeModel{ID: taskID, ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", AssigneeID: &assignee, EffortMinutes: &effort},
		&nodeModel{ID: existingID, ProjectID: "project", ParentKey: "", Name: "Existing", NameKey: "existing", AssigneeID: &assignee, EffortMinutes: &existingEffort, ActualStart: &start, ActualEnd: &start},
		&actualAllocationModel{TaskID: existingID, AssigneeID: memberID, Timeline: "actual", AllocationDate: start, AllocatedMinutes: "360", RemainingCapacityMinutes: "120", Sequence: 1},
	} {
		if err := database.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	allocations, err := replaceActualAllocations(database, nodeModel{ID: taskID, ProjectID: "project", AssigneeID: &assignee, EffortMinutes: &effort, CapacityAllocationPercentage: 20}, nil, start, end)
	if err != nil {
		t.Fatal(err)
	}
	want := []int{120, 240, 240}
	if len(allocations) != len(want) {
		t.Fatalf("allocations=%#v", allocations)
	}
	for index := range want {
		if allocations[index].AllocatedMinutes != want[index] {
			t.Fatalf("allocation %d=%#v", index, allocations[index])
		}
	}
	var existing actualAllocationModel
	if err := database.First(&existing, "task_id = ? AND timeline = ?", existingID, "actual").Error; err != nil {
		t.Fatal(err)
	}
	if existing.AllocatedMinutes != "360" {
		t.Fatalf("existing Actual row changed=%#v", existing)
	}
}
