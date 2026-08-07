package gormrepo

import (
	"fmt"
	"strings"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
)

type lockedScopeImpacts struct {
	ProjectIDs []string
	GroupIDs   []string
}

func (state *portfolioState) potentialLockedScopeImpacts() (lockedScopeImpacts, error) {
	lockedProjects := make(map[string]struct{})
	lockedGroups := make(map[string]struct{})
	for projectID, project := range state.projects {
		if project.Status == "locked" {
			lockedProjects[projectID] = struct{}{}
		}
	}
	for taskID, task := range state.tasks {
		if _, leaf := state.leafOrder[taskID]; leaf {
			continue
		}
		if task.GroupLocalStatus == "locked" {
			lockedGroups[task.ID] = struct{}{}
		}
	}
	if len(lockedProjects) == 0 && len(lockedGroups) == 0 {
		return lockedScopeImpacts{}, nil
	}

	simulation := state.cloneForSimulation()
	for taskID, task := range simulation.tasks {
		if task.EffectiveLifecycle == "locked" {
			task.EffectiveLifecycle = "open"
		}
		simulation.tasks[taskID] = task
	}
	simulation.manualBlockers = make(map[string][]string)
	simulation.fixedBlockers = make(map[string][]string)
	simulation.fixed = map[schedulingdomain.Timeline][]taskModel{
		schedulingdomain.Execution:  {},
		schedulingdomain.Commitment: {},
	}
	simulation.classifyDependencies()
	simulation.classifyFixedTasks()

	execution, err := simulation.scheduleTimeline(schedulingdomain.Execution)
	if err != nil {
		return lockedScopeImpacts{}, err
	}
	commitment, err := simulation.scheduleTimeline(schedulingdomain.Commitment)
	if err != nil {
		return lockedScopeImpacts{}, err
	}
	impactedProjects := make(map[string]struct{})
	impactedGroups := make(map[string]struct{})
	for taskID, task := range state.tasks {
		if _, leaf := state.leafOrder[taskID]; !leaf || task.ActualStart != nil || task.ActualEnd != nil {
			continue
		}
		project := state.projects[task.ProjectID]
		projectOwnerID, groupOwnerID := protectedScopeOwner(task, project)
		if projectOwnerID == "" && groupOwnerID == "" {
			continue
		}
		executionSchedule, executionExists := execution.schedules[taskID]
		commitmentSchedule, commitmentExists := commitment.schedules[taskID]
		if !executionExists || !commitmentExists {
			return lockedScopeImpacts{}, fmt.Errorf("%w: missing protected-scope simulation for task %s", schedulingdomain.ErrDataIntegrity, taskID)
		}
		if scheduleDatesMatchTask(task, executionSchedule, commitmentSchedule) {
			continue
		}
		if groupOwnerID != "" {
			impactedGroups[groupOwnerID] = struct{}{}
		} else {
			impactedProjects[projectOwnerID] = struct{}{}
		}
	}
	return lockedScopeImpacts{ProjectIDs: mapKeys(impactedProjects), GroupIDs: mapKeys(impactedGroups)}, nil
}

func protectedScopeOwner(task taskModel, project projectModel) (string, string) {
	if task.EffectiveLifecycle != "locked" || task.EffectiveLockOwnerID == "" {
		return "", ""
	}
	if task.EffectiveLockOwnerID == project.ID {
		return project.ID, ""
	}
	return "", task.EffectiveLockOwnerID
}

func (state *portfolioState) impactGroup(groupID string, status string) (schedulingimpact.Group, bool) {
	group, exists := state.tasks[groupID]
	if !exists {
		return schedulingimpact.Group{}, false
	}
	if _, exists := state.projects[group.ProjectID]; !exists {
		return schedulingimpact.Group{}, false
	}
	return schedulingimpact.Group{ID: group.ID, ProjectID: group.ProjectID, Name: group.Name, Path: state.qualifiedGroupPath(group.ID), Status: status, Version: group.GroupSchedulingVersion}, true
}

func (state *portfolioState) qualifiedGroupPath(groupID string) string {
	group, exists := state.tasks[groupID]
	if !exists {
		return groupID
	}
	parts := []string{group.Name}
	current := group.ParentID
	seen := map[string]bool{group.ID: true}
	for current != nil && !seen[*current] {
		seen[*current] = true
		parent, ok := state.tasks[*current]
		if !ok {
			break
		}
		parts = append(parts, parent.Name)
		current = parent.ParentID
	}
	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	project := state.projects[group.ProjectID]
	return strings.Join(append([]string{project.Name}, parts...), " / ")
}

func (state *portfolioState) nearestGroupScope(taskID string) (schedulingimpact.Group, bool) {
	task, exists := state.tasks[taskID]
	if !exists {
		return schedulingimpact.Group{}, false
	}
	current := task.ParentID
	seen := map[string]bool{taskID: true}
	for current != nil && !seen[*current] {
		seen[*current] = true
		parent, ok := state.tasks[*current]
		if !ok {
			break
		}
		if _, leaf := state.leafOrder[parent.ID]; !leaf {
			return state.impactGroup(parent.ID, "open")
		}
		current = parent.ParentID
	}
	return schedulingimpact.Group{}, false
}

func (state *portfolioState) isWithinGroup(taskID, groupID string) bool {
	if taskID == groupID {
		return true
	}
	task, exists := state.tasks[taskID]
	if !exists {
		return false
	}
	current := task.ParentID
	seen := map[string]bool{taskID: true}
	for current != nil && !seen[*current] {
		if *current == groupID {
			return true
		}
		seen[*current] = true
		parent, ok := state.tasks[*current]
		if !ok {
			return false
		}
		current = parent.ParentID
	}
	return false
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
