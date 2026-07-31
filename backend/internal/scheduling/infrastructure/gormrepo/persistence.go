package gormrepo

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func (repository *Repository) persist(
	database *gorm.DB,
	state *portfolioState,
	execution *timelineResult,
	commitment *timelineResult,
	requestedProjectIDs []string,
) error {
	now := repository.now()
	recalculatedTaskIDs := make([]string, 0)
	dirtyProjects := projectIDsFromRequest(requestedProjectIDs)

	for taskID, task := range state.tasks {
		project := state.projects[task.ProjectID]
		if _, leaf := state.leafOrder[taskID]; !leaf || task.ActualEnd != nil || project.Status != "open" || !project.AutomaticScheduling {
			continue
		}
		executionSchedule, executionExists := execution.schedules[taskID]
		commitmentSchedule, commitmentExists := commitment.schedules[taskID]
		if !executionExists || !commitmentExists {
			return fmt.Errorf("%w: missing generated schedule for task %s", schedulingdomain.ErrDataIntegrity, taskID)
		}

		updates := map[string]any{
			"execution_start":               executionSchedule.Start,
			"execution_end":                 executionSchedule.End,
			"execution_unscheduled_reason":  executionSchedule.Reason,
			"commitment_start":              commitmentSchedule.Start,
			"commitment_end":                commitmentSchedule.End,
			"commitment_unscheduled_reason": commitmentSchedule.Reason,
			"updated_at":                    now,
		}
		result := database.Model(&taskModel{}).Where("id = ? AND actual_end IS NULL", taskID).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("persist generated task schedule: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("%w: task %s changed while scheduling", schedulingdomain.ErrConcurrentConflict, taskID)
		}

		task.ExecutionStart = executionSchedule.Start
		task.ExecutionEnd = executionSchedule.End
		task.ExecutionUnscheduledReason = executionSchedule.Reason
		task.CommitmentStart = commitmentSchedule.Start
		task.CommitmentEnd = commitmentSchedule.End
		task.CommitmentUnscheduledReason = commitmentSchedule.Reason
		task.UpdatedAt = now
		state.tasks[taskID] = task
		recalculatedTaskIDs = append(recalculatedTaskIDs, taskID)
		dirtyProjects[task.ProjectID] = struct{}{}
	}

	if len(recalculatedTaskIDs) > 0 {
		if err := database.Where("task_id IN ?", recalculatedTaskIDs).Delete(&allocationModel{}).Error; err != nil {
			return fmt.Errorf("replace schedule allocations: %w", err)
		}
		if err := persistTimelineAllocations(database, state, execution); err != nil {
			return err
		}
		if err := persistTimelineAllocations(database, state, commitment); err != nil {
			return err
		}
	}

	for projectID := range dirtyProjects {
		project, exists := state.projects[projectID]
		if !exists || project.Status != "open" || !project.AutomaticScheduling {
			continue
		}
		start, end := derivedProjectDates(state, projectID)
		result := database.Model(&projectModel{}).
			Where("id = ? AND schedule_version = ?", projectID, project.ScheduleVersion).
			Updates(map[string]any{
				"start_date":       start,
				"end_date":         end,
				"schedule_version": project.ScheduleVersion + 1,
				"updated_at":       now,
			})
		if result.Error != nil {
			return fmt.Errorf("persist project schedule version: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("%w: project %s schedule version changed", schedulingdomain.ErrConcurrentConflict, projectID)
		}
		project.StartDate = start
		project.EndDate = end
		project.ScheduleVersion++
		state.projects[projectID] = project
	}
	return nil
}

func persistTimelineAllocations(database *gorm.DB, state *portfolioState, result *timelineResult) error {
	allocations := make([]dailyAllocation, 0)
	for _, schedule := range result.schedules {
		allocations = append(allocations, schedule.Allocations...)
	}
	sort.Slice(allocations, func(left, right int) bool { return allocations[left].Sequence < allocations[right].Sequence })

	used := make(map[string]*big.Rat)
	rows := make([]allocationModel, 0, len(allocations))
	for _, allocation := range allocations {
		task, exists := state.tasks[allocation.TaskID]
		if !exists {
			return fmt.Errorf("%w: allocation task %s missing", schedulingdomain.ErrDataIntegrity, allocation.TaskID)
		}
		capacity, err := result.calendar.capacity(allocation.MemberID, task.ProjectID, allocation.Date)
		if err != nil {
			return err
		}
		key := allocation.MemberID + "|" + schedulingdomain.DateKey(allocation.Date)
		consumed := schedulingdomain.CloneRat(used[key])
		remaining := schedulingdomain.CloneRat(capacity)
		remaining.Sub(remaining, consumed)
		remaining.Sub(remaining, allocation.Minutes)
		if remaining.Sign() < 0 {
			if !allocation.Fixed {
				return fmt.Errorf("%w: negative remaining capacity for %s", schedulingdomain.ErrDataIntegrity, key)
			}
			// A fixed historical reservation may exceed capacity after a later
			// capacity edit. It still consumes the whole day for new work, while
			// its immutable projection remains reconstructable from existing rows.
			remaining = new(big.Rat)
		}
		consumed.Add(consumed, allocation.Minutes)
		used[key] = consumed
		project := state.projects[task.ProjectID]
		if allocation.Fixed || task.ActualEnd != nil || project.Status != "open" || !project.AutomaticScheduling {
			continue
		}
		rows = append(rows, allocationModel{
			TaskID: allocation.TaskID, AssigneeID: allocation.MemberID, Timeline: string(result.timeline),
			AllocationDate: schedulingdomain.DateOnly(allocation.Date), AllocatedMinutes: ratString(allocation.Minutes),
			RemainingCapacityMinutes: ratString(remaining), Sequence: allocation.Sequence,
		})
	}
	if len(rows) == 0 {
		return nil
	}
	if err := database.Create(&rows).Error; err != nil {
		return fmt.Errorf("persist %s allocations: %w", result.timeline, err)
	}
	return nil
}

func derivedProjectDates(state *portfolioState, projectID string) (*time.Time, *time.Time) {
	var earliest *time.Time
	var latest *time.Time
	for taskID, task := range state.tasks {
		if task.ProjectID != projectID {
			continue
		}
		if _, leaf := state.leafOrder[taskID]; !leaf {
			continue
		}
		start := task.ExecutionStart
		if start == nil {
			start = task.CommitmentStart
		}
		end := task.CommitmentEnd
		if end == nil {
			end = task.ExecutionEnd
		}
		if start != nil && (earliest == nil || start.Before(*earliest)) {
			earliest = datePointer(*start)
		}
		if end != nil && (latest == nil || end.After(*latest)) {
			latest = datePointer(*end)
		}
	}
	return earliest, latest
}

func (repository *Repository) MarkProjectUnscheduled(ctx context.Context, projectID, reason string) error {
	ctx, release := persistence.SerializeScheduleMutation(ctx)
	defer release()
	database := persistence.Transaction(ctx, repository.database)
	return database.Transaction(func(transaction *gorm.DB) error {
		if err := persistence.LockScheduleMutation(transaction); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		var project projectModel
		if err := transaction.Clauses(clause.Locking{Strength: "UPDATE"}).First(&project, "id = ?", projectID).Error; err != nil {
			return fmt.Errorf("load project for unscheduled transition: %w", err)
		}
		if project.Status != "open" || !project.AutomaticScheduling || project.SchedulingStartDate != nil {
			return nil
		}
		var taskIDs []string
		if err := transaction.Raw(`
			SELECT n.id
			FROM wbs_nodes n
			WHERE n.project_id = ?
			  AND n.actual_end IS NULL
			  AND NOT EXISTS (
				SELECT 1 FROM wbs_nodes child
				WHERE child.project_id = n.project_id AND child.parent_id = n.id
			  )`, projectID).Scan(&taskIDs).Error; err != nil {
			return fmt.Errorf("load project tasks for unscheduled transition: %w", err)
		}
		now := repository.now()
		if len(taskIDs) > 0 {
			if err := transaction.Model(&taskModel{}).Where("id IN ?", taskIDs).Updates(map[string]any{
				"execution_start": nil, "execution_end": nil,
				"commitment_start": nil, "commitment_end": nil,
				"execution_unscheduled_reason":  reason,
				"commitment_unscheduled_reason": reason,
				"updated_at":                    now,
			}).Error; err != nil {
				return fmt.Errorf("clear generated task dates: %w", err)
			}
			if err := transaction.Where("task_id IN ?", taskIDs).Delete(&allocationModel{}).Error; err != nil {
				return fmt.Errorf("clear generated task allocations: %w", err)
			}
		}
		var aggregate struct {
			StartDate *time.Time
			EndDate   *time.Time
		}
		if err := transaction.Raw(`
			SELECT MIN(COALESCE(execution_start, commitment_start)) AS start_date,
			       MAX(COALESCE(commitment_end, execution_end)) AS end_date
			FROM wbs_nodes n
			WHERE n.project_id = ?
			  AND NOT EXISTS (
				SELECT 1 FROM wbs_nodes child
				WHERE child.project_id = n.project_id AND child.parent_id = n.id
			  )`, projectID).Scan(&aggregate).Error; err != nil {
			return fmt.Errorf("derive project dates after unscheduled transition: %w", err)
		}
		result := transaction.Model(&projectModel{}).
			Where("id = ? AND schedule_version = ?", projectID, project.ScheduleVersion).
			Updates(map[string]any{
				"start_date": aggregate.StartDate, "end_date": aggregate.EndDate,
				"schedule_version": project.ScheduleVersion + 1, "updated_at": now,
			})
		if result.Error != nil {
			return fmt.Errorf("persist unscheduled project version: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("%w: project %s schedule version changed", schedulingdomain.ErrConcurrentConflict, projectID)
		}
		return nil
	})
}
