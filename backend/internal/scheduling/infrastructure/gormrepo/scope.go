package gormrepo

import (
	"fmt"
	"sort"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"gorm.io/gorm"
)

func expandRequestedClosureRoots(database *gorm.DB, requestedProjectIDs []string) ([]string, error) {
	roots := uniqueProjectIDs(requestedProjectIDs)
	if len(roots) == 0 {
		return roots, nil
	}

	type taskScopeRow struct {
		ID         string
		AssigneeID *string
	}
	var ownerTasks []taskScopeRow
	if err := database.Table("wbs_nodes").
		Select("id, assignee_id").
		Where("project_id IN ?", roots).
		Scan(&ownerTasks).Error; err != nil {
		return nil, fmt.Errorf("load requested scheduling scope tasks: %w", err)
	}

	expanded := append([]string{}, roots...)
	assigneeIDs := make([]string, 0, len(ownerTasks))
	taskIDs := make([]string, 0, len(ownerTasks))
	for _, task := range ownerTasks {
		taskIDs = append(taskIDs, task.ID)
		if task.AssigneeID != nil && *task.AssigneeID != "" {
			assigneeIDs = append(assigneeIDs, *task.AssigneeID)
		}
	}
	assigneeIDs = uniqueProjectIDs(assigneeIDs)
	if len(assigneeIDs) > 0 {
		var sharedAssigneeProjects []string
		if err := database.Table("wbs_nodes AS node").
			Joins("JOIN projects AS project ON project.id = node.project_id").
			Where("node.assignee_id IN ? AND project.status <> ?", assigneeIDs, "closed").
			Distinct().
			Pluck("node.project_id", &sharedAssigneeProjects).Error; err != nil {
			return nil, fmt.Errorf("load shared-assignee scheduling scope: %w", err)
		}
		expanded = append(expanded, sharedAssigneeProjects...)
	}

	if len(taskIDs) > 0 {
		var dependencies []dependencyModel
		if err := database.Where("blocking_task_id IN ? OR blocked_task_id IN ?", taskIDs, taskIDs).Find(&dependencies).Error; err != nil {
			return nil, fmt.Errorf("load requested dependency scheduling scope: %w", err)
		}
		connectedTaskIDs := make([]string, 0, len(dependencies)*2)
		for _, dependency := range dependencies {
			connectedTaskIDs = append(connectedTaskIDs, dependency.BlockingTaskID, dependency.BlockedTaskID)
		}
		connectedTaskIDs = uniqueProjectIDs(connectedTaskIDs)
		if len(connectedTaskIDs) > 0 {
			var dependencyProjects []string
			if err := database.Table("wbs_nodes AS node").
				Joins("JOIN projects AS project ON project.id = node.project_id").
				Where("node.id IN ? AND project.status <> ?", connectedTaskIDs, "closed").
				Distinct().
				Pluck("node.project_id", &dependencyProjects).Error; err != nil {
				return nil, fmt.Errorf("load dependency-connected scheduling scope: %w", err)
			}
			expanded = append(expanded, dependencyProjects...)
		}
	}

	return uniqueProjectIDs(expanded), nil
}

func (state *portfolioState) restrictToRequestedClosure(requestedProjectIDs []string) {
	if len(requestedProjectIDs) == 0 || len(state.projects) == 0 {
		return
	}
	adjacency := make(map[string]map[string]struct{}, len(state.projects))
	connect := func(left, right string) {
		if left == "" || right == "" || left == right {
			return
		}
		if _, exists := state.projects[left]; !exists {
			return
		}
		if _, exists := state.projects[right]; !exists {
			return
		}
		if adjacency[left] == nil {
			adjacency[left] = make(map[string]struct{})
		}
		if adjacency[right] == nil {
			adjacency[right] = make(map[string]struct{})
		}
		adjacency[left][right] = struct{}{}
		adjacency[right][left] = struct{}{}
	}

	projectsByAssignee := make(map[string][]string)
	projectByTask := make(map[string]string, len(state.tasks))
	for _, task := range state.tasks {
		projectByTask[task.ID] = task.ProjectID
		if task.AssigneeID != nil {
			projectsByAssignee[*task.AssigneeID] = append(projectsByAssignee[*task.AssigneeID], task.ProjectID)
		}
	}
	for _, projectIDs := range projectsByAssignee {
		unique := uniqueProjectIDs(projectIDs)
		for left := 0; left < len(unique); left++ {
			for right := left + 1; right < len(unique); right++ {
				connect(unique[left], unique[right])
			}
		}
	}
	for _, dependency := range state.dependencies {
		connect(projectByTask[dependency.BlockingTaskID], projectByTask[dependency.BlockedTaskID])
	}

	included := make(map[string]struct{})
	queue := make([]string, 0, len(requestedProjectIDs))
	for _, projectID := range uniqueProjectIDs(requestedProjectIDs) {
		if _, exists := state.projects[projectID]; !exists {
			continue
		}
		included[projectID] = struct{}{}
		queue = append(queue, projectID)
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		neighbors := make([]string, 0, len(adjacency[current]))
		for neighbor := range adjacency[current] {
			neighbors = append(neighbors, neighbor)
		}
		sort.Strings(neighbors)
		for _, neighbor := range neighbors {
			if _, seen := included[neighbor]; seen {
				continue
			}
			included[neighbor] = struct{}{}
			queue = append(queue, neighbor)
		}
	}
	if len(included) == 0 {
		state.projects = map[string]projectModel{}
		state.projectOrder = nil
		state.tasks = map[string]taskModel{}
		state.leafOrder = map[string]int{}
		state.dependencies = nil
		return
	}

	for projectID := range state.projects {
		if _, keep := included[projectID]; !keep {
			delete(state.projects, projectID)
		}
	}
	projectOrder := state.projectOrder[:0]
	for _, projectID := range state.projectOrder {
		if _, keep := included[projectID]; keep {
			projectOrder = append(projectOrder, projectID)
		}
	}
	state.projectOrder = projectOrder

	includedTasks := make(map[string]struct{})
	for taskID, task := range state.tasks {
		if _, keep := included[task.ProjectID]; !keep {
			delete(state.tasks, taskID)
			delete(state.leafOrder, taskID)
			continue
		}
		includedTasks[taskID] = struct{}{}
	}
	dependencies := state.dependencies[:0]
	for _, dependency := range state.dependencies {
		if _, blocking := includedTasks[dependency.BlockingTaskID]; !blocking {
			continue
		}
		if _, blocked := includedTasks[dependency.BlockedTaskID]; !blocked {
			continue
		}
		dependencies = append(dependencies, dependency)
	}
	state.dependencies = dependencies
	for timeline, byTask := range state.existingAllocations {
		for taskID := range byTask {
			if _, keep := includedTasks[taskID]; !keep {
				delete(byTask, taskID)
			}
		}
		state.existingAllocations[timeline] = byTask
	}
	state.manualBlockers = make(map[string][]string)
	state.fixedBlockers = make(map[string][]string)
	state.fixed = map[schedulingdomain.Timeline][]taskModel{
		schedulingdomain.Execution:  {},
		schedulingdomain.Commitment: {},
	}
	state.classifyDependencies()
	state.classifyFixedTasks()
}

func uniqueProjectIDs(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
