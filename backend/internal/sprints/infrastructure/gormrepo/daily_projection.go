package gormrepo

import (
	"sort"
	"time"

	"github.com/banggok/sched_mind/backend/internal/sprints/application"
	"github.com/banggok/sched_mind/backend/internal/sprints/domain"
)

func composeTaskProjection(
	row taskProjectionRow,
	allocations []application.DailyValue,
	allocationAssignees map[string]struct{},
	wbs wbsContextValue,
	startDate time.Time,
	endDate time.Time,
) (application.TaskProjection, bool) {
	readable := hasReadableAllocation(row, allocations, allocationAssignees)
	task := application.TaskProjection{
		ID:              row.ID,
		ProjectID:       row.ProjectID,
		ProjectName:     row.ProjectName,
		ProjectStatus:   row.ProjectStatus,
		ProjectPriority: row.ProjectPriority,
		Name:            row.Name,
		WBSOrder:        wbs.Path,
		WBSPath:         wbs.Path,
		WBSRank:         wbs.Rank,
		AssigneeID:      row.AssigneeID,
		AssigneeName:    row.AssigneeName,
		ExecutionStart:  row.ExecutionStart,
		ExecutionEnd:    row.ExecutionEnd,
		Completed:       row.ActualStart != nil && row.ActualEnd != nil,
	}
	if readable {
		task.Allocations = append([]application.DailyValue(nil), allocations...)
		for _, allocation := range task.Allocations {
			task.TotalAllocationMinutes += allocation.Minutes
			if task.DailyPlanOrderDate == nil || allocation.Date.Before(*task.DailyPlanOrderDate) {
				value := allocation.Date
				task.DailyPlanOrderDate = &value
			}
			if !allocation.Date.Before(startDate) && !allocation.Date.After(endDate) {
				task.InSprintAllocationMinutes += allocation.Minutes
			} else {
				task.OutsideAllocationMinutes += allocation.Minutes
			}
		}
	}
	return task, readable
}

func hasReadableAllocation(row taskProjectionRow, allocations []application.DailyValue, allocationAssignees map[string]struct{}) bool {
	if row.AssigneeID == nil || row.ExecutionStart == nil || row.ExecutionEnd == nil || len(allocations) == 0 || len(allocationAssignees) != 1 {
		return false
	}
	_, matches := allocationAssignees[*row.AssigneeID]
	return matches
}

func sortDailyPlanTasks(tasks []application.TaskProjection) {
	sort.SliceStable(tasks, func(left, right int) bool {
		return domain.CompareDailyPlanTask(
			domain.DailyPlanTask{
				TaskID:             tasks[left].ID,
				Completed:          tasks[left].Completed,
				DailyPlanOrderDate: tasks[left].DailyPlanOrderDate,
				ProjectPriority:    tasks[left].ProjectPriority,
				WBSRank:            tasks[left].WBSRank,
			},
			domain.DailyPlanTask{
				TaskID:             tasks[right].ID,
				Completed:          tasks[right].Completed,
				DailyPlanOrderDate: tasks[right].DailyPlanOrderDate,
				ProjectPriority:    tasks[right].ProjectPriority,
				WBSRank:            tasks[right].WBSRank,
			},
		) < 0
	})
}

func addDailyAllocation(values map[string]int64, allocations []application.DailyValue, startDate, endDate time.Time) {
	for _, allocation := range allocations {
		if allocation.Date.Before(startDate) || allocation.Date.After(endDate) {
			continue
		}
		values[allocation.Date.Format("2006-01-02")] += allocation.Minutes
	}
}

func finalizeMemberProjection(member *application.MemberProjection, selectedByDate map[string]int64) {
	member.DailySummaries = make([]application.DailySummary, 0, len(member.DailyCapacity))
	member.CapacityMinutes = 0
	member.InSprintAllocationMinutes = 0
	member.RemainingMinutes = 0
	member.OvercapacityMinutes = 0
	for _, capacity := range member.DailyCapacity {
		allocation := selectedByDate[capacity.Date.Format("2006-01-02")]
		remaining := int64(0)
		overcapacity := int64(0)
		if allocation > capacity.Minutes {
			overcapacity = allocation - capacity.Minutes
		} else {
			remaining = capacity.Minutes - allocation
		}
		member.DailySummaries = append(member.DailySummaries, application.DailySummary{
			Date:                      capacity.Date,
			CapacityMinutes:           capacity.Minutes,
			SelectedAllocationMinutes: allocation,
			RemainingMinutes:          remaining,
			OvercapacityMinutes:       overcapacity,
		})
		member.CapacityMinutes += capacity.Minutes
		member.InSprintAllocationMinutes += allocation
		member.RemainingMinutes += remaining
		member.OvercapacityMinutes += overcapacity
	}
}

func composeSprintDailyTotals(members []application.MemberProjection) []application.DailySummary {
	byDate := make(map[string]application.DailySummary)
	order := make([]string, 0)
	for _, member := range members {
		for _, daily := range member.DailySummaries {
			key := daily.Date.Format("2006-01-02")
			value, exists := byDate[key]
			if !exists {
				value.Date = daily.Date
				order = append(order, key)
			}
			value.CapacityMinutes += daily.CapacityMinutes
			value.SelectedAllocationMinutes += daily.SelectedAllocationMinutes
			value.RemainingMinutes += daily.RemainingMinutes
			value.OvercapacityMinutes += daily.OvercapacityMinutes
			byDate[key] = value
		}
	}
	sort.Strings(order)
	result := make([]application.DailySummary, 0, len(order))
	for _, key := range order {
		result = append(result, byDate[key])
	}
	return result
}

func dailyValues(values map[string]int64) []application.DailyValue {
	keys := make([]string, 0, len(values))
	for key, minutes := range values {
		if minutes > 0 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	result := make([]application.DailyValue, 0, len(keys))
	for _, key := range keys {
		date, _ := time.Parse("2006-01-02", key)
		result = append(result, application.DailyValue{Date: date, Minutes: values[key]})
	}
	return result
}
