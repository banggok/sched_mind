package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"gorm.io/gorm"
)

var errRecommendationComplete = errors.New("assignee recommendation calculation complete")

const (
	recommendationReasonAutomaticAnchorMissing = "AUTOMATIC_ANCHOR_MISSING"
	recommendationReasonNoPositiveCapacity     = "NO_POSITIVE_CAPACITY"
	recommendationReasonDependencyBlocked      = "DEPENDENCY_BLOCKED"
	recommendationReasonCandidateUnavailable   = "CANDIDATE_UNAVAILABLE"
	recommendationReasonSimulationFailed       = "CANDIDATE_SIMULATION_FAILED"
)

func (repository *Repository) RecommendAssignees(ctx context.Context, input schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	ctx, release := persistence.SerializeScheduleMutation(ctx)
	defer release()

	var result *schedulingdomain.AssigneeRecommendationResult
	err := repository.database.WithContext(ctx).Transaction(func(transaction *gorm.DB) error {
		if err := persistence.LockScheduleMutation(transaction); err != nil {
			return fmt.Errorf("lock assignee recommendation snapshot: %w", err)
		}
		state, err := repository.loadState(transaction)
		if err != nil {
			return fmt.Errorf("load assignee recommendation snapshot: %w", err)
		}
		if err := state.validateOrdering(); err != nil {
			return fmt.Errorf("validate assignee recommendation ordering: %w", err)
		}
		if err := state.validateEffectiveGraph(); err != nil {
			return fmt.Errorf("validate assignee recommendation dependencies: %w", err)
		}
		var roleCount int64
		if err := transaction.Table("roles").Where("id = ?", input.RoleID).Count(&roleCount).Error; err != nil {
			return fmt.Errorf("validate assignee recommendation role: %w", err)
		}
		if roleCount != 1 {
			return schedulingdomain.ErrAssigneeRecommendationInputInvalid
		}

		project, projectExists := state.projects[input.ProjectID]
		if !projectExists {
			var projectStatus string
			if err := transaction.Table("projects").Select("status").Where("id = ?", input.ProjectID).Scan(&projectStatus).Error; err != nil {
				return fmt.Errorf("resolve assignee recommendation Project lifecycle: %w", err)
			}
			if projectStatus == "closed" {
				return schedulingdomain.ErrTaskNotRecommendable
			}
			return schedulingdomain.ErrAssigneeRecommendationInputInvalid
		}
		task, taskExists := state.tasks[input.TaskID]
		if !taskExists || task.ProjectID != input.ProjectID {
			return schedulingdomain.ErrAssigneeRecommendationInputInvalid
		}
		if project.Status != "open" || task.ActualStart != nil || task.ActualEnd != nil {
			return schedulingdomain.ErrTaskNotRecommendable
		}
		if _, leaf := state.leafOrder[input.TaskID]; !leaf {
			return schedulingdomain.ErrTaskNotRecommendable
		}

		calculatedOn := schedulingdomain.DateOnly(input.CalculatedOn)
		mode := schedulingdomain.RecommendationAutomatic
		if !project.AutomaticScheduling {
			mode = schedulingdomain.RecommendationManualAdvisory
		}
		candidates, err := loadRecommendationCandidates(transaction, input.RoleID)
		if err != nil {
			return err
		}
		versions := make(map[string]int64, len(state.projects))
		for projectID, value := range state.projects {
			versions[projectID] = value.ScheduleVersion
		}
		result = &schedulingdomain.AssigneeRecommendationResult{
			CalculatedOnDate:        calculatedOn,
			ProjectScheduleVersions: versions,
			Mode:                    mode,
			Items:                   []schedulingdomain.AssigneeRecommendationItem{},
		}

		if project.AutomaticScheduling && project.SchedulingStartDate == nil {
			for _, candidate := range candidates {
				reason := recommendationReasonAutomaticAnchorMissing
				result.Items = append(result.Items, noCompletionRecommendation(candidate, reason))
			}
			result.Items = schedulingdomain.RankAssigneeRecommendations(result.Items)
			return errRecommendationComplete
		}

		prepared, preserveManual, err := prepareRecommendationState(state, input, project, task, calculatedOn)
		if err != nil {
			return err
		}
		baseline, err := prepared.scheduleTimeline(schedulingdomain.Execution)
		if err != nil {
			return fmt.Errorf("calculate assignee recommendation baseline: %w", err)
		}

		items := make([]schedulingdomain.AssigneeRecommendationItem, 0, len(candidates))
		for _, candidate := range candidates {
			simulation := cloneRecommendationState(prepared)
			candidateTask := simulation.tasks[input.TaskID]
			candidateID := candidate.ID
			candidateTask.AssigneeID = &candidateID
			simulation.tasks[input.TaskID] = candidateTask
			reclassifyRecommendationState(simulation, input.TaskID, input.ProjectID, preserveManual)

			simulated, simulationErr := simulation.scheduleTimeline(schedulingdomain.Execution)
			if simulationErr != nil {
				items = append(items, noCompletionRecommendation(candidate, recommendationReasonSimulationFailed))
				continue
			}
			schedule, scheduled := simulated.schedules[input.TaskID]
			if !scheduled || schedule.End == nil {
				items = append(items, noCompletionRecommendation(candidate, recommendationReason(schedule.Reason)))
				continue
			}
			remaining := 0
			if metric, exists := simulated.completionMetrics[input.TaskID]; exists {
				remaining = roundedRatMinutes(metric.RemainingCapacityMinutes)
			}
			incremental, metricErr := incrementalOvercapacityMinutes(baseline.calendar, simulated.calendar, candidate.ID, input.ProjectID)
			if metricErr != nil {
				items = append(items, noCompletionRecommendation(candidate, recommendationReasonSimulationFailed))
				continue
			}
			group := schedulingdomain.RecommendationFeasible
			if incremental > 0 {
				group = schedulingdomain.RecommendationOvercapacity
			}
			end := schedulingdomain.DateOnly(*schedule.End)
			items = append(items, schedulingdomain.AssigneeRecommendationItem{
				MemberID:                          candidate.ID,
				MemberName:                        candidate.Name,
				RoleID:                            candidate.RoleID,
				RankGroup:                         group,
				ExecutionEnd:                      &end,
				RemainingExecutionCapacityMinutes: remaining,
				IncrementalOvercapacityMinutes:    incremental,
			})
		}
		result.Items = schedulingdomain.RankAssigneeRecommendations(items)
		return errRecommendationComplete
	})
	if errors.Is(err, errRecommendationComplete) {
		return result, nil
	}
	if errors.Is(err, schedulingdomain.ErrTaskNotRecommendable) || errors.Is(err, schedulingdomain.ErrAssigneeRecommendationInputInvalid) {
		return nil, err
	}
	return nil, fmt.Errorf("%w: %v", schedulingdomain.ErrAssigneeRecommendationUnavailable, err)
}

func loadRecommendationCandidates(transaction *gorm.DB, roleID string) ([]memberModel, error) {
	values := make([]memberModel, 0)
	if err := transaction.
		Where("role_id = ? AND deleted_at IS NULL", roleID).
		Order("LOWER(name) ASC").
		Order("id ASC").
		Find(&values).Error; err != nil {
		return nil, fmt.Errorf("load assignee recommendation candidates: %w", err)
	}
	return values, nil
}

func prepareRecommendationState(state *portfolioState, input schedulingdomain.AssigneeRecommendationInput, project projectModel, task taskModel, calculatedOn time.Time) (*portfolioState, bool, error) {
	prepared := cloneRecommendationState(state)
	for timeline := range prepared.existingAllocations {
		delete(prepared.existingAllocations[timeline], input.TaskID)
	}

	effort := input.EffortMinutes
	roleID := input.RoleID
	task.RoleID = &roleID
	task.AssigneeID = nil
	task.EffortMinutes = &effort
	task.LagDays = input.LagDays
	task.CapacityAllocationPercentage = input.CapacityAllocationPercentage
	task.ExecutionStart = nil
	task.ExecutionEnd = nil
	task.CommitmentStart = nil
	task.CommitmentEnd = nil
	task.ExecutionUnscheduledReason = nil
	task.CommitmentUnscheduledReason = nil
	prepared.tasks[input.TaskID] = task

	preserveManual := !project.AutomaticScheduling
	if preserveManual {
		anchor := calculatedOn
		if input.ExecutionStart != nil {
			anchor = schedulingdomain.DateOnly(*input.ExecutionStart)
		} else if project.SchedulingStartDate != nil {
			projectStart := schedulingdomain.DateOnly(*project.SchedulingStartDate)
			if projectStart.After(anchor) {
				anchor = projectStart
			}
		}
		project.AutomaticScheduling = true
		project.SchedulingStartDate = &anchor
		prepared.projects[input.ProjectID] = project
	}
	reclassifyRecommendationState(prepared, input.TaskID, input.ProjectID, preserveManual)
	return prepared, preserveManual, nil
}

func reclassifyRecommendationState(state *portfolioState, editedTaskID, projectID string, preserveManual bool) {
	state.manualBlockers = make(map[string][]string)
	state.fixedBlockers = make(map[string][]string)
	state.fixed = map[schedulingdomain.Timeline][]taskModel{
		schedulingdomain.Execution:  {},
		schedulingdomain.Commitment: {},
	}
	state.classifyDependencies()
	state.classifyFixedTasks()
	if !preserveManual {
		return
	}
	for _, task := range state.tasks {
		if task.ProjectID != projectID || task.ID == editedTaskID {
			continue
		}
		state.fixed[schedulingdomain.Execution] = appendTaskOnce(state.fixed[schedulingdomain.Execution], task)
		state.fixed[schedulingdomain.Commitment] = appendTaskOnce(state.fixed[schedulingdomain.Commitment], task)
	}
}

func appendTaskOnce(values []taskModel, candidate taskModel) []taskModel {
	for _, value := range values {
		if value.ID == candidate.ID {
			return values
		}
	}
	return append(values, candidate)
}

func cloneRecommendationState(state *portfolioState) *portfolioState {
	clone := state.cloneForSimulation()
	clone.members = make(map[string]memberModel, len(state.members))
	for id, member := range state.members {
		clone.members[id] = member
	}
	clone.overrides = make(map[string][]capacityOverrideModel, len(state.overrides))
	for memberID, values := range state.overrides {
		clone.overrides[memberID] = append([]capacityOverrideModel{}, values...)
	}
	clone.holidays = make(map[string]struct{}, len(state.holidays))
	for date := range state.holidays {
		clone.holidays[date] = struct{}{}
	}
	clone.existingAllocations = map[schedulingdomain.Timeline]map[string][]allocationModel{
		schedulingdomain.Execution:  {},
		schedulingdomain.Commitment: {},
		schedulingdomain.Actual:     {},
	}
	for timeline, byTask := range state.existingAllocations {
		for taskID, rows := range byTask {
			clone.existingAllocations[timeline][taskID] = append([]allocationModel{}, rows...)
		}
	}
	return clone
}

func noCompletionRecommendation(candidate memberModel, reason string) schedulingdomain.AssigneeRecommendationItem {
	return schedulingdomain.AssigneeRecommendationItem{
		MemberID:   candidate.ID,
		MemberName: candidate.Name,
		RoleID:     candidate.RoleID,
		RankGroup:  schedulingdomain.RecommendationNoCompletion,
		ReasonCode: &reason,
	}
}

func recommendationReason(reason *string) string {
	if reason == nil {
		return recommendationReasonCandidateUnavailable
	}
	switch *reason {
	case reasonZeroCapacity:
		return recommendationReasonNoPositiveCapacity
	case reasonBlocked:
		return recommendationReasonDependencyBlocked
	case reasonMissingAnchor:
		return recommendationReasonAutomaticAnchorMissing
	default:
		return recommendationReasonCandidateUnavailable
	}
}

func incrementalOvercapacityMinutes(baseline, simulated *allocationCalendar, memberID, projectID string) (int, error) {
	dateKeys := make(map[string]struct{})
	for key := range baseline.allocations[memberID] {
		dateKeys[key] = struct{}{}
	}
	for key := range simulated.allocations[memberID] {
		dateKeys[key] = struct{}{}
	}
	total := new(big.Rat)
	for key := range dateKeys {
		date, err := time.Parse("2006-01-02", key)
		if err != nil {
			return 0, err
		}
		capacity, err := simulated.capacity(memberID, projectID, date)
		if err != nil {
			return 0, err
		}
		baselineOver := positiveDifference(totalAllocationMinutes(baseline.day(memberID, date)), capacity)
		simulatedOver := positiveDifference(totalAllocationMinutes(simulated.day(memberID, date)), capacity)
		increase := new(big.Rat).Sub(simulatedOver, baselineOver)
		if increase.Sign() > 0 {
			total.Add(total, increase)
		}
	}
	return roundedRatMinutes(total), nil
}

func totalAllocationMinutes(values []dailyAllocation) *big.Rat {
	total := new(big.Rat)
	for _, value := range values {
		total.Add(total, value.Minutes)
	}
	return total
}

func positiveDifference(left, right *big.Rat) *big.Rat {
	value := new(big.Rat).Sub(left, right)
	if value.Sign() < 0 {
		return new(big.Rat)
	}
	return value
}

func roundedRatMinutes(value *big.Rat) int {
	if value == nil || value.Sign() == 0 {
		return 0
	}
	sign := value.Sign()
	numerator := new(big.Int).Abs(new(big.Int).Set(value.Num()))
	denominator := new(big.Int).Set(value.Denom())
	quotient, remainder := new(big.Int).QuoRem(numerator, denominator, new(big.Int))
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(denominator) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if sign < 0 {
		quotient.Neg(quotient)
	}
	return int(quotient.Int64())
}
