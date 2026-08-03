package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/banggok/sched_mind/backend/internal/dependencies/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	db      *gorm.DB
	graphMu sync.Mutex
}

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

type dependencyModel struct {
	ID, BlockingTaskID, BlockedTaskID string
	CreatedAt, UpdatedAt              time.Time
}

func (dependencyModel) TableName() string { return "task_dependencies" }

type taskModel struct {
	ID, ProjectID, ParentKey, Name, NameKey string
	ParentID                                *string
	Position                                int
	ActualStart, ActualEnd, ExecutionStart  *time.Time
}

type hierarchyNode struct {
	ID        string
	ParentKey string
	Name      string
}

func (taskModel) TableName() string { return "wbs_nodes" }

type projectModel struct {
	ID, Name, NameKey, Status string
	AutomaticScheduling       bool
}

func (projectModel) TableName() string { return "projects" }

func (r *Repository) List(ctx context.Context, taskID string) (*domain.Detail, error) {
	if _, err := loadTask(r.db.WithContext(ctx), taskID); err != nil {
		return nil, err
	}
	var incoming, outgoing []dependencyRow
	if err := r.db.WithContext(ctx).Table("task_dependencies d").
		Select("d.id,d.blocking_task_id,d.blocked_task_id,d.created_at,d.updated_at,t.id task_id,t.name task_name,t.project_id,p.name project_name,t.actual_start,t.actual_end,t.execution_start expected_start").
		Joins("JOIN wbs_nodes t ON t.id = d.blocking_task_id").Joins("JOIN projects p ON p.id = t.project_id").
		Where("d.blocked_task_id = ?", taskID).Order("LOWER(p.name) ASC").Order("t.parent_key ASC").Order("t.position ASC").Order("t.id ASC").Scan(&incoming).Error; err != nil {
		return nil, fmt.Errorf("load blocked by: %w", err)
	}
	if err := r.db.WithContext(ctx).Table("task_dependencies d").
		Select("d.id,d.blocking_task_id,d.blocked_task_id,d.created_at,d.updated_at,t.id task_id,t.name task_name,t.project_id,p.name project_name,t.actual_start,t.actual_end,t.execution_start expected_start").
		Joins("JOIN wbs_nodes t ON t.id = d.blocked_task_id").Joins("JOIN projects p ON p.id = t.project_id").
		Where("d.blocking_task_id = ?", taskID).Order("LOWER(p.name) ASC").Order("t.parent_key ASC").Order("t.position ASC").Order("t.id ASC").Scan(&outgoing).Error; err != nil {
		return nil, fmt.Errorf("load blocks: %w", err)
	}
	return &domain.Detail{BlockedBy: mapDependencyRows(incoming), Blocks: mapDependencyRows(outgoing)}, nil
}

// dependencyRow is kept package-level so DTO mapping remains explicit.
type dependencyRow struct {
	ID, BlockingTaskID, BlockedTaskID, TaskID, TaskName, ProjectID, ProjectName string
	CreatedAt, UpdatedAt                                                        time.Time
	ActualStart, ActualEnd, ExpectedStart                                       *time.Time
}

func mapDependencyRows(rows []dependencyRow) []domain.Item {
	out := make([]domain.Item, 0, len(rows))
	for _, v := range rows {
		out = append(out, domain.Item{Dependency: domain.Dependency{ID: v.ID, BlockingTaskID: v.BlockingTaskID, BlockedTaskID: v.BlockedTaskID, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}, Task: domain.Task{ID: v.TaskID, Name: v.TaskName, ProjectID: v.ProjectID, ProjectName: v.ProjectName, ActualStart: v.ActualStart, ActualEnd: v.ActualEnd, ExpectedStart: v.ExpectedStart}})
	}
	return out
}

func (r *Repository) Candidates(ctx context.Context, taskID string, direction domain.Direction, search string, page, pageSize int) (*domain.CandidatePage, error) {
	current, err := validateEndpoint(r.db.WithContext(ctx), taskID, direction == domain.BlockedBy)
	if err != nil {
		return nil, err
	}
	q := r.db.WithContext(ctx).Table("wbs_nodes t").Joins("JOIN projects p ON p.id = t.project_id").
		Where("t.id <> ? AND p.status <> ?", taskID, "closed").
		Where("NOT EXISTS (?)", r.db.Table("wbs_nodes c").Select("1").Where("c.project_id = t.project_id AND c.parent_id = t.id"))
	if direction == domain.Blocks {
		q = q.Where("t.actual_start IS NULL AND t.actual_end IS NULL").Where("NOT EXISTS (?)", r.db.Table("task_dependencies d").Select("1").Where("d.blocking_task_id = ? AND d.blocked_task_id = t.id", taskID))
	} else {
		q = q.Where("NOT EXISTS (?)", r.db.Table("task_dependencies d").Select("1").Where("d.blocking_task_id = t.id AND d.blocked_task_id = ?", taskID))
	}
	term := strings.TrimSpace(search)
	if term != "" {
		like := "%" + persistence.EscapeLike(strings.ToLower(term)) + "%"
		q = q.Where("(t.name_key LIKE ? ESCAPE '\\' OR p.name_key LIKE ? ESCAPE '\\')", like, like)
	}
	var rows []struct {
		ID, Name, ProjectID, ProjectName, ParentKey string
		ActualStart, ActualEnd, ExpectedStart       *time.Time
	}
	if err := q.Select("t.id,t.name,t.project_id,p.name project_name,t.parent_key,t.actual_start,t.actual_end,t.execution_start expected_start").Order("p.name_key ASC").Order("t.parent_key ASC").Order("t.position ASC").Order("t.id ASC").Scan(&rows).Error; err != nil {
		return nil, err
	}

	projectIDs := make([]string, 0)
	seenProjectIDs := make(map[string]struct{})

	for _, row := range rows {
		if _, exists := seenProjectIDs[row.ProjectID]; exists {
			continue
		}

		seenProjectIDs[row.ProjectID] = struct{}{}
		projectIDs = append(projectIDs, row.ProjectID)
	}

	nodesByProject := make(map[string]map[string]hierarchyNode)

	if len(projectIDs) > 0 {
		var hierarchyRows []struct {
			ID        string
			ProjectID string
			ParentKey string
			Name      string
		}

		if err := r.db.WithContext(ctx).
			Table("wbs_nodes").
			Select("id,project_id,parent_key,name").
			Where("project_id IN ?", projectIDs).
			Scan(&hierarchyRows).Error; err != nil {
			return nil, fmt.Errorf("load dependency candidate hierarchy: %w", err)
		}

		for _, row := range hierarchyRows {
			nodes := nodesByProject[row.ProjectID]
			if nodes == nil {
				nodes = make(map[string]hierarchyNode)
				nodesByProject[row.ProjectID] = nodes
			}

			nodes[row.ID] = hierarchyNode{
				ID:        row.ID,
				ParentKey: row.ParentKey,
				Name:      row.Name,
			}
		}
	}

	var edges []dependencyModel
	if err := r.db.WithContext(ctx).Find(&edges).Error; err != nil {
		return nil, err
	}
	adjacency := adjacencyMap(edges)
	valid := make([]domain.Task, 0, len(rows))
	for _, v := range rows {
		pathStart, pathTarget := v.ID, current.ID
		if direction == domain.BlockedBy {
			pathStart, pathTarget = current.ID, v.ID
		}
		if reachable(adjacency, pathStart, pathTarget) {
			continue
		}
		valid = append(valid, domain.Task{
			ID:          v.ID,
			Name:        v.Name,
			ProjectID:   v.ProjectID,
			ProjectName: v.ProjectName,
			HierarchyPath: buildHierarchyPath(
				v.ProjectName,
				v.ID,
				nodesByProject[v.ProjectID],
			),
			ActualStart:   v.ActualStart,
			ActualEnd:     v.ActualEnd,
			ExpectedStart: v.ExpectedStart,
		})
	}
	total := int64(len(valid))
	start := (page - 1) * pageSize
	if start > len(valid) {
		start = len(valid)
	}
	end := start + pageSize
	if end > len(valid) {
		end = len(valid)
	}
	return &domain.CandidatePage{Items: valid[start:end], Page: page, PageSize: pageSize, TotalItems: total}, nil
}

func (r *Repository) Create(ctx context.Context, value domain.Dependency, schedule func(context.Context, []string) error) (*domain.Dependency, error) {
	ctx, release := persistence.SerializeScheduleMutation(ctx)
	defer release()
	r.graphMu.Lock()
	defer r.graphMu.Unlock()
	var out *domain.Dependency
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := persistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		projects, err := lockActiveProjects(tx)
		if err != nil {
			return err
		}
		blocking, err := validateEndpoint(tx, value.BlockingTaskID, false)
		if err != nil {
			return err
		}
		blocked, err := validateEndpoint(tx, value.BlockedTaskID, true)
		if err != nil {
			return err
		}
		if blocking.ID == blocked.ID {
			return domain.ErrSelfReference
		}

		var existing dependencyModel
		err = tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("blocking_task_id = ? AND blocked_task_id = ?", blocking.ID, blocked.ID).
			First(&existing).Error
		if err == nil {
			return domain.ErrAlreadyExists
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		path, err := cyclePath(tx, blocked.ID, blocking.ID)
		if err != nil {
			return err
		}
		if len(path) > 0 {
			steps, err := cycleSteps(tx, append(path, blocked.ID))
			if err != nil {
				return err
			}
			return &domain.CycleError{Path: steps}
		}
		m := dependencyModel{
			ID: value.ID, BlockingTaskID: value.BlockingTaskID, BlockedTaskID: value.BlockedTaskID,
			CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt,
		}
		if err := tx.Create(&m).Error; err != nil {
			return mapConflict(err)
		}
		if projects[blocking.ProjectID] || projects[blocked.ProjectID] {
			if err := schedule(dependencyScheduleContext(ctx, tx, blocked.ProjectID), uniqueStrings(blocking.ProjectID, blocked.ProjectID)); err != nil {
				return err
			}
		}
		copy := value
		out = &copy
		return nil
	})
	return out, err
}

func (r *Repository) Delete(ctx context.Context, id string, now time.Time, schedule func(context.Context, []string) error) error {
	ctx, release := persistence.SerializeScheduleMutation(ctx)
	defer release()
	r.graphMu.Lock()
	defer r.graphMu.Unlock()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := persistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		projects, err := lockActiveProjects(tx)
		if err != nil {
			return err
		}
		m, _, blocking, blocked, err := loadOwnedDependency(tx, id)
		if err != nil {
			return err
		}
		if err := ensureEndpointProjectsOpen(tx, blocking, blocked); err != nil {
			return err
		}
		if blocked.ActualStart != nil && blocked.ActualEnd != nil {
			return domain.ErrCompletedHistory
		}
		if err := tx.Delete(&m).Error; err != nil {
			return err
		}
		if projects[blocking.ProjectID] || projects[blocked.ProjectID] {
			return schedule(dependencyScheduleContext(ctx, tx, blocked.ProjectID), uniqueStrings(blocking.ProjectID, blocked.ProjectID))
		}
		return nil
	})
}

func loadOwnedDependency(tx *gorm.DB, id string) (dependencyModel, *domain.Dependency, taskModel, taskModel, error) {
	var model dependencyModel
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model, nil, taskModel{}, taskModel{}, domain.ErrNotFound
	}
	if err != nil {
		return model, nil, taskModel{}, taskModel{}, err
	}
	dependency, err := domain.Rehydrate(model.ID, model.BlockingTaskID, model.BlockedTaskID, model.CreatedAt, model.UpdatedAt)
	if err != nil {
		return model, nil, taskModel{}, taskModel{}, err
	}
	blocking, err := loadTask(tx, model.BlockingTaskID)
	if err != nil {
		return model, nil, taskModel{}, taskModel{}, err
	}
	blocked, err := loadTask(tx, model.BlockedTaskID)
	if err != nil {
		return model, nil, taskModel{}, taskModel{}, err
	}
	return model, dependency, blocking, blocked, nil
}

func validateEndpoint(tx *gorm.DB, id string, blocked bool) (taskModel, error) {
	t, err := loadTask(tx, id)
	if err != nil {
		return t, err
	}
	var p projectModel
	if err := tx.First(&p, "id = ?", t.ProjectID).Error; err != nil {
		return t, err
	}
	if p.Status == "closed" {
		return t, domain.ErrClosedProject
	}
	if p.Status != "open" {
		return t, domain.ErrLockedProject
	}
	var children int64
	if err := tx.Model(&taskModel{}).Where("project_id = ? AND parent_key = ?", t.ProjectID, t.ID).Count(&children).Error; err != nil {
		return t, err
	}
	if children > 0 {
		return t, domain.ErrExecutableNeeded
	}
	if blocked && t.ActualStart != nil && t.ActualEnd != nil {
		return t, domain.ErrCompletedBlocked
	}
	return t, nil
}

func ensureEndpointProjectsOpen(tx *gorm.DB, tasks ...taskModel) error {
	for _, task := range tasks {
		var project projectModel
		if err := tx.First(&project, "id = ?", task.ProjectID).Error; err != nil {
			return err
		}
		if project.Status == "closed" {
			return domain.ErrClosedProject
		}
		if project.Status != "open" {
			return domain.ErrLockedProject
		}
	}
	return nil
}

func loadTask(db *gorm.DB, id string) (taskModel, error) {
	var t taskModel
	err := db.First(&t, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return t, domain.ErrTaskNotFound
	}
	return t, err
}

func lockActiveProjects(tx *gorm.DB) (map[string]bool, error) {
	var rows []projectModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status <> ?", "closed").Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, p := range rows {
		out[p.ID] = p.AutomaticScheduling
	}
	return out, nil
}

func cyclePath(tx *gorm.DB, start, target string) ([]string, error) {
	var edges []dependencyModel
	if err := tx.Find(&edges).Error; err != nil {
		return nil, err
	}
	adj := adjacencyMap(edges)
	queue := [][]string{{start}}
	seen := map[string]bool{start: true}
	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]
		last := path[len(path)-1]
		if last == target {
			return path, nil
		}
		for _, next := range adj[last] {
			if !seen[next] {
				seen[next] = true
				copy := append([]string{}, path...)
				queue = append(queue, append(copy, next))
			}
		}
	}
	return nil, nil
}

func adjacencyMap(edges []dependencyModel) map[string][]string {
	out := map[string][]string{}
	for _, edge := range edges {
		out[edge.BlockingTaskID] = append(out[edge.BlockingTaskID], edge.BlockedTaskID)
	}
	return out
}

func reachable(adjacency map[string][]string, start, target string) bool {
	queue := []string{start}
	seen := map[string]bool{start: true}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current == target {
			return true
		}
		for _, next := range adjacency[current] {
			if !seen[next] {
				seen[next] = true
				queue = append(queue, next)
			}
		}
	}
	return false
}

func cycleSteps(tx *gorm.DB, ids []string) ([]domain.CycleStep, error) {
	var rows []struct{ ID, Name, ProjectName string }
	if err := tx.Table("wbs_nodes t").Select("t.id,t.name,p.name project_name").Joins("JOIN projects p ON p.id=t.project_id").Where("t.id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	byID := map[string]domain.CycleStep{}
	for _, r := range rows {
		byID[r.ID] = domain.CycleStep{TaskID: r.ID, TaskName: r.Name, ProjectName: r.ProjectName}
	}
	out := make([]domain.CycleStep, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out, nil
}

func dependencyScheduleContext(ctx context.Context, tx *gorm.DB, ownerProjectID string) context.Context {
	ctx = persistence.WithTransaction(ctx, tx)
	return schedulingimpact.WithOperation(ctx, ownerProjectID, schedulingimpact.ModeOrdinary)
}

func uniqueStrings(a, b string) []string {
	if a == b {
		return []string{a}
	}
	return []string{a, b}
}

func mapConflict(err error) error {
	if strings.Contains(strings.ToLower(err.Error()), "unique") {
		return domain.ErrAlreadyExists
	}
	return err
}

func buildHierarchyPath(
	projectName string,
	taskID string,
	nodes map[string]hierarchyNode,
) string {
	names := make([]string, 0)
	visited := make(map[string]struct{})
	currentID := taskID

	for currentID != "" {
		if _, exists := visited[currentID]; exists {
			break
		}
		visited[currentID] = struct{}{}

		node, exists := nodes[currentID]
		if !exists {
			break
		}

		names = append(names, node.Name)
		currentID = node.ParentKey
	}

	for left, right := 0, len(names)-1; left < right; left, right = left+1, right-1 {
		names[left], names[right] = names[right], names[left]
	}

	parts := make([]string, 0, len(names)+1)
	parts = append(parts, projectName)
	parts = append(parts, names...)

	return strings.Join(parts, " > ")
}
