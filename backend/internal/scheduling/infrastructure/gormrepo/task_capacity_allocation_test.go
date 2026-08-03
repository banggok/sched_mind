package gormrepo

import (
	"context"
	"math/big"
	"testing"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
)

func TestTaskDailyLimitRoundsHalfUpAndKeepsPositiveMinimum_US63_AC6_AC7(t *testing.T) {
	tests := []struct {
		name       string
		capacity   int64
		percentage int
		want       int64
	}{
		{"five point five hours at fifty percent", 330, 50, 180},
		{"five point five hours at twenty percent", 330, 20, 60},
		{"one percent keeps half hour minimum", 480, 1, 30},
		{"zero capacity stays zero", 0, 50, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := taskDailyLimit(big.NewRat(test.capacity, 1), test.percentage)
			if got.Cmp(big.NewRat(test.want, 1)) != 0 {
				t.Fatalf("limit = %s, want %d", got.RatString(), test.want)
			}
		})
	}
}

func TestConcurrentSameAssigneeAllocationUsesDailyLimits_US63_AC9_AC11_AC15(t *testing.T) {
	repository, database := schedulerRepository(t)
	seedProject(t, database, projectModel{ID: "project", Status: "open", Priority: 1, AutomaticScheduling: true, SchedulingStartDate: datePointer(mustDate("2026-08-03"))})
	seedMember(t, database, "member", "8", "0")
	for _, task := range []taskModel{
		{ID: "a", ProjectID: "project", ParentKey: "", Position: 1, Name: "A", AssigneeID: textPointer("member"), EffortMinutes: intPointer(240), CapacityAllocationPercentage: 100},
		{ID: "b", ProjectID: "project", ParentKey: "", Position: 2, Name: "B", AssigneeID: textPointer("member"), EffortMinutes: intPointer(240), CapacityAllocationPercentage: 20},
		{ID: "c", ProjectID: "project", ParentKey: "", Position: 3, Name: "C", AssigneeID: textPointer("member"), EffortMinutes: intPointer(240), CapacityAllocationPercentage: 100},
	} {
		seedTask(t, database, task)
	}
	if err := repository.RecalculatePortfolio(context.Background(), []string{"project"}); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"a": "240.000000", "b": "90.000000", "c": "150.000000"}
	for taskID, minutes := range want {
		rows := loadAllocations(t, database, taskID, "execution")
		if len(rows) == 0 || schedulingdomain.DateKey(rows[0].AllocationDate) != "2026-08-03" || rows[0].AllocatedMinutes != minutes {
			t.Fatalf("%s first allocation = %#v", taskID, rows)
		}
	}
	var count int64
	if err := database.Model(&dependencyModel{}).Where("automatic_owned = ?", true).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("parallel start automatic dependencies = %d", count)
	}
}

func TestConfiguredPercentagesAboveOneHundredRemainCapacitySafe_US63_AC10(t *testing.T) {
	repository, database := schedulerRepository(t)
	seedProject(t, database, projectModel{ID: "project", Status: "open", Priority: 1, AutomaticScheduling: true, SchedulingStartDate: datePointer(mustDate("2026-08-03"))})
	seedMember(t, database, "member", "8", "0")
	for _, task := range []taskModel{
		{ID: "a", ProjectID: "project", ParentKey: "", Position: 1, Name: "A", AssigneeID: textPointer("member"), EffortMinutes: intPointer(480), CapacityAllocationPercentage: 70},
		{ID: "b", ProjectID: "project", ParentKey: "", Position: 2, Name: "B", AssigneeID: textPointer("member"), EffortMinutes: intPointer(480), CapacityAllocationPercentage: 70},
	} {
		seedTask(t, database, task)
	}
	if err := repository.RecalculatePortfolio(context.Background(), []string{"project"}); err != nil {
		t.Fatal(err)
	}
	a, b := loadAllocations(t, database, "a", "execution"), loadAllocations(t, database, "b", "execution")
	if len(a) == 0 || len(b) == 0 || a[0].AllocatedMinutes != "330.000000" || b[0].AllocatedMinutes != "150.000000" {
		t.Fatalf("a=%#v b=%#v", a, b)
	}
}

func TestManualFixedAllocationMayExceedLimitAndReservesAutomaticCapacity_US63_AC18_AC19(t *testing.T) {
	repository, database := schedulerRepository(t)
	manual := projectModel{ID: "manual", Status: "open", Priority: 1, AutomaticScheduling: false}
	automatic := projectModel{ID: "automatic", Status: "open", Priority: 2, AutomaticScheduling: true, SchedulingStartDate: datePointer(mustDate("2026-08-03"))}
	seedProject(t, database, manual)
	seedProject(t, database, automatic)
	seedMember(t, database, "member", "8", "0")
	start, end := mustDate("2026-08-03"), mustDate("2026-08-04")
	seedTask(t, database, taskModel{ID: "manual-task", ProjectID: "manual", ParentKey: "", Position: 1, Name: "Manual", AssigneeID: textPointer("member"), EffortMinutes: intPointer(600), CapacityAllocationPercentage: 20, ExecutionStart: &start, ExecutionEnd: &end, CommitmentStart: &start, CommitmentEnd: &end})
	seedTask(t, database, taskModel{ID: "automatic-task", ProjectID: "automatic", ParentKey: "", Position: 1, Name: "Automatic", AssigneeID: textPointer("member"), EffortMinutes: intPointer(480), CapacityAllocationPercentage: 100})
	if err := repository.RecalculatePortfolio(context.Background(), []string{"manual"}); err != nil {
		t.Fatal(err)
	}
	manualRows := loadAllocations(t, database, "manual-task", "execution")
	if len(manualRows) != 2 || manualRows[0].AllocatedMinutes != "300.000000" || manualRows[1].AllocatedMinutes != "300.000000" {
		t.Fatalf("manual rows=%#v", manualRows)
	}
	automaticRows := loadAllocations(t, database, "automatic-task", "execution")
	want := []string{"180.000000", "180.000000", "120.000000"}
	if len(automaticRows) != len(want) {
		t.Fatalf("automatic rows=%#v", automaticRows)
	}
	for index := range want {
		if automaticRows[index].AllocatedMinutes != want[index] {
			t.Fatalf("automatic row %d=%#v", index, automaticRows[index])
		}
	}
	storedManual := loadScheduledTask(t, database, "manual-task")
	assertDate(t, "manual start unchanged", storedManual.ExecutionStart, "2026-08-03")
	assertDate(t, "manual end unchanged", storedManual.ExecutionEnd, "2026-08-04")
}

func TestManualFixedAllocationUsesEligibleZeroCapacityWeekdays_US63_AC18_AC20(t *testing.T) {
	repository, database := schedulerRepository(t)
	seedProject(t, database, projectModel{ID: "manual", Status: "open", Priority: 1, AutomaticScheduling: false})
	seedMember(t, database, "member", "0", "0")
	start, end := mustDate("2026-08-03"), mustDate("2026-08-04")
	seedTask(t, database, taskModel{ID: "manual-task", ProjectID: "manual", ParentKey: "", Position: 1, Name: "Manual", AssigneeID: textPointer("member"), EffortMinutes: intPointer(120), CapacityAllocationPercentage: 20, ExecutionStart: &start, ExecutionEnd: &end, CommitmentStart: &start, CommitmentEnd: &end})
	if err := repository.RecalculatePortfolio(context.Background(), []string{"manual"}); err != nil {
		t.Fatal(err)
	}
	rows := loadAllocations(t, database, "manual-task", "execution")
	if len(rows) != 2 || rows[0].AllocatedMinutes != "60.000000" || rows[1].AllocatedMinutes != "60.000000" {
		t.Fatalf("zero-capacity weekday rows=%#v", rows)
	}
}

func TestManualFixedAllocationUsesManualStartWhenRangeHasNoWorkingDate_US63_AC20(t *testing.T) {
	repository, database := schedulerRepository(t)
	seedProject(t, database, projectModel{ID: "manual", Status: "open", Priority: 1, AutomaticScheduling: false})
	seedMember(t, database, "member", "8", "0")
	start, end := mustDate("2026-08-08"), mustDate("2026-08-09")
	seedTask(t, database, taskModel{ID: "manual-task", ProjectID: "manual", ParentKey: "", Position: 1, Name: "Manual", AssigneeID: textPointer("member"), EffortMinutes: intPointer(120), CapacityAllocationPercentage: 20, ExecutionStart: &start, ExecutionEnd: &end, CommitmentStart: &start, CommitmentEnd: &end})
	if err := repository.RecalculatePortfolio(context.Background(), []string{"manual"}); err != nil {
		t.Fatal(err)
	}
	rows := loadAllocations(t, database, "manual-task", "execution")
	if len(rows) != 1 || schedulingdomain.DateKey(rows[0].AllocationDate) != "2026-08-08" || rows[0].AllocatedMinutes != "120.000000" {
		t.Fatalf("no-working-date rows=%#v", rows)
	}
}
