package domain

import (
	"cmp"
	"time"
)

// DailyPlanTask contains only the authoritative keys used to order one Sprint
// Task row. A nil DailyPlanOrderDate means no readable positive canonical
// Execution allocation is available.
type DailyPlanTask struct {
	TaskID             string
	Completed          bool
	DailyPlanOrderDate *time.Time
	ProjectPriority    int
	WBSRank            int
}

func CompareDailyPlanTask(left, right DailyPlanTask) int {
	leftGroup := dailyPlanGroup(left)
	rightGroup := dailyPlanGroup(right)
	if result := cmp.Compare(leftGroup, rightGroup); result != 0 {
		return result
	}
	if left.DailyPlanOrderDate != nil && right.DailyPlanOrderDate != nil {
		if result := left.DailyPlanOrderDate.Compare(*right.DailyPlanOrderDate); result != 0 {
			return result
		}
	}
	if result := cmp.Compare(left.ProjectPriority, right.ProjectPriority); result != 0 {
		return result
	}
	if result := cmp.Compare(left.WBSRank, right.WBSRank); result != 0 {
		return result
	}
	return cmp.Compare(left.TaskID, right.TaskID)
}

func dailyPlanGroup(task DailyPlanTask) int {
	if task.DailyPlanOrderDate == nil {
		return 2
	}
	if task.Completed {
		return 1
	}
	return 0
}
