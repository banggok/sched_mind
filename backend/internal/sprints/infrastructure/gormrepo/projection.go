package gormrepo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/sprints/application"
	"github.com/banggok/sched_mind/backend/internal/sprints/domain"
)

type memberProjectionRow struct {
	ID               string
	Name             string
	RoleName         string
	DailyCapacity    string
	BufferPercentage string
	UpdatedAt        time.Time
}

type taskProjectionRow struct {
	ID              string
	ProjectID       string
	ProjectName     string
	ProjectStatus   string
	ProjectPriority int
	ScheduleVersion int64
	ParentKey       string
	Position        int
	Name            string
	AssigneeID      *string
	AssigneeName    *string
	EffortMinutes   *int
	ExecutionStart  *time.Time
	ExecutionEnd    *time.Time
	CommitmentStart *time.Time
	CommitmentEnd   *time.Time
	ActualStart     *time.Time
	ActualEnd       *time.Time
	UpdatedAt       time.Time
}

type allocationProjectionRow struct {
	TaskID           string
	AssigneeID       string
	AllocationDate   time.Time
	AllocatedMinutes string
}

type overrideProjectionRow struct {
	TeamMemberID string
	StartDate    time.Time
	EndDate      time.Time
	Capacity     string
	UpdatedAt    time.Time
}

type holidayProjectionRow struct {
	Date time.Time
}

func (repository *Repository) Detail(ctx context.Context, id string) (*application.Detail, error) {
	sprint, err := repository.Find(ctx, id)
	if err != nil {
		return nil, err
	}
	detail := &application.Detail{Sprint: application.SprintSummary{
		ID: sprint.ID, Name: sprint.Name, StartDate: sprint.StartDate, EndDate: sprint.EndDate,
		Status: string(sprint.Status), Version: sprint.Version, StartedAt: sprint.StartedAt,
	}}

	var memberRows []memberProjectionRow
	if err := repository.database.WithContext(ctx).Table("sprint_members AS relation").
		Select("member.id, member.name, role.name AS role_name, member.daily_capacity, member.buffer_percentage, member.updated_at").
		Joins("JOIN team_members AS member ON member.id = relation.member_id AND member.deleted_at IS NULL").
		Joins("JOIN roles AS role ON role.id = member.role_id").
		Where("relation.sprint_id = ?", id).Order("LOWER(member.name) ASC").Order("member.id ASC").Scan(&memberRows).Error; err != nil {
		return nil, fmt.Errorf("query sprint member projection: %w", err)
	}

	var taskRows []taskProjectionRow
	if err := repository.database.WithContext(ctx).Table("sprint_tasks AS relation").
		Select("task.id, task.project_id, project.name AS project_name, project.status AS project_status, project.priority AS project_priority, project.schedule_version, task.parent_key, task.position, task.name, task.assignee_id, member.name AS assignee_name, task.effort_minutes, task.execution_start, task.execution_end, task.commitment_start, task.commitment_end, task.actual_start, task.actual_end, task.updated_at").
		Joins("JOIN wbs_nodes AS task ON task.id = relation.task_id").
		Joins("JOIN projects AS project ON project.id = task.project_id").
		Joins("LEFT JOIN team_members AS member ON member.id = task.assignee_id AND member.deleted_at IS NULL").
		Where("relation.sprint_id = ?", id).
		Order("project.priority ASC").Order("task.parent_key ASC").Order("task.position ASC").Order("task.id ASC").Scan(&taskRows).Error; err != nil {
		return nil, fmt.Errorf("query sprint task projection: %w", err)
	}
	wbsContext, wbsRows, err := repository.loadWBSContext(ctx, taskRows)
	if err != nil {
		return nil, err
	}
	taskIDs := make([]string, 0, len(taskRows))
	for _, row := range taskRows {
		taskIDs = append(taskIDs, row.ID)
	}
	var allocationRows []allocationProjectionRow
	if len(taskIDs) > 0 {
		if err := repository.database.WithContext(ctx).Table("task_schedule_allocations").
			Select("task_id, assignee_id, allocation_date, allocated_minutes").
			Where("timeline = ? AND task_id IN ?", "execution", taskIDs).
			Order("task_id ASC").Order("allocation_date ASC").Scan(&allocationRows).Error; err != nil {
			return nil, fmt.Errorf("query sprint execution allocations: %w", err)
		}
	}

	memberIDs := make([]string, 0, len(memberRows))
	for _, row := range memberRows {
		memberIDs = append(memberIDs, row.ID)
	}
	var overrideRows []overrideProjectionRow
	if len(memberIDs) > 0 {
		if err := repository.database.WithContext(ctx).Table("capacity_overrides").
			Select("team_member_id, start_date, end_date, capacity, updated_at").
			Where("deleted_at IS NULL AND team_member_id IN ?", memberIDs).
			Where("start_date <= ? AND end_date >= ?", sprint.EndDate, sprint.StartDate).
			Order("team_member_id ASC").Order("start_date ASC").Order("end_date ASC").Scan(&overrideRows).Error; err != nil {
			return nil, fmt.Errorf("query sprint capacity overrides: %w", err)
		}
	}
	var holidayRows []holidayProjectionRow
	if err := repository.database.WithContext(ctx).Table("public_holiday_dates").Select("date").
		Where("date BETWEEN ? AND ?", sprint.StartDate, sprint.EndDate).Order("date ASC").Scan(&holidayRows).Error; err != nil {
		return nil, fmt.Errorf("query sprint holidays: %w", err)
	}

	allocations, allocationAssignees, _ := composeAllocations(allocationRows)
	selectedMembers := make(map[string]struct{}, len(memberIDs))
	memberIndex := make(map[string]int, len(memberIDs))
	selectedAllocationByMember := make(map[string]map[string]int64, len(memberIDs))
	for index, row := range memberRows {
		selectedMembers[row.ID] = struct{}{}
		daily, total, capacityErr := composeCapacity(row, sprint.StartDate, sprint.EndDate, overrideRows, holidayRows)
		if capacityErr != nil {
			return nil, capacityErr
		}
		detail.Members = append(detail.Members, application.MemberProjection{ID: row.ID, Name: row.Name, RoleName: row.RoleName, DailyCapacity: daily, CapacityMinutes: total})
		memberIndex[row.ID] = index
		selectedAllocationByMember[row.ID] = make(map[string]int64)
	}

	needsReviewByDate := make(map[string]int64)
	for _, row := range taskRows {
		task, readable := composeTaskProjection(row, allocations[row.ID], allocationAssignees[row.ID], wbsContext[row.ID], sprint.StartDate, sprint.EndDate)
		task.Warnings = driftWarnings(row, selectedMembers, readable)
		needsReview := len(task.Warnings) > 0
		if row.AssigneeID != nil {
			if index, selected := memberIndex[*row.AssigneeID]; selected && !needsReview {
				addDailyAllocation(selectedAllocationByMember[*row.AssigneeID], task.Allocations, sprint.StartDate, sprint.EndDate)
				detail.Members[index].TotalAllocationMinutes += task.TotalAllocationMinutes
			}
		}
		if needsReview {
			detail.Totals.NeedsReviewAllocationMinutes += task.InSprintAllocationMinutes
			if readable {
				addDailyAllocation(needsReviewByDate, task.Allocations, sprint.StartDate, sprint.EndDate)
			}
		}
		detail.Totals.AllTaskInSprintMinutes += task.InSprintAllocationMinutes
		detail.Totals.AllTaskTotalMinutes += task.TotalAllocationMinutes
		detail.Tasks = append(detail.Tasks, task)
	}
	sortDailyPlanTasks(detail.Tasks)
	for index := range detail.Members {
		member := &detail.Members[index]
		finalizeMemberProjection(member, selectedAllocationByMember[member.ID])
		detail.Totals.CapacityMinutes += member.CapacityMinutes
		detail.Totals.SelectedMemberAllocationMinutes += member.InSprintAllocationMinutes
		detail.Totals.RemainingMinutes += member.RemainingMinutes
		detail.Totals.OvercapacityMinutes += member.OvercapacityMinutes
	}
	detail.Totals.DailySummaries = composeSprintDailyTotals(detail.Members)
	detail.Totals.NeedsReviewDailyAllocation = dailyValues(needsReviewByDate)
	detail.ProjectionToken = projectionToken(*sprint, memberRows, taskRows, wbsRows, overrideRows, allocationRows, holidayRows)
	return detail, nil
}

func composeCapacity(member memberProjectionRow, startDate, endDate time.Time, overrides []overrideProjectionRow, holidays []holidayProjectionRow) ([]application.DailyValue, int64, error) {
	holidaySet := make(map[string]struct{}, len(holidays))
	for _, holiday := range holidays {
		holidaySet[schedulingdomain.DateKey(holiday.Date)] = struct{}{}
	}
	daily := make([]application.DailyValue, 0)
	total := int64(0)
	for date := schedulingdomain.DateOnly(startDate); !date.After(schedulingdomain.DateOnly(endDate)); date = date.AddDate(0, 0, 1) {
		minutes := int64(0)
		if !schedulingdomain.IsWeekend(date) {
			if _, holiday := holidaySet[schedulingdomain.DateKey(date)]; !holiday {
				resolved, err := schedulingdomain.ParseDecimal(member.DailyCapacity)
				if err != nil {
					return nil, 0, fmt.Errorf("parse member capacity: %w", err)
				}
				for _, override := range overrides {
					if override.TeamMemberID != member.ID || date.Before(schedulingdomain.DateOnly(override.StartDate)) || date.After(schedulingdomain.DateOnly(override.EndDate)) {
						continue
					}
					value, parseErr := schedulingdomain.ParseDecimal(override.Capacity)
					if parseErr != nil {
						return nil, 0, fmt.Errorf("parse capacity override: %w", parseErr)
					}
					if value.Cmp(resolved) < 0 {
						resolved = value
					}
				}
				value, capacityErr := schedulingdomain.CapacityMinutes(resolved, member.BufferPercentage, 0, schedulingdomain.Execution)
				if capacityErr != nil {
					return nil, 0, fmt.Errorf("resolve sprint execution capacity: %w", capacityErr)
				}
				minutes = value.Num().Int64() / value.Denom().Int64()
			}
		}
		daily = append(daily, application.DailyValue{Date: date, Minutes: minutes})
		total += minutes
	}
	return daily, total, nil
}

func composeAllocations(rows []allocationProjectionRow) (map[string][]application.DailyValue, map[string]map[string]struct{}, map[string]struct{}) {
	values := make(map[string][]application.DailyValue)
	assignees := make(map[string]map[string]struct{})
	unreadableTasks := make(map[string]struct{})
	for _, row := range rows {
		minutes, ok := new(big.Rat).SetString(row.AllocatedMinutes)
		if !ok || minutes.Sign() <= 0 || !minutes.IsInt() {
			unreadableTasks[row.TaskID] = struct{}{}
			continue
		}
		values[row.TaskID] = append(values[row.TaskID], application.DailyValue{Date: schedulingdomain.DateOnly(row.AllocationDate), Minutes: minutes.Num().Int64()})
		if assignees[row.TaskID] == nil {
			assignees[row.TaskID] = make(map[string]struct{})
		}
		assignees[row.TaskID][row.AssigneeID] = struct{}{}
	}
	for taskID := range unreadableTasks {
		delete(values, taskID)
		delete(assignees, taskID)
	}
	return values, assignees, unreadableTasks
}

func driftWarnings(task taskProjectionRow, selected map[string]struct{}, allocationReadable bool) []string {
	warnings := make([]string, 0, 2)
	if task.AssigneeID == nil || *task.AssigneeID == "" {
		warnings = append(warnings, "Task no longer has an Assignee.")
	} else if _, exists := selected[*task.AssigneeID]; !exists {
		warnings = append(warnings, "Assignee is not included in this Sprint.")
	}
	if task.ExecutionStart == nil || task.ExecutionEnd == nil {
		warnings = append(warnings, "Task is no longer scheduled.")
	}
	if task.ProjectStatus == "closed" {
		warnings = append(warnings, "Project is Closed.")
	}
	if task.AssigneeID != nil && task.ExecutionStart != nil && task.ExecutionEnd != nil && !allocationReadable {
		warnings = append(warnings, "Execution allocation could not be resolved.")
	}
	return warnings
}

func projectionToken(
	sprint domain.Sprint,
	members []memberProjectionRow,
	tasks []taskProjectionRow,
	wbsRows []wbsContextRow,
	overrides []overrideProjectionRow,
	allocations []allocationProjectionRow,
	holidays []holidayProjectionRow,
) string {
	parts := []string{fmt.Sprintf(
		"sprint:%s:%d:%d:%s:%s:%s",
		sprint.ID,
		sprint.Version,
		sprint.UpdatedAt.UnixNano(),
		schedulingdomain.DateKey(sprint.StartDate),
		schedulingdomain.DateKey(sprint.EndDate),
		sprint.Status,
	)}
	for _, row := range members {
		parts = append(parts, fmt.Sprintf(
			"member:%s:%s:%s:%s:%s:%d",
			row.ID,
			row.Name,
			row.RoleName,
			row.DailyCapacity,
			row.BufferPercentage,
			row.UpdatedAt.UnixNano(),
		))
	}
	for _, row := range tasks {
		parts = append(parts, fmt.Sprintf(
			"task:%s:%s:%s:%s:%d:%d:%s:%d:%s:%s:%s:%s:%s:%s:%s:%s:%s:%s:%d",
			row.ID,
			row.ProjectID,
			row.ProjectName,
			row.ProjectStatus,
			row.ProjectPriority,
			row.ScheduleVersion,
			row.ParentKey,
			row.Position,
			row.Name,
			optionalString(row.AssigneeID),
			optionalString(row.AssigneeName),
			optionalInt(row.EffortMinutes),
			optionalDate(row.ExecutionStart),
			optionalDate(row.ExecutionEnd),
			optionalDate(row.CommitmentStart),
			optionalDate(row.CommitmentEnd),
			optionalDate(row.ActualStart),
			optionalDate(row.ActualEnd),
			row.UpdatedAt.UnixNano(),
		))
	}
	for _, row := range wbsRows {
		parts = append(parts, fmt.Sprintf(
			"wbs:%s:%s:%s:%d:%s",
			row.ID,
			row.ProjectID,
			optionalString(row.ParentID),
			row.Position,
			row.Name,
		))
	}
	for _, row := range overrides {
		parts = append(parts, fmt.Sprintf(
			"override:%s:%s:%s:%s:%d",
			row.TeamMemberID,
			schedulingdomain.DateKey(row.StartDate),
			schedulingdomain.DateKey(row.EndDate),
			row.Capacity,
			row.UpdatedAt.UnixNano(),
		))
	}
	for _, row := range allocations {
		parts = append(parts, fmt.Sprintf(
			"allocation:%s:%s:%s:%s",
			row.TaskID,
			row.AssigneeID,
			schedulingdomain.DateKey(row.AllocationDate),
			row.AllocatedMinutes,
		))
	}
	for _, row := range holidays {
		parts = append(parts, "holiday:"+schedulingdomain.DateKey(row.Date))
	}
	sort.Strings(parts[1:])
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func optionalString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func optionalInt(value *int) string {
	if value == nil {
		return ""
	}
	return fmt.Sprintf("%d", *value)
}

func optionalDate(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
