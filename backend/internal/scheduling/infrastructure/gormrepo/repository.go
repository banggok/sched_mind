package gormrepo

import (
	"context"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/scheduling/application"
	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	groupscheduling "github.com/banggok/sched_mind/backend/internal/shared/groupscheduling"
	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	"github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maximumScheduleDays = 3660

const (
	reasonMissingAnchor   = "Automatic Scheduling requires an effective Scheduling Start Date."
	reasonMissingAssignee = "Task requires an Assignee before it can be scheduled."
	reasonMissingEffort   = "Task requires valid Effort before it can be scheduled."
	reasonBlocked         = "Task is blocked by an unscheduled predecessor."
	reasonZeroCapacity    = "No positive capacity is available for this timeline."
)

type Repository struct {
	database *gorm.DB
	now      func() time.Time
	newID    func() (string, error)
}

func New(database *gorm.DB) *Repository {
	return &Repository{
		database: database,
		now:      func() time.Time { return time.Now().UTC() },
		newID:    identity.NewUUID,
	}
}

func NewWithDependencies(database *gorm.DB, now func() time.Time, newID func() (string, error)) *Repository {
	return &Repository{database: database, now: now, newID: newID}
}

func (repository *Repository) RecalculatePortfolio(ctx context.Context, requestedProjectIDs []string) error {
	ctx, release := persistence.SerializeScheduleMutation(ctx)
	defer release()

	database := persistence.Transaction(ctx, repository.database)
	return database.Transaction(func(transaction *gorm.DB) error {
		if err := persistence.LockScheduleMutation(transaction); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		closureRoots, err := expandRequestedClosureRoots(transaction, requestedProjectIDs)
		if err != nil {
			return err
		}
		state, err := repository.loadState(transaction)
		if err != nil {
			return err
		}
		state.restrictToRequestedClosure(closureRoots)
		if len(state.projects) == 0 {
			return nil
		}
		if err := state.validateOrdering(); err != nil {
			return err
		}
		if err := state.validateEffectiveGraph(); err != nil {
			return err
		}

		execution, err := state.scheduleTimeline(schedulingdomain.Execution)
		if err != nil {
			return err
		}

		commitment, err := state.scheduleTimeline(schedulingdomain.Commitment)
		if err != nil {
			return err
		}
		if err := repository.persist(ctx, transaction, state, execution, commitment, nil); err != nil {
			return err
		}
		return nil
	})
}

func (repository *Repository) RecalculateMemberSchedule(ctx context.Context, memberID string) error {
	database := persistence.Transaction(ctx, repository.database)
	var projectIDs []string
	if err := database.Table("wbs_nodes AS node").
		Joins("JOIN projects AS project ON project.id = node.project_id").
		Where("node.assignee_id = ? AND project.status <> ?", memberID, "closed").
		Distinct().Order("node.project_id ASC").
		Pluck("node.project_id", &projectIDs).Error; err != nil {
		return fmt.Errorf("find projects affected by member capacity: %w", err)
	}
	if len(projectIDs) == 0 {
		return nil
	}
	return repository.RecalculatePortfolio(ctx, projectIDs)
}

type portfolioState struct {
	projects            map[string]projectModel
	projectOrder        []string
	tasks               map[string]taskModel
	leafOrder           map[string]int
	members             map[string]memberModel
	overrides           map[string][]capacityOverrideModel
	holidays            map[string]struct{}
	dependencies        []dependencyModel
	existingAllocations map[schedulingdomain.Timeline]map[string][]allocationModel
	manualBlockers      map[string][]string
	fixedBlockers       map[string][]string
	fixed               map[schedulingdomain.Timeline][]taskModel
}

func (repository *Repository) loadState(database *gorm.DB) (*portfolioState, error) {
	var projects []projectModel
	if err := database.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("status <> ?", "closed").
		Order("priority ASC").
		Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("load active projects for scheduling: %w", err)
	}
	projectByID := make(map[string]projectModel, len(projects))
	projectOrder := make([]string, 0, len(projects))
	for _, project := range projects {
		projectByID[project.ID] = project
		projectOrder = append(projectOrder, project.ID)
	}
	if len(projectOrder) == 0 {
		return &portfolioState{projects: projectByID}, nil
	}

	var tasks []taskModel
	if err := database.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("project_id IN ?", projectOrder).
		Order("project_id ASC").
		Order("parent_key ASC").
		Order("position ASC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("load portfolio tasks: %w", err)
	}
	taskByID := make(map[string]taskModel, len(tasks))
	for _, task := range tasks {
		taskByID[task.ID] = task
	}
	if err := resolveEffectiveTaskSettings(projectByID, tasks, taskByID); err != nil {
		return nil, err
	}
	tasks = tasks[:0]
	for _, task := range taskByID {
		tasks = append(tasks, task)
	}
	leafOrder, err := depthFirstLeafOrder(tasks)
	if err != nil {
		return nil, err
	}

	var members []memberModel
	if err := database.Where("deleted_at IS NULL").Find(&members).Error; err != nil {
		return nil, fmt.Errorf("load scheduling members: %w", err)
	}
	memberByID := make(map[string]memberModel, len(members))
	for _, member := range members {
		memberByID[member.ID] = member
	}

	var overrides []capacityOverrideModel
	if err := database.Where("deleted_at IS NULL").Order("start_date ASC").Find(&overrides).Error; err != nil {
		return nil, fmt.Errorf("load capacity overrides: %w", err)
	}
	overridesByMember := make(map[string][]capacityOverrideModel)
	for _, override := range overrides {
		overridesByMember[override.TeamMemberID] = append(overridesByMember[override.TeamMemberID], override)
	}

	var holidayDates []holidayDateModel
	if err := database.Find(&holidayDates).Error; err != nil {
		return nil, fmt.Errorf("load public holiday dates: %w", err)
	}
	holidays := make(map[string]struct{}, len(holidayDates))
	for _, holiday := range holidayDates {
		holidays[schedulingdomain.DateKey(holiday.Date)] = struct{}{}
	}

	var dependencies []dependencyModel
	if err := database.Clauses(clause.Locking{Strength: "UPDATE"}).Find(&dependencies).Error; err != nil {
		return nil, fmt.Errorf("load dependency graph: %w", err)
	}

	var allocationRows []allocationModel
	if err := database.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("task_id IN ?", taskIDs(tasks)).
		Order("timeline ASC").
		Order("allocation_date ASC").
		Order("sequence ASC").
		Find(&allocationRows).Error; err != nil {
		return nil, fmt.Errorf("load existing schedule allocations: %w", err)
	}
	existingAllocations := map[schedulingdomain.Timeline]map[string][]allocationModel{
		schedulingdomain.Execution:  {},
		schedulingdomain.Commitment: {},
		schedulingdomain.Actual:     {},
	}
	for _, allocation := range allocationRows {
		timeline := schedulingdomain.Timeline(allocation.Timeline)
		if timeline != schedulingdomain.Execution && timeline != schedulingdomain.Commitment && timeline != schedulingdomain.Actual {
			return nil, fmt.Errorf("%w: unknown allocation timeline %q", schedulingdomain.ErrDataIntegrity, allocation.Timeline)
		}
		existingAllocations[timeline][allocation.TaskID] = append(existingAllocations[timeline][allocation.TaskID], allocation)
	}

	state := &portfolioState{
		projects:            projectByID,
		projectOrder:        projectOrder,
		tasks:               taskByID,
		leafOrder:           leafOrder,
		members:             memberByID,
		overrides:           overridesByMember,
		holidays:            holidays,
		dependencies:        dependencies,
		existingAllocations: existingAllocations,
		manualBlockers:      make(map[string][]string),
		fixedBlockers:       make(map[string][]string),
		fixed: map[schedulingdomain.Timeline][]taskModel{
			schedulingdomain.Execution:  {},
			schedulingdomain.Commitment: {},
		},
	}
	state.classifyDependencies()
	state.classifyFixedTasks()
	return state, nil
}

func resolveEffectiveTaskSettings(projects map[string]projectModel, tasks []taskModel, taskByID map[string]taskModel) error {
	byProject := make(map[string][]taskModel)
	for _, task := range tasks {
		byProject[task.ProjectID] = append(byProject[task.ProjectID], task)
	}
	for projectID, projectTasks := range byProject {
		project, exists := projects[projectID]
		if !exists {
			return fmt.Errorf("%w: task project %s missing", schedulingdomain.ErrDataIntegrity, projectID)
		}
		nodes := make([]groupscheduling.Node, 0, len(projectTasks))
		for _, task := range projectTasks {
			source := groupscheduling.Source(task.GroupSchedulingSource)
			if source == "" {
				source = groupscheduling.SourceInherit
			}
			status := groupscheduling.LocalStatus(task.GroupLocalStatus)
			if status == "" {
				status = groupscheduling.StatusOpen
			}
			nodes = append(nodes, groupscheduling.Node{ID: task.ID, ParentID: task.ParentID, Name: task.Name, Source: source, AutomaticSchedulingOverride: task.GroupAutomaticScheduling, SchedulingStartDateOverride: task.GroupSchedulingStartDate, LocalStatus: status, LockedAutomaticScheduling: task.GroupLockedAutomaticScheduling, LockedSchedulingStartDate: task.GroupLockedSchedulingStartDate})
		}
		resolver, err := groupscheduling.New(groupscheduling.Project{ID: project.ID, Name: project.Name, Status: project.Status, AutomaticScheduling: project.AutomaticScheduling, SchedulingStartDate: project.SchedulingStartDate}, nodes)
		if err != nil {
			return fmt.Errorf("resolve effective Group scheduling for project %s: %w", projectID, err)
		}
		for _, task := range projectTasks {
			effective, err := resolver.EffectiveFor(task.ID)
			if err != nil {
				return fmt.Errorf("resolve effective Group scheduling for WBS %s: %w", task.ID, err)
			}
			current := taskByID[task.ID]
			current.EffectiveAutomaticScheduling = effective.AutomaticScheduling
			current.EffectiveSchedulingStartDate = effective.SchedulingStartDate
			current.EffectiveLifecycle = string(effective.Lifecycle)
			current.EffectiveLockOwnerID = effective.LockOwnerID
			localLockID, _, lockErr := resolver.LocalLockOwnerFor(task.ID)
			if lockErr != nil {
				return lockErr
			}
			current.LocalGroupLockOwnerID = localLockID
			taskByID[task.ID] = current
		}
	}
	return nil
}

func projectHasEffectiveAutomatic(state *portfolioState, projectID string) bool {
	for taskID, task := range state.tasks {
		if task.ProjectID != projectID || !task.EffectiveAutomaticScheduling || task.EffectiveLifecycle != "open" {
			continue
		}
		if _, leaf := state.leafOrder[taskID]; leaf {
			return true
		}
	}
	return false
}

func taskIDs(tasks []taskModel) []string {
	values := make([]string, 0, len(tasks))
	for _, task := range tasks {
		values = append(values, task.ID)
	}
	return values
}

func (state *portfolioState) classifyDependencies() {
	state.manualBlockers = make(map[string][]string)
	state.fixedBlockers = make(map[string][]string)
	for _, dependency := range state.dependencies {
		blocking, blockingExists := state.tasks[dependency.BlockingTaskID]
		blocked, blockedExists := state.tasks[dependency.BlockedTaskID]
		if !blockingExists || !blockedExists {
			continue
		}
		blockingProject := state.projects[blocking.ProjectID]
		blockedProject := state.projects[blocked.ProjectID]
		if blockingProject.Status == "closed" || blockedProject.Status == "closed" {
			continue
		}
		state.manualBlockers[dependency.BlockedTaskID] = appendUnique(state.manualBlockers[dependency.BlockedTaskID], dependency.BlockingTaskID)
	}
}

func (state *portfolioState) classifyFixedTasks() {
	for _, task := range state.tasks {
		if task.ActualStart != nil && task.ActualEnd != nil || task.EffectiveLifecycle != "open" || !task.EffectiveAutomaticScheduling {
			state.fixed[schedulingdomain.Execution] = append(state.fixed[schedulingdomain.Execution], task)
			state.fixed[schedulingdomain.Commitment] = append(state.fixed[schedulingdomain.Commitment], task)
		}
	}
}

func (state *portfolioState) validateOrdering() error {
	seenPriority := make(map[int]string)
	for _, projectID := range state.projectOrder {
		project := state.projects[projectID]
		if previous, exists := seenPriority[project.Priority]; exists {
			return fmt.Errorf("%w: duplicate project priority %d for %s and %s", schedulingdomain.ErrDataIntegrity, project.Priority, previous, project.ID)
		}
		seenPriority[project.Priority] = project.ID
	}
	return nil
}

func (state *portfolioState) validateEffectiveGraph() error {
	adjacency := make(map[string][]string)
	for blocked, blockers := range state.manualBlockers {
		for _, blocker := range blockers {
			adjacency[blocker] = appendUnique(adjacency[blocker], blocked)
		}
	}
	for blocked, blockers := range state.fixedBlockers {
		for _, blocker := range blockers {
			adjacency[blocker] = appendUnique(adjacency[blocker], blocked)
		}
	}
	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var walk func(string) error
	walk = func(taskID string) error {
		if visiting[taskID] {
			return fmt.Errorf("%w: effective dependency cycle", schedulingdomain.ErrDataIntegrity)
		}
		if visited[taskID] {
			return nil
		}
		visiting[taskID] = true
		for _, next := range adjacency[taskID] {
			if err := walk(next); err != nil {
				return err
			}
		}
		visiting[taskID] = false
		visited[taskID] = true
		return nil
	}
	for taskID := range state.tasks {
		if err := walk(taskID); err != nil {
			return err
		}
	}
	return nil
}

func depthFirstLeafOrder(tasks []taskModel) (map[string]int, error) {
	children := make(map[string][]taskModel)
	for _, task := range tasks {
		key := task.ProjectID + "|" + task.ParentKey
		children[key] = append(children[key], task)
	}
	for key := range children {
		siblings := children[key]
		sort.Slice(siblings, func(left, right int) bool { return siblings[left].Position < siblings[right].Position })
		for index := 1; index < len(siblings); index++ {
			if siblings[index-1].Position == siblings[index].Position {
				return nil, fmt.Errorf("%w: duplicate WBS position", schedulingdomain.ErrDataIntegrity)
			}
		}
		children[key] = siblings
	}
	order := make(map[string]int)
	rank := 0
	var walk func(taskModel)
	walk = func(task taskModel) {
		childValues := children[task.ProjectID+"|"+task.ID]
		if len(childValues) == 0 {
			order[task.ID] = rank
			rank++
			return
		}
		for _, child := range childValues {
			walk(child)
		}
	}
	projects := make(map[string][]taskModel)
	for _, task := range tasks {
		if task.ParentID == nil {
			projects[task.ProjectID] = append(projects[task.ProjectID], task)
		}
	}
	projectIDs := make([]string, 0, len(projects))
	for projectID := range projects {
		projectIDs = append(projectIDs, projectID)
	}
	sort.Strings(projectIDs)
	for _, projectID := range projectIDs {
		roots := projects[projectID]
		sort.Slice(roots, func(left, right int) bool { return roots[left].Position < roots[right].Position })
		for _, root := range roots {
			walk(root)
		}
	}
	return order, nil
}

func (state *portfolioState) isEffectiveBlocker(taskID, candidateBlockerID string) bool {
	visited := make(map[string]bool)
	var walk func(string) bool
	walk = func(current string) bool {
		if visited[current] {
			return false
		}
		visited[current] = true
		for _, blockerID := range state.blockers(current) {
			if blockerID == candidateBlockerID || walk(blockerID) {
				return true
			}
		}
		return false
	}
	return walk(taskID)
}

func appendUnique(values []string, candidate string) []string {
	for _, value := range values {
		if value == candidate {
			return values
		}
	}
	return append(values, candidate)
}

func (state *portfolioState) blockers(taskID string) []string {
	values := append([]string{}, state.manualBlockers[taskID]...)
	for _, blocker := range state.fixedBlockers[taskID] {
		values = appendUnique(values, blocker)
	}
	return values
}

func datePointer(value time.Time) *time.Time {
	date := schedulingdomain.DateOnly(value)
	return &date
}

func stringPointer(value string) *string { return &value }

func compareTaskOrder(state *portfolioState, left, right taskModel) bool {
	leftProject := state.projects[left.ProjectID]
	rightProject := state.projects[right.ProjectID]
	if leftProject.Priority != rightProject.Priority {
		return leftProject.Priority < rightProject.Priority
	}
	return state.leafOrder[left.ID] < state.leafOrder[right.ID]
}

func ratString(value *big.Rat) string {
	if value == nil {
		return "0"
	}
	return value.FloatString(6)
}

func projectIDsFromRequest(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

var _ application.Store = (*Repository)(nil)
