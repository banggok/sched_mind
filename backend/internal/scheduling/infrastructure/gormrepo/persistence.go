package gormrepo

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
	"gorm.io/gorm"
)

type taskPersistenceChange struct {
	TaskID  string
	Updates map[string]any
}

type projectPersistenceChange struct {
	ProjectID string
	Version   int64
	StartDate *time.Time
	EndDate   *time.Time
}

func completeTimeline(start, end *time.Time) bool { return start != nil && end != nil }

func (repository *Repository) persist(
	ctx context.Context,
	database *gorm.DB,
	state *portfolioState,
	execution *timelineResult,
	commitment *timelineResult,
	dependencyDirtyProjects map[string]struct{},
) error {
	now := repository.now()
	executionRows, err := buildTimelineAllocationRows(state, execution)
	if err != nil {
		return err
	}
	commitmentRows, err := buildTimelineAllocationRows(state, commitment)
	if err != nil {
		return err
	}
	generatedRows := map[schedulingdomain.Timeline]map[string][]allocationModel{
		schedulingdomain.Execution:  groupAllocationRows(executionRows),
		schedulingdomain.Commitment: groupAllocationRows(commitmentRows),
	}

	dirtyTasks := make(map[string]struct{})
	dirtyProjects := make(map[string]struct{}, len(dependencyDirtyProjects))
	for projectID := range dependencyDirtyProjects {
		dirtyProjects[projectID] = struct{}{}
	}
	taskChanges := make([]taskPersistenceChange, 0)
	for taskID, task := range state.tasks {
		project := state.projects[task.ProjectID]
		if _, leaf := state.leafOrder[taskID]; !leaf || task.ActualStart != nil || task.ActualEnd != nil || project.Status != "open" || !project.AutomaticScheduling {
			continue
		}
		executionSchedule, executionExists := execution.schedules[taskID]
		commitmentSchedule, commitmentExists := commitment.schedules[taskID]
		if !executionExists || !commitmentExists {
			return fmt.Errorf("%w: missing generated schedule for task %s", schedulingdomain.ErrDataIntegrity, taskID)
		}
		changed := !scheduleMatchesTask(task, executionSchedule, commitmentSchedule) ||
			!allocationRowsEquivalent(state.existingAllocations[schedulingdomain.Execution][taskID], generatedRows[schedulingdomain.Execution][taskID]) ||
			!allocationRowsEquivalent(state.existingAllocations[schedulingdomain.Commitment][taskID], generatedRows[schedulingdomain.Commitment][taskID])
		if !changed {
			continue
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
		taskChanges = append(taskChanges, taskPersistenceChange{TaskID: taskID, Updates: updates})
		task.ExecutionStart = executionSchedule.Start
		task.ExecutionEnd = executionSchedule.End
		task.ExecutionUnscheduledReason = executionSchedule.Reason
		task.CommitmentStart = commitmentSchedule.Start
		task.CommitmentEnd = commitmentSchedule.End
		task.CommitmentUnscheduledReason = commitmentSchedule.Reason
		task.UpdatedAt = now
		state.tasks[taskID] = task
		dirtyTasks[taskID] = struct{}{}
		dirtyProjects[task.ProjectID] = struct{}{}
	}
	// Open manual Tasks own authoritative dates but their fixed allocation rows
	// still participate in the shared portfolio capacity projection.
	for taskID, task := range state.tasks {
		project := state.projects[task.ProjectID]
		if _, leaf := state.leafOrder[taskID]; !leaf || project.Status != "open" || project.AutomaticScheduling || task.ActualStart != nil || task.ActualEnd != nil {
			continue
		}
		executionComplete := completeTimeline(task.ExecutionStart, task.ExecutionEnd)
		commitmentComplete := completeTimeline(task.CommitmentStart, task.CommitmentEnd)
		if !executionComplete && !commitmentComplete {
			continue
		}
		changed := !allocationRowsEquivalent(state.existingAllocations[schedulingdomain.Execution][taskID], generatedRows[schedulingdomain.Execution][taskID]) ||
			!allocationRowsEquivalent(state.existingAllocations[schedulingdomain.Commitment][taskID], generatedRows[schedulingdomain.Commitment][taskID])
		if changed {
			dirtyTasks[taskID] = struct{}{}
			dirtyProjects[task.ProjectID] = struct{}{}
		}
	}

	projectChanges := make([]projectPersistenceChange, 0, len(dirtyProjects))
	for projectID := range dirtyProjects {
		project, exists := state.projects[projectID]
		if !exists || project.Status != "open" || !project.AutomaticScheduling {
			continue
		}
		start, end := derivedProjectDates(state, projectID)
		projectChanges = append(projectChanges, projectPersistenceChange{
			ProjectID: projectID,
			Version:   project.ScheduleVersion,
			StartDate: start,
			EndDate:   end,
		})
	}
	sort.Slice(projectChanges, func(left, right int) bool { return projectChanges[left].ProjectID < projectChanges[right].ProjectID })

	lockedImpactIDs, err := state.potentialLockedImpacts()
	if err != nil {
		return err
	}
	lockedProjects := impactProjects(state, lockedImpactIDs, "locked")
	openProjects := impactProjects(state, mapKeys(dirtyProjects), "open")
	signature := persistenceSignature(taskChanges, projectChanges, generatedRows, dirtyTasks, dependencyDirtyProjects)
	if err := schedulingimpact.Guard(ctx, signature, lockedProjects, openProjects); err != nil {
		return err
	}

	for _, change := range taskChanges {
		result := database.Model(&taskModel{}).
			Where("id = ? AND actual_start IS NULL AND actual_end IS NULL", change.TaskID).
			Updates(change.Updates)
		if result.Error != nil {
			return fmt.Errorf("persist generated task schedule: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("%w: task %s changed while scheduling", schedulingdomain.ErrConcurrentConflict, change.TaskID)
		}
	}

	if len(dirtyTasks) > 0 {
		taskIDs := mapKeys(dirtyTasks)
		if err := database.Where("task_id IN ? AND timeline IN ?", taskIDs, []string{string(schedulingdomain.Execution), string(schedulingdomain.Commitment)}).Delete(&allocationModel{}).Error; err != nil {
			return fmt.Errorf("replace schedule allocations: %w", err)
		}
		if err := persistSelectedAllocationRows(database, executionRows, dirtyTasks); err != nil {
			return err
		}
		if err := persistSelectedAllocationRows(database, commitmentRows, dirtyTasks); err != nil {
			return err
		}
	}

	for _, change := range projectChanges {
		result := database.Model(&projectModel{}).
			Where("id = ? AND schedule_version = ?", change.ProjectID, change.Version).
			Updates(map[string]any{
				"start_date":       change.StartDate,
				"end_date":         change.EndDate,
				"schedule_version": change.Version + 1,
				"updated_at":       now,
			})
		if result.Error != nil {
			return fmt.Errorf("persist project schedule version: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return fmt.Errorf("%w: project %s schedule version changed", schedulingdomain.ErrConcurrentConflict, change.ProjectID)
		}
	}
	return nil
}

func impactProjects(state *portfolioState, projectIDs []string, requiredStatus string) []schedulingimpact.Project {
	values := make([]schedulingimpact.Project, 0, len(projectIDs))
	for _, projectID := range projectIDs {
		project, exists := state.projects[projectID]
		if !exists || project.Status != requiredStatus {
			continue
		}
		values = append(values, schedulingimpact.Project{
			ID:      project.ID,
			Name:    project.Name,
			Status:  project.Status,
			Version: project.ScheduleVersion,
		})
	}
	return values
}

func persistenceSignature(
	taskChanges []taskPersistenceChange,
	projectChanges []projectPersistenceChange,
	generatedRows map[schedulingdomain.Timeline]map[string][]allocationModel,
	dirtyTasks map[string]struct{},
	dependencyDirtyProjects map[string]struct{},
) string {
	parts := make([]string, 0)
	for _, change := range taskChanges {
		parts = append(parts, "task:"+change.TaskID+":"+updateSignature(change.Updates))
	}
	for _, change := range projectChanges {
		parts = append(parts, "project:"+change.ProjectID+":"+dateSignature(change.StartDate)+":"+dateSignature(change.EndDate))
	}
	for _, timeline := range []schedulingdomain.Timeline{schedulingdomain.Execution, schedulingdomain.Commitment} {
		for _, taskID := range mapKeys(dirtyTasks) {
			for _, row := range generatedRows[timeline][taskID] {
				parts = append(parts, strings.Join([]string{
					"allocation", string(timeline), taskID,
					schedulingdomain.DateKey(row.AllocationDate),
					row.AllocatedMinutes, row.RemainingCapacityMinutes,
				}, ":"))
			}
		}
	}
	for _, projectID := range mapKeys(dependencyDirtyProjects) {
		parts = append(parts, "dependency:"+projectID)
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func updateSignature(updates map[string]any) string {
	keys := make([]string, 0, len(updates))
	for key := range updates {
		if key != "updated_at" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+valueSignature(updates[key]))
	}
	return strings.Join(parts, ",")
}

func valueSignature(value any) string {
	switch typed := value.(type) {
	case nil:
		return "null"
	case *time.Time:
		return dateSignature(typed)
	case *string:
		if typed == nil {
			return "null"
		}
		return *typed
	case string:
		return typed
	default:
		return fmt.Sprint(typed)
	}
}

func dateSignature(value *time.Time) string {
	if value == nil {
		return "null"
	}
	return schedulingdomain.DateKey(*value)
}

func buildTimelineAllocationRows(state *portfolioState, result *timelineResult) ([]allocationModel, error) {
	return buildTimelineAllocationRowsWithOvercapacity(state, result, false)
}

func buildTimelineAllocationRowsWithOvercapacity(state *portfolioState, result *timelineResult, allowOvercapacity bool) ([]allocationModel, error) {
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
			return nil, fmt.Errorf("%w: allocation task %s missing", schedulingdomain.ErrDataIntegrity, allocation.TaskID)
		}
		capacity, err := result.calendar.capacity(allocation.MemberID, task.ProjectID, allocation.Date)
		if err != nil {
			return nil, err
		}
		key := allocation.MemberID + "|" + schedulingdomain.DateKey(allocation.Date)
		consumed := schedulingdomain.CloneRat(used[key])
		remaining := schedulingdomain.CloneRat(capacity)
		remaining.Sub(remaining, consumed)
		remaining.Sub(remaining, allocation.Minutes)
		if remaining.Sign() < 0 {
			if !allocation.Fixed && !allowOvercapacity {
				return nil, fmt.Errorf("%w: negative remaining capacity for %s", schedulingdomain.ErrDataIntegrity, key)
			}
			remaining = new(big.Rat)
		}
		consumed.Add(consumed, allocation.Minutes)
		used[key] = consumed
		project := state.projects[task.ProjectID]
		manualFixed := allocation.Fixed && project.Status == "open" && !project.AutomaticScheduling && task.ActualStart == nil && task.ActualEnd == nil
		if !manualFixed && (allocation.Fixed || task.ActualStart != nil || task.ActualEnd != nil || project.Status != "open" || !project.AutomaticScheduling) {
			continue
		}
		rows = append(rows, allocationModel{
			TaskID: allocation.TaskID, AssigneeID: allocation.MemberID, Timeline: string(result.timeline),
			AllocationDate: schedulingdomain.DateOnly(allocation.Date), AllocatedMinutes: ratString(allocation.Minutes),
			RemainingCapacityMinutes: ratString(remaining), Sequence: allocation.Sequence,
		})
	}
	return rows, nil
}

func persistSelectedAllocationRows(database *gorm.DB, rows []allocationModel, selected map[string]struct{}) error {
	values := make([]allocationModel, 0, len(rows))
	for _, row := range rows {
		if _, include := selected[row.TaskID]; include {
			values = append(values, row)
		}
	}
	if len(values) == 0 {
		return nil
	}
	if err := database.Create(&values).Error; err != nil {
		return fmt.Errorf("persist schedule allocations: %w", err)
	}
	return nil
}

func groupAllocationRows(rows []allocationModel) map[string][]allocationModel {
	grouped := make(map[string][]allocationModel)
	for _, row := range rows {
		grouped[row.TaskID] = append(grouped[row.TaskID], row)
	}
	return grouped
}

func scheduleMatchesTask(task taskModel, execution, commitment taskSchedule) bool {
	return datesMatch(task.ExecutionStart, execution.Start) &&
		datesMatch(task.ExecutionEnd, execution.End) &&
		stringsMatch(task.ExecutionUnscheduledReason, execution.Reason) &&
		datesMatch(task.CommitmentStart, commitment.Start) &&
		datesMatch(task.CommitmentEnd, commitment.End) &&
		stringsMatch(task.CommitmentUnscheduledReason, commitment.Reason)
}

func allocationRowsEquivalent(existing, generated []allocationModel) bool {
	if len(existing) != len(generated) {
		return false
	}
	left := append([]allocationModel{}, existing...)
	right := append([]allocationModel{}, generated...)
	sort.Slice(left, func(i, j int) bool { return allocationRowLess(left[i], left[j]) })
	sort.Slice(right, func(i, j int) bool { return allocationRowLess(right[i], right[j]) })
	for index := range left {
		if left[index].AssigneeID != right[index].AssigneeID ||
			!schedulingdomain.DateOnly(left[index].AllocationDate).Equal(schedulingdomain.DateOnly(right[index].AllocationDate)) ||
			left[index].AllocatedMinutes != right[index].AllocatedMinutes ||
			left[index].RemainingCapacityMinutes != right[index].RemainingCapacityMinutes {
			return false
		}
	}
	return true
}

func allocationRowLess(left, right allocationModel) bool {
	if !left.AllocationDate.Equal(right.AllocationDate) {
		return left.AllocationDate.Before(right.AllocationDate)
	}
	if left.AllocatedMinutes != right.AllocatedMinutes {
		return left.AllocatedMinutes < right.AllocatedMinutes
	}
	return left.RemainingCapacityMinutes < right.RemainingCapacityMinutes
}

func datesMatch(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return schedulingdomain.DateOnly(*left).Equal(schedulingdomain.DateOnly(*right))
}

func stringsMatch(left, right *string) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func mapKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return keys
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
	_ = reason
	return repository.RecalculatePortfolio(ctx, []string{projectID})
}
