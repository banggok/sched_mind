package gormrepo

import (
	"fmt"
	"sort"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
)

func (state *portfolioState) potentialLockedImpacts() ([]string, error) {
	locked := make(map[string]struct{})
	for projectID, project := range state.projects {
		if project.Status == "locked" && project.AutomaticScheduling {
			locked[projectID] = struct{}{}
		}
	}
	if len(locked) == 0 {
		return nil, nil
	}

	simulation := state.cloneForSimulation()
	for projectID := range locked {
		project := simulation.projects[projectID]
		project.Status = "open"
		simulation.projects[projectID] = project
	}
	simulation.manualBlockers = make(map[string][]string)
	simulation.fixedBlockers = make(map[string][]string)
	simulation.fixed = map[schedulingdomain.Timeline][]taskModel{
		schedulingdomain.Execution:  {},
		schedulingdomain.Commitment: {},
	}
	simulation.classifyDependencies()
	simulation.classifyFixedTasks()

	var execution *timelineResult
	previousBlockers := map[string]string{}
	converged := false
	for iteration := 0; iteration < len(simulation.tasks)+1; iteration++ {
		result, err := simulation.scheduleTimeline(schedulingdomain.Execution)
		if err != nil {
			return nil, err
		}
		execution = result
		if blockerMapsEqual(previousBlockers, execution.automaticBlocker) {
			converged = true
			break
		}
		previousBlockers = cloneBlockerMap(execution.automaticBlocker)
		simulation.refreshDependenciesFromExecution(execution)
		if err := simulation.validateEffectiveGraph(); err != nil {
			return nil, err
		}
	}
	if !converged {
		return nil, schedulingdomain.ErrNoConvergence
	}
	commitment, err := simulation.scheduleTimeline(schedulingdomain.Commitment)
	if err != nil {
		return nil, err
	}
	executionRows, err := buildTimelineAllocationRowsWithOvercapacity(simulation, execution, true)
	if err != nil {
		return nil, err
	}
	commitmentRows, err := buildTimelineAllocationRowsWithOvercapacity(simulation, commitment, true)
	if err != nil {
		return nil, err
	}
	generatedRows := map[schedulingdomain.Timeline]map[string][]allocationModel{
		schedulingdomain.Execution:  groupAllocationRows(executionRows),
		schedulingdomain.Commitment: groupAllocationRows(commitmentRows),
	}

	impacted := make(map[string]struct{})
	for taskID, task := range state.tasks {
		if _, lockedProject := locked[task.ProjectID]; !lockedProject {
			continue
		}
		if _, leaf := state.leafOrder[taskID]; !leaf || task.ActualStart != nil || task.ActualEnd != nil {
			continue
		}
		executionSchedule, executionExists := execution.schedules[taskID]
		commitmentSchedule, commitmentExists := commitment.schedules[taskID]
		if !executionExists || !commitmentExists {
			return nil, fmt.Errorf("%w: missing locked-project simulation for task %s", schedulingdomain.ErrDataIntegrity, taskID)
		}
		if !scheduleMatchesTask(task, executionSchedule, commitmentSchedule) ||
			!allocationRowsEquivalent(state.existingAllocations[schedulingdomain.Execution][taskID], generatedRows[schedulingdomain.Execution][taskID]) ||
			!allocationRowsEquivalent(state.existingAllocations[schedulingdomain.Commitment][taskID], generatedRows[schedulingdomain.Commitment][taskID]) ||
			existingAutomaticBlocker(state, taskID) != execution.automaticBlocker[taskID] {
			impacted[task.ProjectID] = struct{}{}
		}
	}
	return mapKeys(impacted), nil
}

func (state *portfolioState) cloneForSimulation() *portfolioState {
	clone := &portfolioState{
		projects:            make(map[string]projectModel, len(state.projects)),
		projectOrder:        append([]string{}, state.projectOrder...),
		tasks:               make(map[string]taskModel, len(state.tasks)),
		leafOrder:           make(map[string]int, len(state.leafOrder)),
		members:             state.members,
		overrides:           state.overrides,
		holidays:            state.holidays,
		dependencies:        append([]dependencyModel{}, state.dependencies...),
		existingAllocations: state.existingAllocations,
		manualBlockers:      make(map[string][]string),
		fixedBlockers:       make(map[string][]string),
		fixed: map[schedulingdomain.Timeline][]taskModel{
			schedulingdomain.Execution:  {},
			schedulingdomain.Commitment: {},
		},
	}
	for id, project := range state.projects {
		clone.projects[id] = project
	}
	for id, task := range state.tasks {
		clone.tasks[id] = task
	}
	for id, order := range state.leafOrder {
		clone.leafOrder[id] = order
	}
	return clone
}

func existingAutomaticBlocker(state *portfolioState, blockedTaskID string) string {
	values := make([]string, 0)
	for _, dependency := range state.dependencies {
		if dependency.BlockedTaskID == blockedTaskID && dependency.AutomaticOwned {
			values = append(values, dependency.BlockingTaskID)
		}
	}
	sort.Strings(values)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func blockerMapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for taskID, blockerID := range left {
		if right[taskID] != blockerID {
			return false
		}
	}
	return true
}

func cloneBlockerMap(value map[string]string) map[string]string {
	clone := make(map[string]string, len(value))
	for taskID, blockerID := range value {
		clone[taskID] = blockerID
	}
	return clone
}
