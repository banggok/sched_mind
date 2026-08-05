package domain

import (
	"slices"
	"testing"
	"time"
)

func TestCompareDailyPlanTaskOrdersCanonicalDailyWorkingPlan_SPD01_AC34And78(t *testing.T) {
	day1 := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	day2 := day1.AddDate(0, 0, 1)
	tasks := []DailyPlanTask{
		{TaskID: "unreadable", ProjectPriority: 1, WBSRank: 1},
		{TaskID: "completed", Completed: true, DailyPlanOrderDate: &day1, ProjectPriority: 1, WBSRank: 1},
		{TaskID: "later", DailyPlanOrderDate: &day2, ProjectPriority: 1, WBSRank: 1},
		{TaskID: "same-date-lower-priority", DailyPlanOrderDate: &day1, ProjectPriority: 2, WBSRank: 1},
		{TaskID: "same-date-later-wbs", DailyPlanOrderDate: &day1, ProjectPriority: 1, WBSRank: 2},
		{TaskID: "same-date-first", DailyPlanOrderDate: &day1, ProjectPriority: 1, WBSRank: 1},
	}

	slices.SortStableFunc(tasks, CompareDailyPlanTask)
	got := make([]string, 0, len(tasks))
	for _, task := range tasks {
		got = append(got, task.TaskID)
	}
	want := []string{"same-date-first", "same-date-later-wbs", "same-date-lower-priority", "later", "completed", "unreadable"}
	if !slices.Equal(got, want) {
		t.Fatalf("unexpected daily working-plan order: got %v want %v", got, want)
	}
}
