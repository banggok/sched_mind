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
	timelineImpactedProjects := make(map[string]struct{})
	timelineImpactedTasks := make(map[string]struct{})
	for projectID := range dependencyDirtyProjects {
		dirtyProjects[projectID] = struct{}{}
	}
	taskChanges := make([]taskPersistenceChange, 0)
	for taskID, task := range state.tasks {
		if _, leaf := state.leafOrder[taskID]; !leaf || task.ActualStart != nil || task.ActualEnd != nil || task.EffectiveLifecycle != "open" || !task.EffectiveAutomaticScheduling {
			continue
		}
		executionSchedule, executionExists := execution.schedules[taskID]
		commitmentSchedule, commitmentExists := commitment.schedules[taskID]
		if !executionExists || !commitmentExists {
			return fmt.Errorf("%w: missing generated schedule for task %s", schedulingdomain.ErrDataIntegrity, taskID)
		}
		timelineChanged := !scheduleDatesMatchTask(task, executionSchedule, commitmentSchedule)
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
		if timelineChanged {
			timelineImpactedProjects[task.ProjectID] = struct{}{}
			timelineImpactedTasks[taskID] = struct{}{}
		}
	}
	// Open manual Tasks own authoritative dates. Their fixed allocation rows
	// participate in the shared projection according to Project Priority.
	for taskID, task := range state.tasks {
		if _, leaf := state.leafOrder[taskID]; !leaf || task.EffectiveLifecycle != "open" || task.EffectiveAutomaticScheduling || task.ActualStart != nil || task.ActualEnd != nil {
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
		if !exists || project.Status != "open" || !projectHasEffectiveAutomatic(state, projectID) {
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

	lockedImpacts, err := state.potentialLockedScopeImpacts()
	if err != nil {
		return err
	}
	lockedProjects := impactProjects(state, lockedImpacts.ProjectIDs, "locked")
	lockedGroups := make([]schedulingimpact.Group, 0, len(lockedImpacts.GroupIDs))
	for _, groupID := range lockedImpacts.GroupIDs {
		if group, exists := state.impactGroup(groupID, "locked"); exists {
			lockedGroups = append(lockedGroups, group)
		}
	}
	_, _, ownerGroupID, _, _ := schedulingimpact.OperationScope(ctx)
	openProjects := make([]schedulingimpact.Project, 0)
	openGroups := make([]schedulingimpact.Group, 0)
	if ownerGroupID == "" {
		openProjects = impactProjects(state, mapKeys(timelineImpactedProjects), "open")
	} else {
		projectSeen := make(map[string]struct{})
		groupSeen := make(map[string]struct{})
		for taskID := range timelineImpactedTasks {
			if state.isWithinGroup(taskID, ownerGroupID) {
				continue
			}
			if group, exists := state.nearestGroupScope(taskID); exists {
				if _, seen := groupSeen[group.ID]; !seen {
					groupSeen[group.ID] = struct{}{}
					openGroups = append(openGroups, group)
				}
				continue
			}
			task := state.tasks[taskID]
			if _, seen := projectSeen[task.ProjectID]; seen {
				continue
			}
			projectSeen[task.ProjectID] = struct{}{}
			openProjects = append(openProjects, impactProjects(state, []string{task.ProjectID}, "open")...)
		}
	}
	signature := persistenceSignature(state, taskChanges, projectChanges, generatedRows, dirtyTasks, dependencyDirtyProjects)
	if err := schedulingimpact.GuardScopes(ctx, signature, lockedProjects, openProjects, lockedGroups, openGroups); err != nil {
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
	state *portfolioState,
	taskChanges []taskPersistenceChange,
	projectChanges []projectPersistenceChange,
	generatedRows map[schedulingdomain.Timeline]map[string][]allocationModel,
	dirtyTasks map[string]struct{},
	dependencyDirtyProjects map[string]struct{},
) string {
	parts := []string{"state:" + schedulingStateSignature(state)}
	for _, change := range taskChanges {
		parts = append(parts, "task:"+change.TaskID+":"+updateSignature(change.Updates))
	}
	for _, change := range projectChanges {
		parts = append(parts, "project:"+change.ProjectID+":version:"+integerSignature(change.Version)+":"+dateSignature(change.StartDate)+":"+dateSignature(change.EndDate))
	}
	for _, timeline := range []schedulingdomain.Timeline{schedulingdomain.Execution, schedulingdomain.Commitment} {
		for _, taskID := range mapKeys(dirtyTasks) {
			for _, row := range generatedRows[timeline][taskID] {
				parts = append(parts, strings.Join([]string{
					"allocation", string(timeline), taskID,
					schedulingdomain.DateKey(row.AllocationDate),
					row.AssigneeID, row.AllocatedMinutes, row.RemainingCapacityMinutes, integerSignature(int64(row.Sequence)),
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

func schedulingStateSignature(state *portfolioState) string {
	if state == nil {
		return "nil"
	}
	parts := make([]string, 0)
	projectIDs := make([]string, 0, len(state.projects))
	for projectID := range state.projects {
		projectIDs = append(projectIDs, projectID)
	}
	sort.Strings(projectIDs)
	for _, projectID := range projectIDs {
		project := state.projects[projectID]
		parts = append(parts, strings.Join([]string{
			"project-state", project.ID, project.Status,
			integerSignature(int64(project.Priority)),
			fmt.Sprint(project.AutomaticScheduling),
			dateSignature(project.SchedulingStartDate),
			integerSignature(int64(project.ProjectBuffer)),
			dateSignature(project.StartDate), dateSignature(project.EndDate),
			integerSignature(project.ScheduleVersion),
		}, ":"))
	}

	taskIDs := make([]string, 0, len(state.tasks))
	memberIDs := make(map[string]struct{})
	for taskID, task := range state.tasks {
		taskIDs = append(taskIDs, taskID)
		if task.AssigneeID != nil && *task.AssigneeID != "" {
			memberIDs[*task.AssigneeID] = struct{}{}
		}
	}
	sort.Strings(taskIDs)
	for _, taskID := range taskIDs {
		task := state.tasks[taskID]
		parts = append(parts, strings.Join([]string{
			"task-state", task.ID, task.ProjectID, task.ParentKey, stringPointerSignature(task.ParentID),
			integerSignature(int64(task.Position)), stringPointerSignature(task.RoleID), stringPointerSignature(task.AssigneeID),
			intPointerSignature(task.EffortMinutes), integerSignature(int64(task.LagDays)), integerSignature(int64(task.CapacityAllocationPercentage)),
			dateSignature(task.ExecutionStart), dateSignature(task.ExecutionEnd), dateSignature(task.CommitmentStart), dateSignature(task.CommitmentEnd),
			dateSignature(task.ActualStart), dateSignature(task.ActualEnd), stringPointerSignature(task.ExecutionUnscheduledReason), stringPointerSignature(task.CommitmentUnscheduledReason),
			task.GroupSchedulingSource, boolPointerSignature(task.GroupAutomaticScheduling), dateSignature(task.GroupSchedulingStartDate),
			task.GroupLocalStatus, integerSignature(task.GroupSchedulingVersion), boolPointerSignature(task.GroupLockedAutomaticScheduling), dateSignature(task.GroupLockedSchedulingStartDate),
			fmt.Sprint(task.EffectiveAutomaticScheduling), dateSignature(task.EffectiveSchedulingStartDate), task.EffectiveLifecycle, task.EffectiveLockOwnerID, task.LocalGroupLockOwnerID,
		}, ":"))
	}

	sortedMemberIDs := make([]string, 0, len(memberIDs))
	for memberID := range memberIDs {
		sortedMemberIDs = append(sortedMemberIDs, memberID)
	}
	sort.Strings(sortedMemberIDs)
	for _, memberID := range sortedMemberIDs {
		member, exists := state.members[memberID]
		if !exists {
			parts = append(parts, "member-state:"+memberID+":missing")
			continue
		}
		parts = append(parts, strings.Join([]string{
			"member-state", member.ID, member.DailyCapacity, member.BufferPercentage,
		}, ":"))
		for _, override := range state.overrides[memberID] {
			parts = append(parts, strings.Join([]string{
				"override-state", memberID, schedulingdomain.DateKey(override.StartDate), schedulingdomain.DateKey(override.EndDate), override.Capacity,
			}, ":"))
		}
	}

	holidayDates := make([]string, 0, len(state.holidays))
	for date := range state.holidays {
		holidayDates = append(holidayDates, date)
	}
	sort.Strings(holidayDates)
	for _, date := range holidayDates {
		parts = append(parts, "holiday-state:"+date)
	}

	for _, dependency := range state.dependencies {
		parts = append(parts, "dependency-state:"+dependency.BlockingTaskID+":"+dependency.BlockedTaskID)
	}

	for _, timeline := range []schedulingdomain.Timeline{schedulingdomain.Execution, schedulingdomain.Commitment, schedulingdomain.Actual} {
		byTask := state.existingAllocations[timeline]
		allocationTaskIDs := make([]string, 0, len(byTask))
		for taskID := range byTask {
			allocationTaskIDs = append(allocationTaskIDs, taskID)
		}
		sort.Strings(allocationTaskIDs)
		for _, taskID := range allocationTaskIDs {
			for _, row := range byTask[taskID] {
				parts = append(parts, strings.Join([]string{
					"existing-allocation-state", string(timeline), taskID, row.AssigneeID, schedulingdomain.DateKey(row.AllocationDate),
					row.AllocatedMinutes, row.RemainingCapacityMinutes, integerSignature(int64(row.Sequence)),
				}, ":"))
			}
		}
	}

	sort.Strings(parts)
	return strings.Join(parts, "~")
}

func stringPointerSignature(value *string) string {
	if value == nil {
		return "null"
	}
	return *value
}

func intPointerSignature(value *int) string {
	if value == nil {
		return "null"
	}
	return integerSignature(int64(*value))
}

func boolPointerSignature(value *bool) string {
	if value == nil {
		return "null"
	}
	return fmt.Sprint(*value)
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

func integerSignature(value int64) string {
	return new(big.Int).SetInt64(value).String()
}

func buildTimelineAllocationRows(state *portfolioState, result *timelineResult) ([]allocationModel, error) {
	return buildTimelineAllocationRowsWithOvercapacity(state, result, false)
}

func buildTimelineAllocationRowsWithOvercapacity(state *portfolioState, result *timelineResult, allowOvercapacity bool) ([]allocationModel, error) {
	allocations := make([]dailyAllocation, 0)
	for _, schedule := range result.schedules {
		allocations = append(allocations, schedule.Allocations...)
	}
	sort.SliceStable(allocations, func(left, right int) bool { return allocationPrecedes(state, allocations[left], allocations[right]) })

	used := make(map[string]*big.Rat)
	rows := make([]allocationModel, 0, len(allocations))
	for projectionSequence, allocation := range allocations {
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
		manualFixed := allocation.Fixed && task.EffectiveLifecycle == "open" && !task.EffectiveAutomaticScheduling && task.ActualStart == nil && task.ActualEnd == nil
		if !manualFixed && (allocation.Fixed || task.ActualStart != nil || task.ActualEnd != nil || task.EffectiveLifecycle != "open" || !task.EffectiveAutomaticScheduling) {
			continue
		}
		rows = append(rows, allocationModel{
			TaskID: allocation.TaskID, AssigneeID: allocation.MemberID, Timeline: string(result.timeline),
			AllocationDate: schedulingdomain.DateOnly(allocation.Date), AllocatedMinutes: ratString(allocation.Minutes),
			RemainingCapacityMinutes: ratString(remaining), Sequence: projectionSequence + 1,
		})
	}
	return rows, nil
}

func allocationPrecedes(state *portfolioState, left, right dailyAllocation) bool {
	if left.MemberID != right.MemberID {
		return left.MemberID < right.MemberID
	}
	if !left.Date.Equal(right.Date) {
		return left.Date.Before(right.Date)
	}

	leftTask, leftExists := state.tasks[left.TaskID]
	rightTask, rightExists := state.tasks[right.TaskID]
	if leftExists && rightExists {
		leftAbsolute := left.Fixed && (leftTask.ActualStart != nil && leftTask.ActualEnd != nil || leftTask.EffectiveLifecycle == "locked")
		rightAbsolute := right.Fixed && (rightTask.ActualStart != nil && rightTask.ActualEnd != nil || rightTask.EffectiveLifecycle == "locked")
		if leftAbsolute != rightAbsolute {
			return leftAbsolute
		}
		if left.TaskID != right.TaskID {
			return compareTaskOrder(state, leftTask, rightTask)
		}
	}
	if left.TaskID != right.TaskID {
		return left.TaskID < right.TaskID
	}
	return left.Sequence < right.Sequence
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

func scheduleDatesMatchTask(task taskModel, execution, commitment taskSchedule) bool {
	return datesMatch(task.ExecutionStart, execution.Start) &&
		datesMatch(task.ExecutionEnd, execution.End) &&
		datesMatch(task.CommitmentStart, commitment.Start) &&
		datesMatch(task.CommitmentEnd, commitment.End)
}

func scheduleMatchesTask(task taskModel, execution, commitment taskSchedule) bool {
	return scheduleDatesMatchTask(task, execution, commitment) &&
		stringsMatch(task.ExecutionUnscheduledReason, execution.Reason) &&
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
			left[index].RemainingCapacityMinutes != right[index].RemainingCapacityMinutes ||
			left[index].Sequence != right[index].Sequence {
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
