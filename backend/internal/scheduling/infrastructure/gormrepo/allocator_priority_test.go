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
