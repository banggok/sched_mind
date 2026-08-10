package gormrepo

import (
	"math/big"
	"testing"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
)

func TestAllocationConsumesCapacityForManualReservationByProjectPriority_US63_AC19(t *testing.T) {
	automaticTask := schedulableTask("automatic-task", "automatic", 1, "member", 960, 0)
	manualTask := schedulableTask("manual-task", "manual", 1, "member", 480, 0)
	manualAllocation := dailyAllocation{
		TaskID:   manualTask.ID,
		MemberID: "member",
		Date:     time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		Minutes:  big.NewRat(240, 1),
		Fixed:    true,
	}

	tests := []struct {
		name              string
		automaticPriority int
		manualPriority    int
		wantConsumes      bool
	}{
		{
			name:              "lower-priority manual allocation does not consume higher-priority automatic capacity",
			automaticPriority: 1,
			manualPriority:    2,
			wantConsumes:      false,
		},
		{
			name:              "higher-priority manual allocation consumes lower-priority automatic capacity",
			automaticPriority: 2,
			manualPriority:    1,
			wantConsumes:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			state := &portfolioState{
				projects: map[string]projectModel{
					"automatic": {ID: "automatic", Status: "open", Priority: tc.automaticPriority, AutomaticScheduling: true},
					"manual":    {ID: "manual", Status: "open", Priority: tc.manualPriority, AutomaticScheduling: false},
				},
				tasks: map[string]taskModel{
					automaticTask.ID: automaticTask,
					manualTask.ID:    manualTask,
				},
			}
			calendar := newAllocationCalendar(state, schedulingdomain.Execution)
			if got := calendar.allocationConsumesCapacityFor(automaticTask, manualAllocation); got != tc.wantConsumes {
				t.Fatalf("allocationConsumesCapacityFor() = %v, want %v", got, tc.wantConsumes)
			}
		})
	}
}

func TestAllocationConsumesCapacityForResolvedManualGroupUsesEffectiveMode_US44_AC10_US63_AC19(t *testing.T) {
	candidate := schedulableTask("automatic-task", "high", 1, "member", 960, 0)
	reserved := schedulableTask("manual-group-task", "low", 1, "member", 480, 0)
	reserved.EffectiveLifecycle = "open"
	reserved.EffectiveAutomaticScheduling = false
	allocation := dailyAllocation{
		TaskID: reserved.ID, MemberID: "member",
		Date:    time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC),
		Minutes: big.NewRat(240, 1), Fixed: true,
	}
	state := &portfolioState{
		projects: map[string]projectModel{
			"high": {ID: "high", Status: "open", Priority: 1, AutomaticScheduling: true},
			// Raw Project mode is intentionally ON: the Task is manual only because
			// its resolved Group configuration overrides Automatic Scheduling OFF.
			"low": {ID: "low", Status: "open", Priority: 2, AutomaticScheduling: true},
		},
		tasks: map[string]taskModel{candidate.ID: candidate, reserved.ID: reserved},
	}
	calendar := newAllocationCalendar(state, schedulingdomain.Execution)
	if calendar.allocationConsumesCapacityFor(candidate, allocation) {
		t.Fatal("lower-priority resolved manual Group allocation consumed higher-priority automatic capacity")
	}
}

func TestScheduleTimelineKeepsFixedDatesWithoutAllocationPrerequisites_US62_AC22_US44_AC24(t *testing.T) {
	start := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	memberID := "member"
	effort := 480

	tests := []struct {
		name       string
		assigneeID *string
		effort     *int
	}{
		{name: "missing assignee", effort: &effort},
		{name: "missing effort", assigneeID: &memberID},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			task := taskModel{
				ID:                           "fixed-task",
				ProjectID:                    "manual",
				AssigneeID:                   tc.assigneeID,
				EffortMinutes:                tc.effort,
				ExecutionStart:               &start,
				ExecutionEnd:                 &end,
				CommitmentStart:              &start,
				CommitmentEnd:                &end,
				EffectiveLifecycle:           "open",
				EffectiveAutomaticScheduling: false,
			}
			state := &portfolioState{
				projects:       map[string]projectModel{"manual": {ID: "manual", Priority: 1, Status: "open", AutomaticScheduling: false}},
				tasks:          map[string]taskModel{task.ID: task},
				leafOrder:      map[string]int{task.ID: 0},
				members:        map[string]memberModel{},
				overrides:      map[string][]capacityOverrideModel{},
				holidays:       map[string]struct{}{},
				fixed:          map[schedulingdomain.Timeline][]taskModel{schedulingdomain.Execution: {task}, schedulingdomain.Commitment: {task}},
				manualBlockers: map[string][]string{},
				fixedBlockers:  map[string][]string{},
				existingAllocations: map[schedulingdomain.Timeline]map[string][]allocationModel{
					schedulingdomain.Execution:  {},
					schedulingdomain.Commitment: {},
					schedulingdomain.Actual:     {},
				},
			}

			result, err := state.scheduleTimeline(schedulingdomain.Execution)
			if err != nil {
				t.Fatalf("schedule fixed timeline: %v", err)
			}
			schedule, exists := result.schedules[task.ID]
			if !exists {
				t.Fatal("fixed Task timeline missing from scheduling result")
			}
			if schedule.Start == nil || !schedule.Start.Equal(start) || schedule.End == nil || !schedule.End.Equal(end) {
				t.Fatalf("fixed Task timeline=%#v, want %s..%s", schedule, start, end)
			}
			if len(schedule.Allocations) != 0 {
				t.Fatalf("fixed Task allocations=%#v, want none without complete allocation prerequisites", schedule.Allocations)
			}
		})
	}
}
