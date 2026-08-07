package gormrepo

import (
	"testing"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
)

func TestScheduleDatesMatchTaskIgnoresNonTimelineProjectionFields_US62_D01_D02_AC20_AC21_AC22(t *testing.T) {
	day := mustDate("2026-08-03")
	nextDay := mustDate("2026-08-04")
	task := taskModel{
		ExecutionStart:  &day,
		ExecutionEnd:    &day,
		CommitmentStart: &day,
		CommitmentEnd:   &day,
	}
	withReasonOnly := taskSchedule{Start: &day, End: &day, Reason: stringPointerForImpactTest("CAPACITY_CHANGED")}
	if !scheduleDatesMatchTask(task, withReasonOnly, withReasonOnly) {
		t.Fatal("unscheduled reason-only difference must not classify as timeline impact")
	}
	missingAnchorTask := taskModel{}
	missingAnchorSchedule := taskSchedule{Reason: stringPointerForImpactTest(reasonMissingAnchor)}
	if !scheduleDatesMatchTask(missingAnchorTask, missingAnchorSchedule, missingAnchorSchedule) {
		t.Fatal("missing-anchor Project with no existing timeline must not classify as timeline impact")
	}

	cases := []struct {
		name       string
		execution  taskSchedule
		commitment taskSchedule
	}{
		{name: "execution start", execution: taskSchedule{Start: &nextDay, End: &day}, commitment: taskSchedule{Start: &day, End: &day}},
		{name: "execution end", execution: taskSchedule{Start: &day, End: &nextDay}, commitment: taskSchedule{Start: &day, End: &day}},
		{name: "commitment start", execution: taskSchedule{Start: &day, End: &day}, commitment: taskSchedule{Start: &nextDay, End: &day}},
		{name: "commitment end", execution: taskSchedule{Start: &day, End: &day}, commitment: taskSchedule{Start: &day, End: &nextDay}},
		{name: "date to null", execution: taskSchedule{Start: nil, End: &day}, commitment: taskSchedule{Start: &day, End: &day}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if scheduleDatesMatchTask(task, tc.execution, tc.commitment) {
				t.Fatalf("%s difference must classify as timeline impact", tc.name)
			}
		})
	}
}

func TestPersistenceSignatureIncludesHiddenRecalculatedProjectVersion_US62_D04_AC24(t *testing.T) {
	day := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	base := []projectPersistenceChange{{ProjectID: "hidden", Version: 7, StartDate: &day, EndDate: &day}}
	changed := []projectPersistenceChange{{ProjectID: "hidden", Version: 8, StartDate: &day, EndDate: &day}}
	rows := map[schedulingdomain.Timeline]map[string][]allocationModel{
		schedulingdomain.Execution:  {},
		schedulingdomain.Commitment: {},
	}

	state := minimalImpactSignatureState()
	baseSignature := persistenceSignature(state, nil, base, rows, map[string]struct{}{}, map[string]struct{}{})
	changedSignature := persistenceSignature(state, nil, changed, rows, map[string]struct{}{}, map[string]struct{}{})
	if baseSignature == changedSignature {
		t.Fatalf("hidden recalculation version did not change signature: %q", baseSignature)
	}
}

func stringPointerForImpactTest(value string) *string { return &value }

func TestSchedulingStateSignatureCoversHiddenScopedState_US62_D04_AC24(t *testing.T) {
	base := minimalImpactSignatureState()
	baseSignature := schedulingStateSignature(base)

	priorityChanged := minimalImpactSignatureState()
	project := priorityChanged.projects["hidden"]
	project.Priority = 2
	priorityChanged.projects["hidden"] = project
	if schedulingStateSignature(priorityChanged) == baseSignature {
		t.Fatal("hidden Project priority change did not change scheduling-state signature")
	}

	capacityChanged := minimalImpactSignatureState()
	member := capacityChanged.members["member"]
	member.DailyCapacity = "10"
	capacityChanged.members["member"] = member
	if schedulingStateSignature(capacityChanged) == baseSignature {
		t.Fatal("hidden Member capacity change did not change scheduling-state signature")
	}

	dependencyChanged := minimalImpactSignatureState()
	dependencyChanged.dependencies = []dependencyModel{{BlockingTaskID: "task", BlockedTaskID: "other-task"}}
	if schedulingStateSignature(dependencyChanged) == baseSignature {
		t.Fatal("hidden dependency/readiness change did not change scheduling-state signature")
	}
}

func minimalImpactSignatureState() *portfolioState {
	memberID := "member"
	effort := 60
	day := mustDate("2026-08-03")
	return &portfolioState{
		projects: map[string]projectModel{
			"hidden": {ID: "hidden", Status: "open", Priority: 1, AutomaticScheduling: true, SchedulingStartDate: &day, ScheduleVersion: 7},
		},
		tasks: map[string]taskModel{
			"task": {ID: "task", ProjectID: "hidden", Position: 1, AssigneeID: &memberID, EffortMinutes: &effort, CapacityAllocationPercentage: 100},
		},
		members: map[string]memberModel{
			"member": {ID: "member", DailyCapacity: "8", BufferPercentage: "0"},
		},
		overrides:    map[string][]capacityOverrideModel{},
		holidays:     map[string]struct{}{},
		dependencies: nil,
		existingAllocations: map[schedulingdomain.Timeline]map[string][]allocationModel{
			schedulingdomain.Execution:  {},
			schedulingdomain.Commitment: {},
			schedulingdomain.Actual:     {},
		},
	}
}
