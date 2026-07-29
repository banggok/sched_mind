package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/wbs/application"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Tree(ctx context.Context, p string) ([]domain.Node, error) {
	if _, err := loadProject(r.db.WithContext(ctx), p, false); err != nil {
		return nil, err
	}
	var models []nodeModel
	if err := r.db.WithContext(ctx).Where("project_id = ?", p).Order("parent_key ASC").Order("position ASC").Order("id ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("load WBS tree: %w", err)
	}
	return buildTree(models), nil
}
func (r *Repository) Find(ctx context.Context, p, id string) (*domain.Node, error) {
	var models []nodeModel
	if err := r.db.WithContext(ctx).Where("project_id = ?", p).Order("parent_key ASC").Order("position ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	for _, n := range flatten(buildTree(models)) {
		if n.ID == id {
			return &n, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *Repository) Create(ctx context.Context, id, p string, parent *string, name string, confirm bool, now time.Time, schedule func(context.Context, string) error, invalidate func(context.Context, []string) error) (*domain.Node, error) {
	var result *domain.Node
	err := r.tx(ctx, p, func(tx *gorm.DB, project projectModel) error {
		var parentModel *nodeModel
		var convertParentTask bool
		if parent != nil {
			value, err := findNode(tx, p, *parent, true)
			if err != nil {
				return domain.ErrParentNotFound
			}
			parentModel = &value
			if value.ActualEnd != nil {
				return domain.ErrCompletedReadOnly
			}
			hasChildren, err := childCount(tx, p, value.ID)
			if err != nil {
				return err
			}
			dependencies, err := dependencyCount(tx, value.ID)
			if err != nil {
				return err
			}
			convertParentTask = hasChildren == 0
			if hasChildren == 0 && (hasData(value) || dependencies > 0) {
				if !confirm {
					return domain.ErrConversionRequired
				}
			}
		}
		position, err := nextPosition(tx, p, parentKey(parent))
		if err != nil {
			return err
		}
		value, err := domain.New(id, p, parent, name, position, now)
		if err != nil {
			return err
		}
		model := fromDomain(*value)
		var convertedTaskID string
		if parentModel != nil && convertParentTask {
			convertedTaskID = parentModel.ID
			if hasData(*parentModel) {
				copyExecutable(&model, *parentModel)
				clearExecutable(parentModel)
				if err := tx.Model(&nodeModel{}).Where("id = ?", parentModel.ID).Updates(executableUpdates(*parentModel, now)).Error; err != nil {
					return err
				}
			}
		}
		if err := tx.Create(&model).Error; err != nil {
			return mapConflict(err)
		}
		if convertedTaskID != "" {
			if err := retargetDependencies(tx, convertedTaskID, model.ID); err != nil {
				return err
			}
			if err := invalidateDependencyProjects(ctx, tx, model.ID, invalidate); err != nil {
				return err
			}
		}
		if project.AutomaticScheduling {
			if err := schedule(ctx, p); err != nil {
				return err
			}
		}
		loaded := toDomain(model, false)
		result = &loaded
		return nil
	})
	return result, err
}

func (r *Repository) Rename(ctx context.Context, p, id, name string, now time.Time) (*domain.Node, error) {
	var out *domain.Node
	err := r.tx(ctx, p, func(tx *gorm.DB, _ projectModel) error {
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		n := toDomain(m, false)
		if err := n.Rename(name, now); err != nil {
			return err
		}
		res := tx.Model(&nodeModel{}).Where("id = ?", id).Updates(map[string]any{"name": n.Name, "name_key": domain.NameKey(n.Name), "updated_at": now})
		if res.Error != nil {
			return mapConflict(res.Error)
		}
		out = &n
		return nil
	})
	return out, err
}

func (r *Repository) UpdateExecutable(ctx context.Context, p, id string, input application.WriteExecutableInput, now time.Time, schedule func(context.Context, string) error) (*domain.Node, error) {
	var out *domain.Node
	err := r.tx(ctx, p, func(tx *gorm.DB, project projectModel) error {
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		count, err := childCount(tx, p, id)
		if err != nil {
			return err
		}
		n := toDomain(m, count > 0)
		fields := domain.ExecutableFields{RoleID: input.RoleID, AssigneeID: input.AssigneeID, EffortMinutes: input.EffortMinutes, ExecutionTimeline: input.Execution, CommitmentTimeline: input.Commitment, ActualEnd: n.Executable.ActualEnd}
		if input.Name != nil {
			if err := n.Rename(*input.Name, now); err != nil {
				return err
			}
		}
		if err := validateMember(tx, fields); err != nil {
			return err
		}
		oldAssignee, oldEffort := n.Executable.AssigneeID, n.Executable.EffortMinutes
		if err := n.UpdateExecutable(fields, project.AutomaticScheduling, project.Status == "open", now); err != nil {
			return err
		}
		applyFields(&m, n.Executable)
		updates := executableUpdates(m, now)
		updates["name"] = n.Name
		updates["name_key"] = domain.NameKey(n.Name)
		if err := tx.Model(&nodeModel{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return mapConflict(err)
		}
		if project.AutomaticScheduling && (differentString(oldAssignee, n.Executable.AssigneeID) || differentInt(oldEffort, n.Executable.EffortMinutes)) {
			if err := schedule(ctx, p); err != nil {
				return err
			}
		}
		out = &n
		return nil
	})
	return out, err
}
func (r *Repository) Complete(ctx context.Context, p, id string, actual, now time.Time, forecast func(context.Context, string) error) (*domain.Node, error) {
	var out *domain.Node
	err := r.tx(ctx, p, func(tx *gorm.DB, _ projectModel) error {
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		count, _ := childCount(tx, p, id)
		n := toDomain(m, count > 0)
		if err := n.Complete(actual, now); err != nil {
			return err
		}
		m.ActualEnd = n.Executable.ActualEnd
		if err := tx.Model(&nodeModel{}).Where("id = ?", id).Updates(map[string]any{"actual_end": m.ActualEnd, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := forecast(ctx, p); err != nil {
			return err
		}
		out = &n
		return nil
	})
	return out, err
}

func (r *Repository) Reopen(ctx context.Context, p, id string, now time.Time, forecast func(context.Context, string) error) (*domain.Node, error) {
	var out *domain.Node
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Read the Task state before serialising on the owning Project. Requests
		// that observed the completed state but lose the conditional transition
		// are reported as concurrent conflicts rather than as initially unfinished.
		m, err := findNode(tx, p, id, false)
		if err != nil {
			return err
		}
		count, err := childCount(tx, p, id)
		if err != nil {
			return err
		}
		project, err := loadProject(tx, p, true)
		if err != nil {
			return err
		}
		if project.Status == "closed" {
			return domain.ErrProjectClosedReadOnly
		}
		n := toDomain(m, count > 0)
		if err := n.Reopen(now); err != nil {
			return err
		}
		result := tx.Model(&nodeModel{}).
			Where("id = ? AND project_id = ? AND actual_end IS NOT NULL", id, p).
			Updates(map[string]any{"actual_end": nil, "updated_at": now})
		if result.Error != nil {
			if isReopenConcurrencyError(tx, result.Error) {
				return fmt.Errorf("%w: %v", domain.ErrTaskReopenConflict, result.Error)
			}
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrTaskReopenConflict
		}
		if err := forecast(ctx, p); err != nil {
			return err
		}
		out = &n
		return nil
	})
	return out, err
}

func isReopenConcurrencyError(db *gorm.DB, err error) bool {
	if db.Dialector.Name() != "sqlite" {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "database is locked") || strings.Contains(message, "database table is locked")
}

func (r *Repository) Reorder(ctx context.Context, p, id string, d domain.Direction, now time.Time, schedule func(context.Context, string) error) error {
	return r.tx(ctx, p, func(tx *gorm.DB, project projectModel) error {
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		var siblings []nodeModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("project_id = ? AND parent_key = ?", p, m.ParentKey).Order("position ASC").Find(&siblings).Error; err != nil {
			return err
		}
		idx := -1
		for i := range siblings {
			if siblings[i].ID == id {
				idx = i
			}
		}
		other := idx - 1
		if d == domain.MoveDown {
			other = idx + 1
		}
		if idx < 0 || other < 0 || other >= len(siblings) {
			return domain.ErrMoveNotAllowed
		}
		a, b := siblings[idx], siblings[other]
		tmp := len(siblings) + 1
		if err := tx.Model(&nodeModel{}).Where("id = ?", a.ID).Update("position", tmp).Error; err != nil {
			return err
		}
		if err := tx.Model(&nodeModel{}).Where("id = ?", b.ID).Update("position", a.Position).Error; err != nil {
			return err
		}
		if err := tx.Model(&nodeModel{}).Where("id = ?", a.ID).Updates(map[string]any{"position": b.Position, "updated_at": now}).Error; err != nil {
			return err
		}
		if project.AutomaticScheduling {
			return schedule(ctx, p)
		}
		return nil
	})
}

func (r *Repository) Move(ctx context.Context, p, id, conversionID string, parent *string, confirm bool, now time.Time, schedule func(context.Context, string) error, invalidate func(context.Context, []string) error) error {
	return r.tx(ctx, p, func(tx *gorm.DB, project projectModel) error {
		all, err := loadLocked(tx, p)
		if err != nil {
			return err
		}
		moving, ok := all[id]
		if !ok {
			return domain.ErrNotFound
		}
		if parent != nil {
			if *parent == id || isDescendant(all, id, *parent) {
				return domain.ErrCycle
			}
			dest, ok := all[*parent]
			if !ok {
				return domain.ErrParentNotFound
			}
			destCount, _ := childCount(tx, p, dest.ID)
			dependencies, err := dependencyCount(tx, dest.ID)
			if err != nil {
				return err
			}
			if destCount == 0 && (hasData(dest) || dependencies > 0) {
				if !confirm {
					return domain.ErrConversionRequired
				}
				if err := r.convertDestination(tx, p, conversionID, &dest, now); err != nil {
					return err
				}
				if err := invalidateDependencyProjects(ctx, tx, conversionID, invalidate); err != nil {
					return err
				}
			}
		}
		newKey := parentKey(parent)
		position, err := nextPosition(tx, p, newKey)
		if err != nil {
			return err
		}
		if err := tx.Model(&nodeModel{}).Where("id = ?", id).Updates(map[string]any{"parent_id": parent, "parent_key": newKey, "position": position, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := compact(tx, p, moving.ParentKey); err != nil {
			return err
		}
		if project.AutomaticScheduling {
			return schedule(ctx, p)
		}
		return nil
	})
}
func (r *Repository) Delete(ctx context.Context, p, id string, now time.Time, schedule func(context.Context, string) error, invalidate func(context.Context, []string) error) error {
	return r.tx(ctx, p, func(tx *gorm.DB, project projectModel) error {
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		if m.ActualEnd != nil {
			return domain.ErrCompletedReadOnly
		}
		count, err := childCount(tx, p, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return domain.ErrHasChildren
		}
		dependencyProjects, err := dependencyProjectIDs(tx, id)
		if err != nil {
			return err
		}
		if err := tx.Where("blocking_task_id = ? OR blocked_task_id = ?", id, id).Delete(&dependencyLinkModel{}).Error; err != nil {
			return err
		}
		if err := invalidateProjectsIfAutomatic(ctx, tx, dependencyProjects, invalidate); err != nil {
			return err
		}
		if err := tx.Where("id = ?", id).Delete(&nodeModel{}).Error; err != nil {
			return err
		}
		if err := compact(tx, p, m.ParentKey); err != nil {
			return err
		}
		if project.AutomaticScheduling {
			return schedule(ctx, p)
		}
		return nil
	})
}

func (r *Repository) tx(ctx context.Context, p string, fn func(*gorm.DB, projectModel) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		project, err := loadProject(tx, p, true)
		if err != nil {
			return err
		}
		if project.Status == "closed" {
			return domain.ErrProjectClosedReadOnly
		}
		return fn(tx, project)
	})
}
func loadProject(db *gorm.DB, id string, lock bool) (projectModel, error) {
	var m projectModel
	q := db
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m, domain.ErrProjectNotFound
	}
	return m, err
}
func findNode(db *gorm.DB, p, id string, lock bool) (nodeModel, error) {
	var m nodeModel
	q := db
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := q.Where("project_id = ?", p).First(&m, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return m, domain.ErrNotFound
	}
	return m, err
}
func childCount(db *gorm.DB, p, id string) (int64, error) {
	var n int64
	err := db.Model(&nodeModel{}).Where("project_id = ? AND parent_key = ?", p, id).Count(&n).Error
	return n, err
}
func dependencyCount(db *gorm.DB, taskID string) (int64, error) {
	var count int64
	err := db.Model(&dependencyLinkModel{}).Where("blocking_task_id = ? OR blocked_task_id = ?", taskID, taskID).Count(&count).Error
	return count, err
}
func nextPosition(db *gorm.DB, p, key string) (int, error) {
	var max int
	if err := db.Model(&nodeModel{}).Where("project_id = ? AND parent_key = ?", p, key).Select("COALESCE(MAX(position),0)").Scan(&max).Error; err != nil {
		return 0, err
	}
	return max + 1, nil
}
func parentKey(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
func hasData(m nodeModel) bool {
	return m.RoleID != nil || m.AssigneeID != nil || m.EffortMinutes != nil || m.ExecutionStart != nil || m.ExecutionEnd != nil || m.CommitmentStart != nil || m.CommitmentEnd != nil || m.ActualEnd != nil
}
func clearExecutable(m *nodeModel) {
	m.RoleID = nil
	m.AssigneeID = nil
	m.EffortMinutes = nil
	m.ExecutionStart = nil
	m.ExecutionEnd = nil
	m.CommitmentStart = nil
	m.CommitmentEnd = nil
	m.ActualEnd = nil
}
func copyExecutable(dst *nodeModel, src nodeModel) {
	dst.RoleID = src.RoleID
	dst.AssigneeID = src.AssigneeID
	dst.EffortMinutes = src.EffortMinutes
	dst.ExecutionStart = src.ExecutionStart
	dst.ExecutionEnd = src.ExecutionEnd
	dst.CommitmentStart = src.CommitmentStart
	dst.CommitmentEnd = src.CommitmentEnd
	dst.ActualEnd = src.ActualEnd
}
func executableUpdates(m nodeModel, now time.Time) map[string]any {
	return map[string]any{"role_id": m.RoleID, "assignee_id": m.AssigneeID, "effort_minutes": m.EffortMinutes, "execution_start": m.ExecutionStart, "execution_end": m.ExecutionEnd, "commitment_start": m.CommitmentStart, "commitment_end": m.CommitmentEnd, "actual_end": m.ActualEnd, "updated_at": now}
}
func (r *Repository) convertDestination(tx *gorm.DB, p, conversionID string, dest *nodeModel, now time.Time) error {
	name := dest.Name
	for i := 1; ; i++ {
		var count int64
		key := domain.NameKey(name)
		if err := tx.Model(&nodeModel{}).Where("project_id = ? AND parent_key = ? AND name_key = ?", p, dest.ID, key).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			child := nodeModel{ID: conversionID, ProjectID: p, ParentID: &dest.ID, ParentKey: dest.ID, Name: name, NameKey: key, Position: 1, CreatedAt: now, UpdatedAt: now}
			copyExecutable(&child, *dest)
			if err := tx.Create(&child).Error; err != nil {
				return err
			}
			if err := retargetDependencies(tx, dest.ID, child.ID); err != nil {
				return err
			}
			clearExecutable(dest)
			return tx.Model(&nodeModel{}).Where("id = ?", dest.ID).Updates(executableUpdates(*dest, now)).Error
		}
		name = fmt.Sprintf("%s (converted %d)", dest.Name, i)
	}
}

func retargetDependencies(tx *gorm.DB, fromTaskID, toTaskID string) error {
	if err := tx.Model(&dependencyLinkModel{}).Where("blocking_task_id = ?", fromTaskID).Update("blocking_task_id", toTaskID).Error; err != nil {
		return fmt.Errorf("retarget blocking dependencies: %w", err)
	}
	if err := tx.Model(&dependencyLinkModel{}).Where("blocked_task_id = ?", fromTaskID).Update("blocked_task_id", toTaskID).Error; err != nil {
		return fmt.Errorf("retarget blocked dependencies: %w", err)
	}
	return nil
}
func dependencyProjectIDs(tx *gorm.DB, taskID string) ([]string, error) {
	var links []dependencyLinkModel
	if err := tx.Where("blocking_task_id = ? OR blocked_task_id = ?", taskID, taskID).Find(&links).Error; err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return []string{}, nil
	}
	ids, seen := []string{}, map[string]bool{}
	var current nodeModel
	if err := tx.Select("project_id").First(&current, "id = ?", taskID).Error; err != nil {
		return nil, err
	}
	ids = append(ids, current.ProjectID)
	seen[current.ProjectID] = true
	for _, link := range links {
		other := link.BlockingTaskID
		if other == taskID {
			other = link.BlockedTaskID
		}
		var node nodeModel
		if err := tx.Select("project_id").First(&node, "id = ?", other).Error; err != nil {
			return nil, err
		}
		if !seen[node.ProjectID] {
			seen[node.ProjectID] = true
			ids = append(ids, node.ProjectID)
		}
	}
	return ids, nil
}
func invalidateDependencyProjects(ctx context.Context, tx *gorm.DB, taskID string, invalidate func(context.Context, []string) error) error {
	ids, err := dependencyProjectIDs(tx, taskID)
	if err != nil {
		return err
	}
	return invalidateProjectsIfAutomatic(ctx, tx, ids, invalidate)
}
func invalidateProjectsIfAutomatic(ctx context.Context, tx *gorm.DB, ids []string, invalidate func(context.Context, []string) error) error {
	if len(ids) == 0 {
		return nil
	}
	var count int64
	if err := tx.Model(&projectModel{}).Where("id IN ? AND automatic_scheduling = ?", ids, true).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return nil
	}
	return invalidate(ctx, ids)
}
func loadLocked(tx *gorm.DB, p string) (map[string]nodeModel, error) {
	var values []nodeModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("project_id = ?", p).Find(&values).Error; err != nil {
		return nil, err
	}
	out := map[string]nodeModel{}
	for _, v := range values {
		out[v.ID] = v
	}
	return out, nil
}
func isDescendant(all map[string]nodeModel, root, candidate string) bool {
	current := candidate
	for current != "" {
		m, ok := all[current]
		if !ok {
			return false
		}
		if m.ParentID == nil {
			return false
		}
		if *m.ParentID == root {
			return true
		}
		current = *m.ParentID
	}
	return false
}
func compact(tx *gorm.DB, p, key string) error {
	var values []nodeModel
	if err := tx.Where("project_id = ? AND parent_key = ?", p, key).Order("position ASC").Find(&values).Error; err != nil {
		return err
	}
	for i, v := range values {
		if v.Position != i+1 {
			if err := tx.Model(&nodeModel{}).Where("id = ?", v.ID).Update("position", i+1).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
func validateMember(tx *gorm.DB, f domain.ExecutableFields) error {
	if f.AssigneeID == nil {
		return nil
	}
	var m memberModel
	if err := tx.Where("deleted_at IS NULL").First(&m, "id = ?", *f.AssigneeID).Error; err != nil {
		return domain.ErrRoleAssigneeMismatch
	}
	if f.RoleID == nil || *f.RoleID != m.RoleID {
		return domain.ErrRoleAssigneeMismatch
	}
	return nil
}
func mapConflict(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrNameExists
	}
	return err
}
func differentString(a, b *string) bool {
	if a == nil || b == nil {
		return a != nil || b != nil
	}
	return *a != *b
}
func differentInt(a, b *int) bool {
	if a == nil || b == nil {
		return a != nil || b != nil
	}
	return *a != *b
}
func fromDomain(n domain.Node) nodeModel {
	m := nodeModel{ID: n.ID, ProjectID: n.ProjectID, ParentID: n.ParentID, ParentKey: parentKey(n.ParentID), Name: n.Name, NameKey: domain.NameKey(n.Name), Position: n.Position, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}
	applyFields(&m, n.Executable)
	return m
}
func applyFields(m *nodeModel, f domain.ExecutableFields) {
	m.RoleID = f.RoleID
	m.AssigneeID = f.AssigneeID
	m.EffortMinutes = f.EffortMinutes
	m.ExecutionStart = f.ExecutionTimeline.Start
	m.ExecutionEnd = f.ExecutionTimeline.End
	m.CommitmentStart = f.CommitmentTimeline.Start
	m.CommitmentEnd = f.CommitmentTimeline.End
	m.ActualEnd = f.ActualEnd
}
func toDomain(m nodeModel, children bool) domain.Node {
	return domain.Node{ID: m.ID, ProjectID: m.ProjectID, ParentID: m.ParentID, Name: m.Name, Position: m.Position, HasChildren: children, Executable: domain.ExecutableFields{RoleID: m.RoleID, AssigneeID: m.AssigneeID, EffortMinutes: m.EffortMinutes, ExecutionTimeline: domain.Timeline{Start: m.ExecutionStart, End: m.ExecutionEnd}, CommitmentTimeline: domain.Timeline{Start: m.CommitmentStart, End: m.CommitmentEnd}, ActualEnd: m.ActualEnd}, Children: []domain.Node{}, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
func buildTree(models []nodeModel) []domain.Node {
	children := map[string][]nodeModel{}
	for _, m := range models {
		children[m.ParentKey] = append(children[m.ParentKey], m)
	}
	var walk func(string) []domain.Node
	walk = func(key string) []domain.Node {
		out := []domain.Node{}
		for _, m := range children[key] {
			n := toDomain(m, len(children[m.ID]) > 0)
			n.Children = walk(m.ID)
			out = append(out, n)
		}
		return out
	}
	return walk("")
}
func flatten(nodes []domain.Node) []domain.Node {
	out := []domain.Node{}
	for _, n := range nodes {
		out = append(out, n)
		out = append(out, flatten(n.Children)...)
	}
	return out
}

var _ application.Store = (*Repository)(nil)
