package gormrepo

import (
	"context"
	"fmt"
	"sort"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/sprints/application"
	"github.com/banggok/sched_mind/backend/internal/sprints/domain"
)

func (repository *Repository) Suggest(ctx context.Context, input application.SuggestionInput) (*application.Suggestion, error) {
	if err := validateMembers(repository.database.WithContext(ctx), input.MemberIDs); err != nil {
		return nil, err
	}
	var memberRows []memberProjectionRow
	if err := repository.database.WithContext(ctx).Table("team_members AS member").
		Select("member.id, member.name, role.name AS role_name, member.daily_capacity, member.buffer_percentage, member.updated_at").
		Joins("JOIN roles AS role ON role.id = member.role_id").
		Where("member.deleted_at IS NULL AND member.id IN ?", input.MemberIDs).
		Order("LOWER(member.name) ASC").Order("member.id ASC").Scan(&memberRows).Error; err != nil {
		return nil, fmt.Errorf("query suggestion members: %w", err)
	}

	var taskRows []taskProjectionRow
	if err := repository.database.WithContext(ctx).Table("wbs_nodes AS task").
		Select("task.id, task.project_id, project.name AS project_name, project.status AS project_status, project.priority AS project_priority, project.schedule_version, task.parent_key, task.position, task.name, task.assignee_id, member.name AS assignee_name, task.execution_start, task.execution_end, task.actual_start, task.actual_end, task.updated_at").
		Joins("JOIN projects AS project ON project.id = task.project_id").
		Joins("JOIN team_members AS member ON member.id = task.assignee_id AND member.deleted_at IS NULL").
		Where("task.assignee_id IN ?", input.MemberIDs).
		Where("task.execution_start IS NOT NULL AND task.execution_end IS NOT NULL").
		Where("task.actual_start IS NULL AND task.actual_end IS NULL").
		Where("project.status IN ?", []string{"open", "locked"}).
		Where("NOT EXISTS (?)", repository.database.Table("wbs_nodes AS child").Select("1").Where("child.project_id = task.project_id AND child.parent_id = task.id")).
		Order("project.priority ASC").Order("task.parent_key ASC").Order("task.position ASC").Order("task.id ASC").Scan(&taskRows).Error; err != nil {
		return nil, fmt.Errorf("query suggestion tasks: %w", err)
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
			return nil, fmt.Errorf("query suggestion allocations: %w", err)
		}
	}
	allocations, allocationAssignees, unreadableTasks := composeAllocations(allocationRows)
	if len(unreadableTasks) > 0 {
		return nil, fmt.Errorf("compose sprint allocation: %w", schedulingdomain.ErrDataIntegrity)
	}

	var overrideRows []overrideProjectionRow
	if err := repository.database.WithContext(ctx).Table("capacity_overrides").
		Select("team_member_id, start_date, end_date, capacity, updated_at").
		Where("deleted_at IS NULL AND team_member_id IN ?", input.MemberIDs).
		Where("start_date <= ? AND end_date >= ?", input.EndDate, input.StartDate).
		Order("team_member_id ASC").Order("start_date ASC").Order("end_date ASC").Scan(&overrideRows).Error; err != nil {
		return nil, fmt.Errorf("query suggestion overrides: %w", err)
	}
	var holidayRows []holidayProjectionRow
	if err := repository.database.WithContext(ctx).Table("public_holiday_dates").Select("date").
		Where("date BETWEEN ? AND ?", input.StartDate, input.EndDate).Order("date ASC").Scan(&holidayRows).Error; err != nil {
		return nil, fmt.Errorf("query suggestion holidays: %w", err)
	}

	suggestion := &application.Suggestion{}
	capacityByMember := make(map[string]int64, len(memberRows))
	memberIndex := make(map[string]int, len(memberRows))
	selectedAllocationByMember := make(map[string]map[string]int64, len(memberRows))
	for index, row := range memberRows {
		daily, total, capacityErr := composeCapacity(row, input.StartDate, input.EndDate, overrideRows, holidayRows)
		if capacityErr != nil {
			return nil, capacityErr
		}
		suggestion.Members = append(suggestion.Members, application.MemberProjection{ID: row.ID, Name: row.Name, RoleName: row.RoleName, DailyCapacity: daily, CapacityMinutes: total})
		capacityByMember[row.ID] = total
		memberIndex[row.ID] = index
		selectedAllocationByMember[row.ID] = make(map[string]int64)
	}
	candidates := make([]domain.Candidate, 0, len(taskRows))
	projections := make(map[string]application.TaskProjection, len(taskRows))
	for _, row := range taskRows {
		projection, readable := composeTaskProjection(row, allocations[row.ID], allocationAssignees[row.ID], wbsContext[row.ID], input.StartDate, input.EndDate)
		if !readable || row.AssigneeID == nil {
			continue
		}
		projections[row.ID] = projection
		candidates = append(candidates, domain.Candidate{TaskID: row.ID, MemberID: *row.AssigneeID, ExecutionEnd: *row.ExecutionEnd,
			ProjectPriority: row.ProjectPriority, WBSOrder: projection.WBSRank, InSprintAllocationMinutes: projection.InSprintAllocationMinutes})
	}
	selected := domain.Suggest(input.MemberIDs, input.EndDate, capacityByMember, candidates)
	for _, value := range selected {
		projection := projections[value.TaskID]
		suggestion.Tasks = append(suggestion.Tasks, application.SuggestedTask{Task: projection, Reason: string(value.Reason)})
		if projection.AssigneeID != nil {
			index := memberIndex[*projection.AssigneeID]
			addDailyAllocation(selectedAllocationByMember[*projection.AssigneeID], projection.Allocations, input.StartDate, input.EndDate)
			suggestion.Members[index].TotalAllocationMinutes += projection.TotalAllocationMinutes
		}
		suggestion.Totals.AllTaskInSprintMinutes += projection.InSprintAllocationMinutes
		suggestion.Totals.AllTaskTotalMinutes += projection.TotalAllocationMinutes
	}
	sort.SliceStable(suggestion.Tasks, func(left, right int) bool {
		return domain.CompareDailyPlanTask(
			domain.DailyPlanTask{TaskID: suggestion.Tasks[left].Task.ID, Completed: suggestion.Tasks[left].Task.Completed, DailyPlanOrderDate: suggestion.Tasks[left].Task.DailyPlanOrderDate, ProjectPriority: suggestion.Tasks[left].Task.ProjectPriority, WBSRank: suggestion.Tasks[left].Task.WBSRank},
			domain.DailyPlanTask{TaskID: suggestion.Tasks[right].Task.ID, Completed: suggestion.Tasks[right].Task.Completed, DailyPlanOrderDate: suggestion.Tasks[right].Task.DailyPlanOrderDate, ProjectPriority: suggestion.Tasks[right].Task.ProjectPriority, WBSRank: suggestion.Tasks[right].Task.WBSRank},
		) < 0
	})
	for index := range suggestion.Members {
		member := &suggestion.Members[index]
		finalizeMemberProjection(member, selectedAllocationByMember[member.ID])
		suggestion.Totals.CapacityMinutes += member.CapacityMinutes
		suggestion.Totals.SelectedMemberAllocationMinutes += member.InSprintAllocationMinutes
		suggestion.Totals.RemainingMinutes += member.RemainingMinutes
		suggestion.Totals.OvercapacityMinutes += member.OvercapacityMinutes
	}
	suggestion.Totals.DailySummaries = composeSprintDailyTotals(suggestion.Members)
	pseudoSprint := domain.Sprint{ID: "suggestion", Version: 0, StartDate: input.StartDate, EndDate: input.EndDate, UpdatedAt: time.Time{}}
	suggestion.ProjectionToken = projectionToken(pseudoSprint, memberRows, taskRows, wbsRows, overrideRows, allocationRows, holidayRows)
	return suggestion, nil
}
