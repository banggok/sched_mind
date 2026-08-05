package gormrepo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"github.com/banggok/sched_mind/backend/internal/sprints/application"
)

func (repository *Repository) Candidates(ctx context.Context, sprintID string, query listing.Query) (listing.Page[application.TaskProjection], error) {
	sprint, err := repository.Find(ctx, sprintID)
	if err != nil {
		return listing.Page[application.TaskProjection]{}, err
	}
	if sprint == nil {
		return listing.Page[application.TaskProjection]{}, application.ErrNotFound
	}
	return repository.candidates(ctx, sprint.StartDate, sprint.EndDate, sprint.MemberIDs, sprint.TaskIDs, query)
}

func (repository *Repository) DraftCandidates(ctx context.Context, input application.CandidateInput, query listing.Query) (listing.Page[application.TaskProjection], error) {
	return repository.candidates(ctx, input.StartDate, input.EndDate, input.MemberIDs, input.ExcludedTaskIDs, query)
}

func (repository *Repository) candidates(ctx context.Context, startDate, endDate time.Time, memberIDs, excludedTaskIDs []string, query listing.Query) (listing.Page[application.TaskProjection], error) {
	statement := repository.database.WithContext(ctx).Table("wbs_nodes AS task").
		Joins("JOIN projects AS project ON project.id = task.project_id").
		Joins("JOIN team_members AS member ON member.id = task.assignee_id AND member.deleted_at IS NULL").
		Where("task.assignee_id IN ?", memberIDs).
		Where("task.execution_start IS NOT NULL AND task.execution_end IS NOT NULL").
		Where("task.actual_start IS NULL AND task.actual_end IS NULL").
		Where("project.status IN ?", []string{"open", "locked"}).
		Where("NOT EXISTS (?)", repository.database.Table("wbs_nodes AS child").Select("1").Where("child.project_id = task.project_id AND child.parent_id = task.id"))
	if len(excludedTaskIDs) > 0 {
		statement = statement.Where("task.id NOT IN ?", excludedTaskIDs)
	}
	if search := strings.ToLower(strings.TrimSpace(query.Search)); search != "" {
		pattern := sharedpersistence.EscapeLike(search) + "%"
		statement = statement.Where("LOWER(task.name) LIKE ? OR LOWER(project.name) LIKE ?", pattern, pattern)
	}
	var total int64
	if err := statement.Distinct("task.id").Count(&total).Error; err != nil {
		return listing.Page[application.TaskProjection]{}, fmt.Errorf("count sprint task candidates: %w", err)
	}
	var rows []taskProjectionRow
	if err := statement.Select("task.id, task.project_id, project.name AS project_name, project.status AS project_status, project.priority AS project_priority, project.schedule_version, task.parent_key, task.position, task.name, task.assignee_id, member.name AS assignee_name, task.execution_start, task.execution_end, task.actual_start, task.actual_end, task.updated_at").
		Order("task.execution_end ASC").Order("project.priority ASC").Order("task.parent_key ASC").Order("task.position ASC").Order("task.id ASC").
		Limit(query.PageSize).Offset((query.Page - 1) * query.PageSize).Scan(&rows).Error; err != nil {
		return listing.Page[application.TaskProjection]{}, fmt.Errorf("query sprint task candidates: %w", err)
	}
	taskIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		taskIDs = append(taskIDs, row.ID)
	}
	var allocationRows []allocationProjectionRow
	if len(taskIDs) > 0 {
		if err := repository.database.WithContext(ctx).Table("task_schedule_allocations").Select("task_id, assignee_id, allocation_date, allocated_minutes").
			Where("timeline = ? AND task_id IN ?", "execution", taskIDs).Order("task_id ASC").Order("allocation_date ASC").Scan(&allocationRows).Error; err != nil {
			return listing.Page[application.TaskProjection]{}, fmt.Errorf("query candidate allocations: %w", err)
		}
	}
	allocations, allocationAssignees, err := composeAllocations(allocationRows)
	if err != nil {
		return listing.Page[application.TaskProjection]{}, err
	}
	items := make([]application.TaskProjection, 0, len(rows))
	for _, row := range rows {
		if row.AssigneeID == nil || len(allocationAssignees[row.ID]) > 1 {
			continue
		}
		if len(allocationAssignees[row.ID]) == 1 {
			if _, matches := allocationAssignees[row.ID][*row.AssigneeID]; !matches {
				continue
			}
		}
		item := application.TaskProjection{ID: row.ID, ProjectID: row.ProjectID, ProjectName: row.ProjectName, ProjectStatus: row.ProjectStatus,
			Name: row.Name, WBSOrder: fmt.Sprintf("%s:%08d", row.ParentKey, row.Position), AssigneeID: row.AssigneeID, AssigneeName: row.AssigneeName,
			ExecutionStart: row.ExecutionStart, ExecutionEnd: row.ExecutionEnd, Allocations: allocations[row.ID]}
		for _, allocation := range item.Allocations {
			item.TotalAllocationMinutes += allocation.Minutes
			if !allocation.Date.Before(startDate) && !allocation.Date.After(endDate) {
				item.InSprintAllocationMinutes += allocation.Minutes
			} else {
				item.OutsideAllocationMinutes += allocation.Minutes
			}
		}
		items = append(items, item)
	}
	return listing.Page[application.TaskProjection]{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}
