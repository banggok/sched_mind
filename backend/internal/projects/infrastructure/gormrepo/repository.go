package gormrepo

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/projects/application"
	"github.com/banggok/sched_mind/backend/internal/projects/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ database *gorm.DB }

func New(database *gorm.DB) *Repository { return &Repository{database: database} }

func (r *Repository) List(ctx context.Context, query listing.Query) (listing.Page[domain.Project], error) {
	base := r.listBase(ctx, query.Search)
	var activeCount, closedCount int64
	if err := base.Where("status IN ?", activeStatuses()).Count(&activeCount).Error; err != nil {
		return listing.Page[domain.Project]{}, fmt.Errorf("count active projects: %w", err)
	}
	if err := r.listBase(ctx, query.Search).Where("status = ?", string(domain.StatusClosed)).Count(&closedCount).Error; err != nil {
		return listing.Page[domain.Project]{}, fmt.Errorf("count closed projects: %w", err)
	}
	offset := (query.Page - 1) * query.PageSize
	models := make([]projectModel, 0, query.PageSize)
	if offset < int(activeCount) {
		part, err := r.listSegment(ctx, query.Search, true, offset, query.PageSize)
		if err != nil {
			return listing.Page[domain.Project]{}, err
		}
		models = append(models, part...)
		if len(models) < query.PageSize {
			part, err = r.listSegment(ctx, query.Search, false, 0, query.PageSize-len(models))
			if err != nil {
				return listing.Page[domain.Project]{}, err
			}
			models = append(models, part...)
		}
	} else {
		part, err := r.listSegment(ctx, query.Search, false, offset-int(activeCount), query.PageSize)
		if err != nil {
			return listing.Page[domain.Project]{}, err
		}
		models = append(models, part...)
	}
	items := make([]domain.Project, 0, len(models))
	for _, model := range models {
		item, err := toDomain(model)
		if err != nil {
			return listing.Page[domain.Project]{}, err
		}
		items = append(items, *item)
	}
	return listing.Page[domain.Project]{Items: items, Page: query.Page, PageSize: query.PageSize, Total: activeCount + closedCount}, nil
}

func (r *Repository) listBase(ctx context.Context, search string) *gorm.DB {
	base := r.database.WithContext(ctx).Model(&projectModel{})
	if value := sharedpersistence.EscapeLike(domain.NormalizedNameKey(search)); value != "" {
		base = base.Where("name_key LIKE ?", value+"%")
	}
	return base
}

func (r *Repository) listSegment(ctx context.Context, search string, active bool, offset, limit int) ([]projectModel, error) {
	statement := r.listBase(ctx, search)
	if active {
		statement = statement.Where("status IN ?", activeStatuses()).Order("priority ASC")
	} else {
		statement = statement.Where("status = ?", string(domain.StatusClosed)).Order("closed_at DESC").Order("id ASC")
	}
	var models []projectModel
	if err := statement.Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list project segment: %w", err)
	}
	return models, nil
}

func activeStatuses() []string {
	return []string{string(domain.StatusOpen), string(domain.StatusLocked)}
}

func (r *Repository) Find(ctx context.Context, id string) (*domain.Project, error) {
	return find(r.database.WithContext(ctx), id, false)
}

func (r *Repository) CreateNext(ctx context.Context, id, name string, automaticScheduling bool, schedulingStartDate *time.Time, projectBuffer int, now time.Time) (*domain.Project, error) {
	for attempt := 0; attempt < 5; attempt++ {
		var created *domain.Project
		err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var nameCount int64
			if err := tx.Model(&projectModel{}).Where("name_key = ?", domain.NormalizedNameKey(name)).Count(&nameCount).Error; err != nil {
				return err
			}
			if nameCount > 0 {
				return domain.ErrNameExists
			}
			var maxPriority int
			if err := tx.Model(&projectModel{}).Select("COALESCE(MAX(priority), 0)").Scan(&maxPriority).Error; err != nil {
				return err
			}
			value, err := domain.NewProject(id, name, maxPriority+1, now)
			if err != nil {
				return err
			}
			if value == nil {
				return errors.New("create project: domain returned nil")
			}
			if err := value.UpdateSettings(automaticScheduling, schedulingStartDate, projectBuffer, now); err != nil {
				return err
			}
			if err := tx.Create(fromDomain(*value)).Error; err != nil {
				return err
			}
			created = value
			return nil
		}, &sql.TxOptions{Isolation: sql.LevelSerializable})
		if err == nil {
			return created, nil
		}
		if errors.Is(err, domain.ErrNameExists) {
			return nil, err
		}
		if !errors.Is(err, gorm.ErrDuplicatedKey) && !strings.Contains(strings.ToLower(err.Error()), "serialize") {
			return nil, fmt.Errorf("create project with next priority: %w", err)
		}
	}
	return nil, errors.New("create project with next priority: concurrent allocation did not settle")
}

func (r *Repository) UpdateDetails(ctx context.Context, id, name string, automaticScheduling bool, schedulingStartDate *time.Time, projectBuffer int, now time.Time, schedule func(context.Context, string) error, markUnscheduled func(context.Context, string, string) error) (*domain.Project, error) {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	var changed *domain.Project
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		value, err := find(tx, id, true)
		if err != nil {
			return err
		}
		if value == nil {
			return errors.New("update project: find returned nil")
		}
		if value.Status == domain.StatusLocked {
			return domain.ErrLockedReadOnly
		}
		wasAutomatic := value.AutomaticScheduling
		previousAnchor := value.SchedulingStartDate
		previousBuffer := value.ProjectBuffer
		if err := value.Rename(name, now); err != nil {
			return err
		}
		if value.AutomaticScheduling != automaticScheduling || !datesEqual(value.SchedulingStartDate, schedulingStartDate) || value.ProjectBuffer != projectBuffer {
			if err := value.UpdateSettings(automaticScheduling, schedulingStartDate, projectBuffer, now); err != nil {
				return err
			}
		}
		result := tx.Model(&projectModel{}).Where("id = ?", id).Updates(map[string]interface{}{"name": value.Name, "name_key": domain.NormalizedNameKey(value.Name), "automatic_scheduling": value.AutomaticScheduling, "scheduling_start_date": value.SchedulingStartDate, "project_buffer": value.ProjectBuffer, "updated_at": value.UpdatedAt})
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrNameExists
		}
		if result.Error != nil {
			return result.Error
		}
		settingsAffectSchedule := !datesEqual(previousAnchor, value.SchedulingStartDate) || previousBuffer != value.ProjectBuffer || wasAutomatic != value.AutomaticScheduling
		if value.AutomaticScheduling && settingsAffectSchedule {
			txContext := sharedpersistence.WithTransaction(ctx, tx)
			if value.SchedulingStartDate == nil {
				if err := markUnscheduled(txContext, id, "Automatic Scheduling requires a Project Scheduling Start Date."); err != nil {
					return fmt.Errorf("mark project schedule unscheduled: %w", err)
				}
			} else if err := schedule(txContext, id); err != nil {
				return fmt.Errorf("recalculate project schedule: %w", err)
			}
		}
		changed = value
		return nil
	})
	if err != nil {
		return nil, err
	}
	return changed, nil
}

func (r *Repository) ChangeStatus(ctx context.Context, id string, target domain.Status, now time.Time, schedule func(context.Context) error) (*domain.Project, error) {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	var changed *domain.Project
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		value, err := find(tx, id, true)
		if err != nil {
			return err
		}
		if value == nil {
			return errors.New("change status: find returned nil")
		}
		if value.Status == domain.StatusLocked && target == domain.StatusOpen {
			plan, err := requiredReopenPlan(tx, id)
			if err != nil {
				return err
			}
			if len(plan.Locked) > 1 {
				return domain.BulkReopenRequiredError{
					RootProjectID: id,
					Locked:        plan.Locked,
					Open:          plan.Open,
					Token:         plan.Token,
				}
			}
		}
		leaves, err := loadLifecycleLeaves(tx, id)
		if err != nil {
			return err
		}
		hasUnfinishedLeaves := false
		hasUnscheduledUnfinishedLeaf := false
		for _, leaf := range leaves {
			completed := leaf.ActualStart != nil && leaf.ActualEnd != nil
			if completed {
				continue
			}
			hasUnfinishedLeaves = true
			if leaf.ExecutionStart == nil || leaf.ExecutionEnd == nil ||
				leaf.CommitmentStart == nil || leaf.CommitmentEnd == nil ||
				leaf.ExecutionUnscheduledReason != nil || leaf.CommitmentUnscheduledReason != nil {
				hasUnscheduledUnfinishedLeaf = true
			}
		}
		if target == domain.StatusLocked && hasUnscheduledUnfinishedLeaf {
			return domain.ErrCannotLockUnscheduled
		}
		var executionSnapshot, commitmentSnapshot *string
		if target == domain.StatusLocked {
			executionSnapshot, err = marshalTimelineSnapshot(leaves, true)
			if err != nil {
				return err
			}
			commitmentSnapshot, err = marshalTimelineSnapshot(leaves, false)
			if err != nil {
				return err
			}
		}
		if err := value.ChangeStatus(target, len(leaves) > 0, hasUnfinishedLeaves, executionSnapshot, commitmentSnapshot, now); err != nil {
			return err
		}
		if err := tx.Model(&projectModel{}).Where("id = ?", value.ID).Updates(statusUpdates(*value)).Error; err != nil {
			return err
		}
		if target != domain.StatusLocked {
			if err := schedule(sharedpersistence.WithTransaction(ctx, tx)); err != nil {
				return fmt.Errorf("schedule active projects after status change: %w", err)
			}
		}
		changed = value
		return nil
	})
	if err != nil {
		return nil, err
	}
	return changed, nil
}

type lifecycleLeaf struct {
	ID                          string
	ExecutionStart              *time.Time
	ExecutionEnd                *time.Time
	CommitmentStart             *time.Time
	CommitmentEnd               *time.Time
	ExecutionUnscheduledReason  *string
	CommitmentUnscheduledReason *string
	ActualStart                 *time.Time
	ActualEnd                   *time.Time
}

type timelineSnapshotEntry struct {
	TaskID string     `json:"taskId"`
	Start  *time.Time `json:"start"`
	End    *time.Time `json:"end"`
}

func loadLifecycleLeaves(database *gorm.DB, projectID string) ([]lifecycleLeaf, error) {
	leaves := make([]lifecycleLeaf, 0)
	childQuery := database.Table("wbs_nodes AS child").Select("1").
		Where("child.project_id = task.project_id AND child.parent_id = task.id")
	err := database.Table("wbs_nodes AS task").
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Select("task.id, task.execution_start, task.execution_end, task.commitment_start, task.commitment_end, task.execution_unscheduled_reason, task.commitment_unscheduled_reason, task.actual_start, task.actual_end").
		Where("task.project_id = ?", projectID).
		Where("NOT EXISTS (?)", childQuery).
		Order("task.id ASC").
		Scan(&leaves).Error
	if err != nil {
		return nil, fmt.Errorf("load project executable leaves: %w", err)
	}
	return leaves, nil
}

func marshalTimelineSnapshot(leaves []lifecycleLeaf, execution bool) (*string, error) {
	entries := make([]timelineSnapshotEntry, 0, len(leaves))
	for _, leaf := range leaves {
		entry := timelineSnapshotEntry{TaskID: leaf.ID}
		if execution {
			entry.Start, entry.End = leaf.ExecutionStart, leaf.ExecutionEnd
		} else {
			entry.Start, entry.End = leaf.CommitmentStart, leaf.CommitmentEnd
		}
		entries = append(entries, entry)
	}
	payload, err := json.Marshal(entries)
	if err != nil {
		return nil, fmt.Errorf("marshal locked timeline snapshot: %w", err)
	}
	value := string(payload)
	return &value, nil
}

func (r *Repository) MovePriority(ctx context.Context, id string, direction domain.PriorityDirection, now time.Time, schedule func(context.Context) error) (*domain.Project, error) {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	var changed *domain.Project
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		var active []projectModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status IN ?", activeStatuses()).Order("priority ASC").Find(&active).Error; err != nil {
			return err
		}
		index := -1
		for candidate := range active {
			if active[candidate].ID == id {
				index = candidate
				break
			}
		}
		if index < 0 {
			var count int64
			if err := tx.Model(&projectModel{}).Where("id = ?", id).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return domain.ErrNotFound
			}
			return domain.ErrPriorityMoveNotAllowed
		}
		neighbour := index - 1
		if direction == domain.PriorityDown {
			neighbour = index + 1
		}
		if neighbour < 0 || neighbour >= len(active) {
			return domain.ErrPriorityMoveNotAllowed
		}
		var maxPriority int
		if err := tx.Model(&projectModel{}).Select("COALESCE(MAX(priority), 0)").Scan(&maxPriority).Error; err != nil {
			return err
		}
		current, other := active[index], active[neighbour]
		if err := tx.Model(&projectModel{}).Where("id = ?", current.ID).Update("priority", maxPriority+1).Error; err != nil {
			return err
		}
		if err := tx.Model(&projectModel{}).Where("id = ?", other.ID).Update("priority", current.Priority).Error; err != nil {
			return err
		}
		if err := tx.Model(&projectModel{}).Where("id = ?", current.ID).Updates(map[string]interface{}{"priority": other.Priority, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := schedule(sharedpersistence.WithTransaction(ctx, tx)); err != nil {
			return fmt.Errorf("schedule active projects: %w", err)
		}
		current.Priority = other.Priority
		current.UpdatedAt = now
		changed, _ = toDomain(current)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return changed, nil
}

func (r *Repository) UpdateSettings(ctx context.Context, id string, automaticScheduling bool, schedulingStartDate *time.Time, projectBuffer int, now time.Time, schedule func(context.Context, string) error, markUnscheduled func(context.Context, string, string) error) (*domain.Project, error) {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	var changed *domain.Project
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		value, err := find(tx, id, true)
		if err != nil {
			return err
		}
		if value == nil {
			return errors.New("update settings: find returned nil")
		}
		wasAutomatic := value.AutomaticScheduling
		previousAnchor := value.SchedulingStartDate
		previousBuffer := value.ProjectBuffer
		if err := value.UpdateSettings(automaticScheduling, schedulingStartDate, projectBuffer, now); err != nil {
			return err
		}
		if err := tx.Model(&projectModel{}).Where("id = ?", id).Updates(map[string]interface{}{"automatic_scheduling": value.AutomaticScheduling, "scheduling_start_date": value.SchedulingStartDate, "project_buffer": value.ProjectBuffer, "updated_at": value.UpdatedAt}).Error; err != nil {
			return err
		}
		settingsAffectSchedule := !datesEqual(previousAnchor, value.SchedulingStartDate) || previousBuffer != value.ProjectBuffer || wasAutomatic != value.AutomaticScheduling
		if value.AutomaticScheduling && settingsAffectSchedule {
			txContext := sharedpersistence.WithTransaction(ctx, tx)
			if value.SchedulingStartDate == nil {
				if err := markUnscheduled(txContext, id, "Automatic Scheduling requires a Project Scheduling Start Date."); err != nil {
					return fmt.Errorf("mark project schedule unscheduled: %w", err)
				}
			} else if err := schedule(txContext, id); err != nil {
				return fmt.Errorf("recalculate project schedule: %w", err)
			}
		}
		changed = value
		return nil
	})
	if err != nil {
		return nil, err
	}
	return changed, nil
}

func (r *Repository) DeleteChildless(ctx context.Context, id string) error {
	return r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		value, err := find(tx, id, true)
		if err != nil {
			return err
		}
		if value == nil {
			return errors.New("delete project: find returned nil")
		}
		if err := value.CanDelete(false); err != nil {
			return err
		}
		result := tx.Where("id = ?", id).Delete(&projectModel{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func find(database *gorm.DB, id string, lock bool) (*domain.Project, error) {
	var model projectModel
	query := database
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	err := query.First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find project: %w", err)
	}
	return toDomain(model)
}

func fromDomain(value domain.Project) *projectModel {
	return &projectModel{ID: value.ID, Name: value.Name, NameKey: domain.NormalizedNameKey(value.Name), Status: string(value.Status), StartDate: value.StartDate, EndDate: value.EndDate, AutoCalculateDate: value.AutoCalculateDate, AutomaticScheduling: value.AutomaticScheduling, SchedulingStartDate: value.SchedulingStartDate, ProjectBuffer: value.ProjectBuffer, Priority: value.Priority, ScheduleVersion: value.ScheduleVersion, ClosedAt: value.ClosedAt, LockedExecutionSnapshot: value.LockedExecutionSnapshot, LockedCommitmentSnapshot: value.LockedCommitmentSnapshot, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func toDomain(model projectModel) (*domain.Project, error) {
	return domain.Rehydrate(model.ID, model.Name, domain.Status(model.Status), model.StartDate, model.EndDate, model.AutoCalculateDate, model.AutomaticScheduling, model.SchedulingStartDate, model.ProjectBuffer, model.Priority, model.ScheduleVersion, model.ClosedAt, model.LockedExecutionSnapshot, model.LockedCommitmentSnapshot, model.CreatedAt, model.UpdatedAt)
}

func datesEqual(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}

type reopenPlan struct {
	Locked []domain.ReopenProject
	Open   []domain.ReopenProject
	Token  string
}

type reopenTaskRow struct {
	ID         string
	ProjectID  string
	AssigneeID *string
}

func (r *Repository) BulkReopen(ctx context.Context, rootProjectID, token string, now time.Time, schedule func(context.Context) error) ([]domain.Project, error) {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	var changed []domain.Project
	err := sharedpersistence.Transaction(ctx, r.database).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock bulk reopen mutation: %w", err)
		}
		plan, err := requiredReopenPlan(tx, rootProjectID)
		if err != nil {
			return err
		}
		if plan.Token != token {
			return domain.ErrBulkReopenStale
		}
		if len(plan.Locked) == 0 {
			return domain.ErrBulkReopenStale
		}
		changed = make([]domain.Project, 0, len(plan.Locked))
		for _, candidate := range plan.Locked {
			value, err := find(tx, candidate.ID, true)
			if err != nil {
				return err
			}
			if value == nil || value.Status != domain.StatusLocked || value.ScheduleVersion != candidate.Version {
				return domain.ErrBulkReopenStale
			}
			leaves, err := loadLifecycleLeaves(tx, candidate.ID)
			if err != nil {
				return err
			}
			hasUnfinished := false
			for _, leaf := range leaves {
				if leaf.ActualStart == nil || leaf.ActualEnd == nil {
					hasUnfinished = true
					break
				}
			}
			if err := value.ChangeStatus(domain.StatusOpen, len(leaves) > 0, hasUnfinished, nil, nil, now); err != nil {
				return err
			}
			result := tx.Model(&projectModel{}).Where("id = ? AND schedule_version = ?", candidate.ID, candidate.Version).Updates(statusUpdates(*value))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return domain.ErrBulkReopenStale
			}
			changed = append(changed, *value)
		}
		if err := schedule(sharedpersistence.WithTransaction(ctx, tx)); err != nil {
			return fmt.Errorf("recalculate projects after bulk reopen: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return changed, nil
}

func requiredReopenPlan(tx *gorm.DB, rootProjectID string) (reopenPlan, error) {
	var projects []projectModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status IN ?", activeStatuses()).Order("priority ASC").Order("id ASC").Find(&projects).Error; err != nil {
		return reopenPlan{}, fmt.Errorf("load active projects for reopen plan: %w", err)
	}
	projectByID := make(map[string]projectModel, len(projects))
	for _, project := range projects {
		projectByID[project.ID] = project
	}
	root, exists := projectByID[rootProjectID]
	if !exists {
		return reopenPlan{}, domain.ErrNotFound
	}
	if root.Status != string(domain.StatusLocked) {
		return reopenPlan{}, domain.ErrStatusTransitionNotAllowed
	}

	var tasks []reopenTaskRow
	if err := tx.Table("wbs_nodes").Select("id, project_id, assignee_id").Where("project_id IN ?", projectIDs(projects)).Scan(&tasks).Error; err != nil {
		return reopenPlan{}, fmt.Errorf("load tasks for reopen plan: %w", err)
	}
	projectByTask := make(map[string]string, len(tasks))
	projectsByAssignee := make(map[string][]string)
	for _, task := range tasks {
		projectByTask[task.ID] = task.ProjectID
		if task.AssigneeID != nil {
			projectsByAssignee[*task.AssigneeID] = append(projectsByAssignee[*task.AssigneeID], task.ProjectID)
		}
	}
	adjacency := make(map[string]map[string]struct{}, len(projects))
	connect := func(left, right string) {
		if left == right || projectByID[left].ID == "" || projectByID[right].ID == "" {
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
	for _, ids := range projectsByAssignee {
		unique := uniqueStrings(ids)
		for left := 0; left < len(unique); left++ {
			for right := left + 1; right < len(unique); right++ {
				connect(unique[left], unique[right])
			}
		}
	}
	var dependencyPairs []struct {
		BlockingTaskID string
		BlockedTaskID  string
	}
	if err := tx.Table("task_dependencies").Select("blocking_task_id, blocked_task_id").Scan(&dependencyPairs).Error; err != nil {
		return reopenPlan{}, fmt.Errorf("load dependencies for reopen plan: %w", err)
	}
	for _, dependency := range dependencyPairs {
		connect(projectByTask[dependency.BlockingTaskID], projectByTask[dependency.BlockedTaskID])
	}

	visited := map[string]struct{}{rootProjectID: {}}
	queue := []string{rootProjectID}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		neighbors := make([]string, 0, len(adjacency[current]))
		for neighbor := range adjacency[current] {
			neighbors = append(neighbors, neighbor)
		}
		sort.Strings(neighbors)
		for _, neighbor := range neighbors {
			if _, seen := visited[neighbor]; seen {
				continue
			}
			visited[neighbor] = struct{}{}
			queue = append(queue, neighbor)
		}
	}
	plan := reopenPlan{}
	for _, project := range projects {
		if _, included := visited[project.ID]; !included {
			continue
		}
		ref := domain.ReopenProject{ID: project.ID, Name: project.Name, Version: project.ScheduleVersion}
		if project.Status == string(domain.StatusLocked) {
			plan.Locked = append(plan.Locked, ref)
		} else {
			plan.Open = append(plan.Open, ref)
		}
	}
	plan.Token = reopenToken(rootProjectID, plan.Locked, plan.Open)
	return plan, nil
}

func projectIDs(projects []projectModel) []string {
	ids := make([]string, 0, len(projects))
	for _, project := range projects {
		ids = append(ids, project.ID)
	}
	return ids
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func reopenToken(root string, locked, open []domain.ReopenProject) string {
	parts := []string{root}
	for _, project := range locked {
		parts = append(parts, fmt.Sprintf("locked:%s:%d", project.ID, project.Version))
	}
	for _, project := range open {
		parts = append(parts, fmt.Sprintf("open:%s:%d", project.ID, project.Version))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(digest[:])
}

func statusUpdates(value domain.Project) map[string]interface{} {
	return map[string]interface{}{"status": value.Status, "closed_at": value.ClosedAt, "locked_execution_snapshot": value.LockedExecutionSnapshot, "locked_commitment_snapshot": value.LockedCommitmentSnapshot, "updated_at": value.UpdatedAt}
}

var _ application.Store = (*Repository)(nil)
