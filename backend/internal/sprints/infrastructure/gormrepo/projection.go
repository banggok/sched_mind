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
	ExecutionStart  *time.Time
	ExecutionEnd    *time.Time
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
		Select("task.id, task.project_id, project.name AS project_name, project.status AS project_status, project.priority AS project_priority, project.schedule_version, task.parent_key, task.position, task.name, task.assignee_id, member.name AS assignee_name, task.execution_start, task.execution_end, task.actual_start, task.actual_end, task.updated_at").
		Joins("JOIN wbs_nodes AS task ON task.id = relation.task_id").
		Joins("JOIN projects AS project ON project.id = task.project_id").
		Joins("LEFT JOIN team_members AS member ON member.id = task.assignee_id AND member.deleted_at IS NULL").
		Where("relation.sprint_id = ?", id).
		Order("project.priority ASC").Order("task.parent_key ASC").Order("task.position ASC").Order("task.id ASC").Scan(&taskRows).Error; err != nil {
		return nil, fmt.Errorf("query sprint task projection: %w", err)
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

	allocations, allocationAssignees, err := composeAllocations(allocationRows)
	if err != nil {
		return nil, err
	}
	selectedMembers := make(map[string]struct{}, len(memberIDs))
	memberIndex := make(map[string]int, len(memberIDs))
	for index, row := range memberRows {
		selectedMembers[row.ID] = struct{}{}
		daily, total, capacityErr := composeCapacity(row, sprint.StartDate, sprint.EndDate, overrideRows, holidayRows)
		if capacityErr != nil {
			return nil, capacityErr
		}
		detail.Members = append(detail.Members, application.MemberProjection{ID: row.ID, Name: row.Name, RoleName: row.RoleName, DailyCapacity: daily, CapacityMinutes: total})
		memberIndex[row.ID] = index
	}

	for _, row := range taskRows {
		task := application.TaskProjection{ID: row.ID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, ProjectStatus: row.ProjectStatus,
			Name: row.Name, WBSOrder: fmt.Sprintf("%s:%08d", row.ParentKey, row.Position), AssigneeID: row.AssigneeID, AssigneeName: row.AssigneeName,
			ExecutionStart: row.ExecutionStart, ExecutionEnd: row.ExecutionEnd, Completed: row.ActualStart != nil && row.ActualEnd != nil,
			Allocations: allocations[row.ID]}
		for _, allocation := range task.Allocations {
			task.TotalAllocationMinutes += allocation.Minutes
			if !allocation.Date.Before(sprint.StartDate) && !allocation.Date.After(sprint.EndDate) {
				task.InSprintAllocationMinutes += allocation.Minutes
			} else {
				task.OutsideAllocationMinutes += allocation.Minutes
			}
		}
		task.Warnings = driftWarnings(row, selectedMembers, allocationAssignees[row.ID])
		needsReview := len(task.Warnings) > 0
		if row.AssigneeID != nil {
			if index, selected := memberIndex[*row.AssigneeID]; selected && !needsReview {
				detail.Members[index].InSprintAllocationMinutes += task.InSprintAllocationMinutes
			}
		}
		if needsReview {
			detail.Totals.NeedsReviewAllocationMinutes += task.InSprintAllocationMinutes
		}
		detail.Totals.AllTaskInSprintMinutes += task.InSprintAllocationMinutes
		detail.Totals.AllTaskTotalMinutes += task.TotalAllocationMinutes
		detail.Tasks = append(detail.Tasks, task)
	}
	for index := range detail.Members {
		member := &detail.Members[index]
		detail.Totals.CapacityMinutes += member.CapacityMinutes
		detail.Totals.SelectedMemberAllocationMinutes += member.InSprintAllocationMinutes
		if member.InSprintAllocationMinutes > member.CapacityMinutes {
			member.OvercapacityMinutes = member.InSprintAllocationMinutes - member.CapacityMinutes
		} else {
			member.RemainingMinutes = member.CapacityMinutes - member.InSprintAllocationMinutes
		}
	}
	detail.ProjectionToken = projectionToken(*sprint, memberRows, taskRows, overrideRows, allocationRows, holidayRows)
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

func composeAllocations(rows []allocationProjectionRow) (map[string][]application.DailyValue, map[string]map[string]struct{}, error) {
	values := make(map[string][]application.DailyValue)
	assignees := make(map[string]map[string]struct{})
	for _, row := range rows {
		minutes, ok := new(big.Rat).SetString(row.AllocatedMinutes)
		if !ok || minutes.Sign() <= 0 || !minutes.IsInt() {
			return nil, nil, fmt.Errorf("compose sprint allocation: %w", schedulingdomain.ErrDataIntegrity)
		}
		values[row.TaskID] = append(values[row.TaskID], application.DailyValue{Date: schedulingdomain.DateOnly(row.AllocationDate), Minutes: minutes.Num().Int64()})
		if assignees[row.TaskID] == nil {
			assignees[row.TaskID] = make(map[string]struct{})
		}
		assignees[row.TaskID][row.AssigneeID] = struct{}{}
	}
	return values, assignees, nil
}

func driftWarnings(task taskProjectionRow, selected map[string]struct{}, allocationAssignees map[string]struct{}) []string {
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
	if task.AssigneeID != nil && len(allocationAssignees) > 0 {
		if _, consistent := allocationAssignees[*task.AssigneeID]; !consistent || len(allocationAssignees) != 1 {
			warnings = append(warnings, "Execution allocation could not be resolved.")
		}
	}
	return warnings
}

func projectionToken(sprint domain.Sprint, members []memberProjectionRow, tasks []taskProjectionRow, overrides []overrideProjectionRow, allocations []allocationProjectionRow, holidays []holidayProjectionRow) string {
	parts := []string{fmt.Sprintf("sprint:%s:%d:%d", sprint.ID, sprint.Version, sprint.UpdatedAt.UnixNano())}
	for _, row := range members {
		parts = append(parts, fmt.Sprintf("member:%s:%d", row.ID, row.UpdatedAt.UnixNano()))
	}
	for _, row := range tasks {
		parts = append(parts, fmt.Sprintf("task:%s:%d:%d", row.ID, row.UpdatedAt.UnixNano(), row.ScheduleVersion))
	}
	for _, row := range overrides {
		parts = append(parts, fmt.Sprintf("override:%s:%d", row.TeamMemberID, row.UpdatedAt.UnixNano()))
	}
	for _, row := range allocations {
		parts = append(parts, fmt.Sprintf("allocation:%s:%s:%s:%s", row.TaskID, row.AssigneeID, schedulingdomain.DateKey(row.AllocationDate), row.AllocatedMinutes))
	}
	for _, row := range holidays {
		parts = append(parts, "holiday:"+schedulingdomain.DateKey(row.Date))
	}
	sort.Strings(parts[1:])
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}
