package gormrepo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	groupscheduling "github.com/banggok/sched_mind/backend/internal/shared/groupscheduling"
	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
	"github.com/banggok/sched_mind/backend/internal/wbs/application"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ db *gorm.DB }

var errSchedulePreviewComplete = errors.New("schedule preview complete")

func New(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Tree(ctx context.Context, p string) ([]domain.Node, error) {
	project, err := loadProject(r.db.WithContext(ctx), p, false)
	if err != nil {
		return nil, err
	}
	var models []nodeModel
	if err := r.db.WithContext(ctx).Where("project_id = ?", p).Order("parent_key ASC").Order("position ASC").Order("id ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("load WBS tree: %w", err)
	}
	resolver, err := newGroupResolver(project, models)
	if err != nil {
		return nil, err
	}
	return buildTree(models, resolver), nil
}
func (r *Repository) Find(ctx context.Context, p, id string) (*domain.Node, error) {
	project, err := loadProject(r.db.WithContext(ctx), p, false)
	if err != nil {
		return nil, err
	}
	var models []nodeModel
	if err := r.db.WithContext(ctx).Where("project_id = ?", p).Order("parent_key ASC").Order("position ASC").Find(&models).Error; err != nil {
		return nil, err
	}
	resolver, err := newGroupResolver(project, models)
	if err != nil {
		return nil, err
	}
	for _, n := range flatten(buildTree(models, resolver)) {
		if n.ID == id {
			return &n, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (r *Repository) Create(ctx context.Context, id, p string, parent *string, name string, confirm bool, now time.Time, schedule func(context.Context, string) error, invalidate func(context.Context, []string) error) (*domain.Node, error) {
	var result *domain.Node
	err := r.tx(ctx, p, func(txContext context.Context, tx *gorm.DB, project projectModel) error {
		var parentModel *nodeModel
		var convertParentTask bool
		var schedulingAffected bool
		if parent != nil {
			value, err := findNode(tx, p, *parent, true)
			if err != nil {
				return domain.ErrParentNotFound
			}
			parentModel = &value
			resolver, _, err := loadGroupResolver(tx, project, true)
			if err != nil {
				return err
			}
			if _, err := requirePlanningOpen(resolver, value.ID); err != nil {
				return err
			}
			if value.ActualStart != nil && value.ActualEnd != nil {
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
			schedulingAffected = convertParentTask && (hasData(value) || dependencies > 0)
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
			if err := invalidateDependencyProjects(txContext, tx, model.ID, invalidate); err != nil {
				return err
			}
		}
		if schedulingAffected {
			if err := schedule(txContext, p); err != nil {
				return err
			}
		}
		loaded, err := loadDecoratedNode(tx, project, model.ID)
		if err != nil {
			return err
		}
		result = loaded
		return nil
	})
	return result, err
}

func (r *Repository) CreateSibling(ctx context.Context, id, p, insertAfterID, name string, now time.Time, _ func(context.Context, string) error) (*domain.Node, error) {
	var result *domain.Node
	err := r.tx(ctx, p, func(_ context.Context, tx *gorm.DB, project projectModel) error {
		anchor, err := findNode(tx, p, insertAfterID, true)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.ErrCreateAnchorConflict
			}
			return err
		}
		resolver, all, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		if _, err := requirePlanningOpen(resolver, anchor.ID); err != nil {
			return err
		}
		var siblings []nodeModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("project_id = ? AND parent_key = ?", p, anchor.ParentKey).Order("position ASC").Order("id ASC").Find(&siblings).Error; err != nil {
			return err
		}
		anchorIndex := -1
		for index := range siblings {
			if siblings[index].ID == anchor.ID {
				anchorIndex = index
				break
			}
		}
		if anchorIndex < 0 {
			return domain.ErrCreateAnchorConflict
		}
		position := anchorIndex + 2
		for _, sibling := range siblings {
			if sibling.Position < position {
				continue
			}
			if subtreeHasLocked(resolver, all, sibling.ID) {
				return domain.ErrGroupLockedReadOnly
			}
		}
		if err := shiftSiblingPositionsForInsert(tx, p, anchor.ParentKey, siblings, position); err != nil {
			return err
		}
		value, err := domain.New(id, p, anchor.ParentID, name, position, now)
		if err != nil {
			return err
		}
		model := fromDomain(*value)
		if err := tx.Create(&model).Error; err != nil {
			return mapConflict(err)
		}
		loaded, err := loadDecoratedNode(tx, project, model.ID)
		if err != nil {
			return err
		}
		result = loaded
		return nil
	})
	return result, err
}

func (r *Repository) Rename(ctx context.Context, p, id, name string, now time.Time) (*domain.Node, error) {
	var out *domain.Node
	err := r.tx(ctx, p, func(_ context.Context, tx *gorm.DB, project projectModel) error {
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		resolver, _, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		if _, err := requirePlanningOpen(resolver, id); err != nil {
			return err
		}
		n := toDomain(m, false)
		if err := n.Rename(name, now); err != nil {
			return err
		}
		updates := map[string]any{"name": n.Name, "name_key": domain.NameKey(n.Name), "updated_at": now}
		if resolver.IsGrouping(id) {
			updates["group_scheduling_version"] = gorm.Expr("group_scheduling_version + 1")
		}
		res := tx.Model(&nodeModel{}).Where("id = ?", id).Updates(updates)
		if res.Error != nil {
			return mapConflict(res.Error)
		}
		loaded, err := loadDecoratedNode(tx, project, id)
		if err != nil {
			return err
		}
		out = loaded
		return nil
	})
	return out, err
}

func (r *Repository) UpdateGroupScheduling(ctx context.Context, p, id string, input application.GroupSchedulingInput, now time.Time, schedule func(context.Context, string) error) (*domain.Node, error) {
	var out *domain.Node
	err := r.tx(ctx, p, func(txContext context.Context, tx *gorm.DB, project projectModel) error {
		resolver, all, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		model, ok := all[id]
		if !ok {
			return domain.ErrNotFound
		}
		if !resolver.IsGrouping(id) {
			return domain.ErrGroupNotGroupingWBS
		}
		if model.GroupSchedulingVersion != input.ExpectedVersion {
			return domain.ErrGroupSchedulingStale
		}
		if _, err := requirePlanningOpen(resolver, id); err != nil {
			return err
		}
		if input.Source != "inherit" && input.Source != "override" {
			return domain.ErrGroupSchedulingInvalid
		}
		if input.Source == "override" && input.AutomaticScheduling == nil {
			return domain.ErrGroupSchedulingInvalid
		}
		updates := map[string]any{"group_scheduling_source": input.Source, "group_scheduling_version": gorm.Expr("group_scheduling_version + 1"), "updated_at": now}
		if input.Name != nil {
			node := toDomain(model, true)
			if err := node.Rename(*input.Name, now); err != nil {
				return err
			}
			updates["name"] = node.Name
			updates["name_key"] = domain.NameKey(node.Name)
			model.Name = node.Name
		}
		before := make(map[string]groupscheduling.Effective)
		for taskID := range all {
			if taskID == id || isDescendant(all, id, taskID) {
				if !resolver.IsGrouping(taskID) {
					value, effectiveErr := resolver.EffectiveFor(taskID)
					if effectiveErr != nil {
						return effectiveErr
					}
					before[taskID] = value
				}
			}
		}
		model.GroupSchedulingSource = input.Source
		if input.Source == "inherit" {
			updates["group_automatic_scheduling"] = nil
			updates["group_scheduling_start_date"] = nil
			model.GroupAutomaticScheduling = nil
			model.GroupSchedulingStartDate = nil
		} else {
			updates["group_automatic_scheduling"] = input.AutomaticScheduling
			updates["group_scheduling_start_date"] = input.SchedulingStartDate
			model.GroupAutomaticScheduling = input.AutomaticScheduling
			model.GroupSchedulingStartDate = input.SchedulingStartDate
		}
		if err := tx.Model(&nodeModel{}).Where("id = ? AND project_id = ?", id, p).Updates(updates).Error; err != nil {
			return mapConflict(err)
		}
		model.GroupSchedulingVersion++
		all[id] = model
		models := make([]nodeModel, 0, len(all))
		for _, item := range all {
			models = append(models, item)
		}
		afterResolver, err := newGroupResolver(project, models)
		if err != nil {
			return err
		}
		effectiveChanged := false
		for taskID, previous := range before {
			current, effectiveErr := afterResolver.EffectiveFor(taskID)
			if effectiveErr != nil {
				return effectiveErr
			}
			if previous.AutomaticScheduling != current.AutomaticScheduling || differentDate(previous.SchedulingStartDate, current.SchedulingStartDate) {
				effectiveChanged = true
				break
			}
		}
		if effectiveChanged {
			if err := schedule(txContext, p); err != nil {
				return err
			}
		}
		node := toDomain(model, true)
		effective, err := afterResolver.EffectiveFor(id)
		if err != nil {
			return err
		}
		decorateEffective(&node, effective)
		inherited, err := afterResolver.InheritedFor(id)
		if err != nil {
			return err
		}
		decorateInherited(&node, inherited)
		out = &node
		return nil
	})
	return out, err
}

func (r *Repository) ChangeGroupStatus(ctx context.Context, p, id, target string, expectedVersion int64, now time.Time, schedule func(context.Context, string) error) (*domain.Node, error) {
	if target != "locked" && target != "open" {
		return nil, domain.ErrGroupSchedulingInvalid
	}
	var out *domain.Node
	err := r.tx(ctx, p, func(txContext context.Context, tx *gorm.DB, project projectModel) error {
		resolver, all, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		model, ok := all[id]
		if !ok {
			return domain.ErrNotFound
		}
		if !resolver.IsGrouping(id) {
			return domain.ErrGroupNotGroupingWBS
		}
		if model.GroupSchedulingVersion != expectedVersion {
			return domain.ErrGroupSchedulingStale
		}
		if target == "locked" {
			if model.GroupLocalStatus == "locked" {
				return domain.ErrGroupLockedReadOnly
			}
			effective, err := requirePlanningOpen(resolver, id)
			if err != nil {
				return err
			}
			hasLeaf := false
			for taskID, task := range all {
				if taskID == id || !isDescendant(all, id, taskID) || resolver.IsGrouping(taskID) {
					continue
				}
				hasLeaf = true
				completed := task.ActualStart != nil && task.ActualEnd != nil
				if !completed && (task.ExecutionStart == nil || task.ExecutionEnd == nil || task.CommitmentStart == nil || task.CommitmentEnd == nil || task.ExecutionUnscheduledReason != nil || task.CommitmentUnscheduledReason != nil) {
					return domain.ErrGroupCannotLockUnscheduled
				}
			}
			if !hasLeaf {
				return domain.ErrGroupCannotLockUnscheduled
			}
			updates := map[string]any{"group_local_status": "locked", "group_locked_automatic_scheduling": effective.AutomaticScheduling, "group_locked_scheduling_start_date": effective.SchedulingStartDate, "group_scheduling_version": gorm.Expr("group_scheduling_version + 1"), "updated_at": now}
			if err := tx.Model(&nodeModel{}).Where("id = ? AND project_id = ? AND group_local_status = ?", id, p, "open").Updates(updates).Error; err != nil {
				return err
			}
			model.GroupLocalStatus = "locked"
			lockedAuto := effective.AutomaticScheduling
			model.GroupLockedAutomaticScheduling = &lockedAuto
			model.GroupLockedSchedulingStartDate = effective.SchedulingStartDate
			model.GroupSchedulingVersion++
		} else {
			if model.GroupLocalStatus != "locked" {
				return domain.ErrGroupReopenRequired
			}
			currentEffective, effectiveErr := resolver.EffectiveFor(id)
			if effectiveErr != nil {
				return effectiveErr
			}
			if currentEffective.LockOwnerID != id {
				return domain.ErrGroupReopenRequired
			}
			plan, planErr := requiredGroupReopenPlan(txContext, tx, p, id, schedule)
			if planErr != nil {
				return planErr
			}
			if len(plan.LockedProjects)+len(plan.LockedGroups) > 1 {
				return schedulingimpact.Error{Kind: schedulingimpact.ReopenClosureRequired, Token: plan.Token, LockedProjects: plan.LockedProjects, OpenProjects: plan.OpenProjects, LockedGroups: plan.LockedGroups, OpenGroups: plan.OpenGroups}
			}
			model.GroupLocalStatus = "open"
			model.GroupLockedAutomaticScheduling = nil
			model.GroupLockedSchedulingStartDate = nil
			model.GroupSchedulingVersion++
			all[id] = model
			models := make([]nodeModel, 0, len(all))
			for _, item := range all {
				models = append(models, item)
			}
			probe, err := newGroupResolver(project, models)
			if err != nil {
				return err
			}
			effective, err := probe.EffectiveFor(id)
			if err != nil {
				return err
			}
			if effective.Lifecycle != groupscheduling.LifecycleOpen {
				return domain.ErrGroupReopenRequired
			}
			if err := tx.Model(&nodeModel{}).Where("id = ? AND project_id = ? AND group_local_status = ?", id, p, "locked").Updates(map[string]any{"group_local_status": "open", "group_locked_automatic_scheduling": nil, "group_locked_scheduling_start_date": nil, "group_scheduling_version": gorm.Expr("group_scheduling_version + 1"), "updated_at": now}).Error; err != nil {
				return err
			}
			if err := schedule(txContext, p); err != nil {
				return err
			}
		}
		models := make([]nodeModel, 0, len(all))
		for key, item := range all {
			if key == id {
				item = model
			}
			models = append(models, item)
		}
		finalResolver, err := newGroupResolver(project, models)
		if err != nil {
			return err
		}
		node := toDomain(model, true)
		effective, err := finalResolver.EffectiveFor(id)
		if err != nil {
			return err
		}
		decorateEffective(&node, effective)
		inherited, err := finalResolver.InheritedFor(id)
		if err != nil {
			return err
		}
		decorateInherited(&node, inherited)
		out = &node
		return nil
	})
	return out, err
}

func (r *Repository) BulkReopenGroup(ctx context.Context, p, id, token string, expectedVersion int64, now time.Time, schedule func(context.Context, string) error) (*domain.Node, error) {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	var out *domain.Node
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock Group reopen closure mutation: %w", err)
		}
		plan, err := requiredGroupReopenPlan(ctx, tx, p, id, schedule)
		if err != nil {
			return err
		}
		var root nodeModel
		if err := tx.Select("group_scheduling_version").Where("id = ? AND project_id = ?", id, p).First(&root).Error; err != nil {
			return err
		}
		if root.GroupSchedulingVersion != expectedVersion {
			return domain.ErrGroupSchedulingStale
		}
		if token == "" || plan.Token != token {
			return schedulingimpact.Error{Kind: schedulingimpact.StaleImpact, Token: plan.Token, LockedProjects: plan.LockedProjects, OpenProjects: plan.OpenProjects, LockedGroups: plan.LockedGroups, OpenGroups: plan.OpenGroups}
		}
		for _, project := range plan.LockedProjects {
			result := tx.Model(&projectModel{}).
				Where("id = ? AND status = ? AND schedule_version = ?", project.ID, "locked", project.Version).
				Updates(map[string]any{"status": "open", "locked_execution_snapshot": nil, "locked_commitment_snapshot": nil, "updated_at": now})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return schedulingimpact.Error{Kind: schedulingimpact.StaleImpact, Token: plan.Token, LockedProjects: plan.LockedProjects, OpenProjects: plan.OpenProjects, LockedGroups: plan.LockedGroups, OpenGroups: plan.OpenGroups}
			}
		}
		for _, group := range plan.LockedGroups {
			result := tx.Model(&nodeModel{}).
				Where("id = ? AND project_id = ? AND group_local_status = ?", group.ID, group.ProjectID, "locked").
				Updates(map[string]any{"group_local_status": "open", "group_locked_automatic_scheduling": nil, "group_locked_scheduling_start_date": nil, "group_scheduling_version": gorm.Expr("group_scheduling_version + 1"), "updated_at": now})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return schedulingimpact.Error{Kind: schedulingimpact.StaleImpact, Token: plan.Token, LockedProjects: plan.LockedProjects, OpenProjects: plan.OpenProjects, LockedGroups: plan.LockedGroups, OpenGroups: plan.OpenGroups}
			}
		}
		txContext := sharedpersistence.WithTransaction(ctx, tx)
		txContext = schedulingimpact.WithGroupOperation(txContext, p, id, schedulingimpact.ModeBulkReopen)
		if err := schedule(txContext, p); err != nil {
			return fmt.Errorf("recalculate schedule after Group reopen closure: %w", err)
		}
		project, err := loadProject(tx, p, false)
		if err != nil {
			return err
		}
		resolver, all, err := loadGroupResolver(tx, project, false)
		if err != nil {
			return err
		}
		model, exists := all[id]
		if !exists {
			return domain.ErrNotFound
		}
		node := toDomain(model, resolver.IsGrouping(id))
		effective, err := resolver.EffectiveFor(id)
		if err != nil {
			return err
		}
		decorateEffective(&node, effective)
		inherited, err := resolver.InheritedFor(id)
		if err != nil {
			return err
		}
		decorateInherited(&node, inherited)
		out = &node
		return nil
	})
	return out, err
}

func (r *Repository) UpdateExecutable(ctx context.Context, p, id string, input application.WriteExecutableInput, now time.Time, schedule func(context.Context, string) error) (*domain.Node, error) {
	var out *domain.Node
	err := r.tx(ctx, p, func(txContext context.Context, tx *gorm.DB, project projectModel) error {
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		count, err := childCount(tx, p, id)
		if err != nil {
			return err
		}
		n := toDomain(m, count > 0)
		resolver, _, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		effective, err := requirePlanningOpen(resolver, id)
		if err != nil {
			return err
		}
		percentage := n.Executable.CapacityAllocationPercentage
		if input.CapacityAllocationPercentage != nil {
			percentage = *input.CapacityAllocationPercentage
		}
		fields := domain.ExecutableFields{RoleID: input.RoleID, AssigneeID: input.AssigneeID, EffortMinutes: input.EffortMinutes, LagDays: input.LagDays, CapacityAllocationPercentage: percentage, ExecutionTimeline: input.Execution, CommitmentTimeline: input.Commitment, ActualStart: n.Executable.ActualStart, ActualEnd: n.Executable.ActualEnd}
		if input.Name != nil {
			if err := n.Rename(*input.Name, now); err != nil {
				return err
			}
		}
		if err := validateMember(tx, fields); err != nil {
			return err
		}
		oldAssignee, oldEffort, oldLag, oldPercentage := n.Executable.AssigneeID, n.Executable.EffortMinutes, n.Executable.LagDays, n.Executable.CapacityAllocationPercentage
		oldExecution, oldCommitment := n.Executable.ExecutionTimeline, n.Executable.CommitmentTimeline
		if err := n.UpdateExecutable(fields, effective.AutomaticScheduling, effective.Lifecycle == groupscheduling.LifecycleOpen, now); err != nil {
			return err
		}
		applyFields(&m, n.Executable)
		updates := executableUpdates(m, now)
		updates["name"] = n.Name
		updates["name_key"] = domain.NameKey(n.Name)
		if err := tx.Model(&nodeModel{}).Where("id = ?", id).Updates(updates).Error; err != nil {
			return mapConflict(err)
		}
		schedulingChanged := differentString(oldAssignee, n.Executable.AssigneeID) ||
			differentInt(oldEffort, n.Executable.EffortMinutes) || oldLag != n.Executable.LagDays ||
			oldPercentage != n.Executable.CapacityAllocationPercentage ||
			differentTimeline(oldExecution, n.Executable.ExecutionTimeline) ||
			differentTimeline(oldCommitment, n.Executable.CommitmentTimeline)
		if schedulingChanged {
			if err := schedule(txContext, p); err != nil {
				return err
			}
		}
		loaded, err := loadDecoratedNode(tx, project, id)
		if err != nil {
			return err
		}
		out = loaded
		return nil
	})
	return out, err
}
func (r *Repository) PreviewExecutableSchedule(ctx context.Context, p, id string, input application.PreviewExecutableInput, now time.Time, schedule func(context.Context, string) error) (*application.SchedulePreview, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()

	var preview *application.SchedulePreview
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock schedule preview: %w", err)
		}
		project, err := loadProject(tx, p, true)
		if err != nil {
			return err
		}
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		if m.ActualStart != nil && m.ActualEnd != nil {
			return domain.ErrSchedulePreviewUnavailable
		}
		resolver, _, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		effective, err := requirePlanningOpen(resolver, id)
		if err != nil || !effective.AutomaticScheduling {
			return domain.ErrSchedulePreviewUnavailable
		}
		count, err := childCount(tx, p, id)
		if err != nil {
			return err
		}
		n := toDomain(m, count > 0)
		percentage := input.CapacityAllocationPercentage
		if percentage == 0 {
			percentage = 100
		}
		fields := domain.ExecutableFields{
			RoleID:                       input.RoleID,
			AssigneeID:                   input.AssigneeID,
			EffortMinutes:                input.EffortMinutes,
			LagDays:                      input.LagDays,
			CapacityAllocationPercentage: percentage,
		}
		if err := validateMember(tx, fields); err != nil {
			return err
		}
		if err := n.UpdateExecutable(fields, true, true, now); err != nil {
			return err
		}
		applyFields(&m, n.Executable)
		if err := tx.Model(&nodeModel{}).Where("id = ?", id).Updates(executableUpdates(m, now)).Error; err != nil {
			return mapConflict(err)
		}
		if err := schedule(sharedpersistence.WithTransaction(ctx, tx), p); err != nil {
			return err
		}
		generated, err := findNode(tx, p, id, false)
		if err != nil {
			return err
		}
		value := toDomain(generated, false)
		decorateEffective(&value, effective)
		inherited, err := resolver.InheritedFor(id)
		if err != nil {
			return err
		}
		decorateInherited(&value, inherited)
		preview = &application.SchedulePreview{Task: &value}
		return errSchedulePreviewComplete
	})
	if errors.Is(err, errSchedulePreviewComplete) {
		return preview, nil
	}
	return nil, err
}

func (r *Repository) Complete(ctx context.Context, p, id string, actualStart, actualEnd, now time.Time, invalidate func(context.Context, []string) error) (*domain.Node, error) {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	var out *domain.Node
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		project, err := loadProject(tx, p, true)
		if err != nil {
			return err
		}
		if project.Status == "closed" {
			return domain.ErrProjectClosedReadOnly
		}
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		count, err := childCount(tx, p, id)
		if err != nil {
			return err
		}
		n := toDomain(m, count > 0)
		resolver, _, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		effective, err := resolver.EffectiveFor(id)
		if err != nil {
			return err
		}
		if err := ensureCompletedPredecessors(tx, id); err != nil {
			return err
		}
		baselineExecutionStart := m.ExecutionStart
		if err := n.Complete(actualStart, actualEnd, now); err != nil {
			return err
		}
		m.ActualStart = n.Executable.ActualStart
		m.ActualEnd = n.Executable.ActualEnd
		if effective.Lifecycle == groupscheduling.LifecycleOpen {
			m.ExecutionStart = earlierDate(m.ExecutionStart, n.Executable.ActualStart)
			m.ExecutionEnd = n.Executable.ActualEnd
			m.CommitmentStart = earlierDate(m.CommitmentStart, n.Executable.ActualStart)
			m.CommitmentEnd = n.Executable.ActualEnd
			m.ExecutionUnscheduledReason = nil
			m.CommitmentUnscheduledReason = nil
			n.Executable.ExecutionTimeline = domain.Timeline{Start: m.ExecutionStart, End: m.ExecutionEnd}
			n.Executable.CommitmentTimeline = domain.Timeline{Start: m.CommitmentStart, End: m.CommitmentEnd}
			n.Executable.ExecutionUnscheduledReason = nil
			n.Executable.CommitmentUnscheduledReason = nil
		}
		updates := executableUpdates(m, now)
		result := tx.Model(&nodeModel{}).
			Where("id = ? AND project_id = ? AND actual_start IS NULL AND actual_end IS NULL", id, p).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrCompletedReadOnly
		}
		if _, err := replaceActualAllocations(tx, m, baselineExecutionStart, actualStart, actualEnd); err != nil {
			return err
		}
		if err := invalidate(sharedpersistence.WithTransaction(ctx, tx), []string{p}); err != nil {
			return err
		}
		loaded, err := loadDecoratedNode(tx, project, id)
		if err != nil {
			return err
		}
		out = loaded
		return nil
	})
	return out, err
}

func (r *Repository) Reopen(ctx context.Context, p, id string, now time.Time, invalidate func(context.Context, []string) error) (*domain.Node, error) {
	observed, err := findNode(r.db.WithContext(ctx), p, id, false)
	if err != nil {
		return nil, err
	}
	observedChildren, err := childCount(r.db.WithContext(ctx), p, id)
	if err != nil {
		return nil, err
	}
	observedNode := toDomain(observed, observedChildren > 0)
	if err := observedNode.Reopen(now); err != nil {
		return nil, err
	}

	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	var out *domain.Node
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		// The preflight read established that this request observed a completed
		// executable Task. If the serialized transaction now sees it unfinished,
		// another request won the transition and this request is a conflict.
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
		resolver, _, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		if _, err := requirePlanningOpen(resolver, id); err != nil {
			return err
		}
		n := toDomain(m, count > 0)
		if err := n.Reopen(now); err != nil {
			if errors.Is(err, domain.ErrTaskNotCompleted) {
				return domain.ErrTaskReopenConflict
			}
			return err
		}
		result := tx.Model(&nodeModel{}).
			Where("id = ? AND project_id = ? AND actual_start IS NOT NULL AND actual_end IS NOT NULL", id, p).
			Updates(map[string]any{"actual_start": nil, "actual_end": nil, "updated_at": now})
		if result.Error != nil {
			if isReopenConcurrencyError(tx, result.Error) {
				return fmt.Errorf("%w: %v", domain.ErrTaskReopenConflict, result.Error)
			}
			return result.Error
		}
		if result.RowsAffected != 1 {
			return domain.ErrTaskReopenConflict
		}
		if err := tx.Where("task_id = ? AND timeline = ?", id, "actual").Delete(&actualAllocationModel{}).Error; err != nil {
			return err
		}
		if err := invalidate(sharedpersistence.WithTransaction(ctx, tx), []string{p}); err != nil {
			return err
		}
		loaded, err := loadDecoratedNode(tx, project, id)
		if err != nil {
			return err
		}
		out = loaded
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
	return r.tx(ctx, p, func(txContext context.Context, tx *gorm.DB, project projectModel) error {
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		resolver, all, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		if _, err := requirePlanningOpen(resolver, id); err != nil {
			return err
		}
		if subtreeHasLocked(resolver, all, id) {
			return domain.ErrGroupLockedReadOnly
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
		if _, err := requirePlanningOpen(resolver, b.ID); err != nil {
			return err
		}
		if subtreeHasLocked(resolver, all, b.ID) {
			return domain.ErrGroupLockedReadOnly
		}
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
		return schedule(txContext, p)
	})
}

func (r *Repository) Place(ctx context.Context, p, id, targetID string, placement domain.Placement, now time.Time, schedule func(context.Context, string) error) error {
	return r.tx(ctx, p, func(txContext context.Context, tx *gorm.DB, project projectModel) error {
		source, err := findNode(tx, p, id, true)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.ErrReorderTargetInvalid
			}
			return err
		}
		target, err := findNode(tx, p, targetID, true)
		if err != nil {
			if errors.Is(err, domain.ErrNotFound) {
				return domain.ErrReorderTargetInvalid
			}
			return err
		}
		resolver, all, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		if _, err := requirePlanningOpen(resolver, source.ID); err != nil {
			return err
		}
		if _, err := requirePlanningOpen(resolver, target.ID); err != nil {
			return err
		}
		if subtreeHasLocked(resolver, all, source.ID) || subtreeHasLocked(resolver, all, target.ID) {
			return domain.ErrGroupLockedReadOnly
		}
		if source.ID == target.ID || source.ParentKey != target.ParentKey {
			return domain.ErrReorderTargetInvalid
		}
		var siblings []nodeModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("project_id = ? AND parent_key = ?", p, source.ParentKey).Order("position ASC").Order("id ASC").Find(&siblings).Error; err != nil {
			return err
		}
		sourceIndex, targetIndex := -1, -1
		for index := range siblings {
			if siblings[index].ID == source.ID {
				sourceIndex = index
			}
			if siblings[index].ID == target.ID {
				targetIndex = index
			}
		}
		if sourceIndex < 0 || targetIndex < 0 {
			return domain.ErrReorderTargetInvalid
		}
		ordered := append([]nodeModel(nil), siblings...)
		moving := ordered[sourceIndex]
		ordered = append(ordered[:sourceIndex], ordered[sourceIndex+1:]...)
		newTargetIndex := -1
		for index := range ordered {
			if ordered[index].ID == target.ID {
				newTargetIndex = index
				break
			}
		}
		if newTargetIndex < 0 {
			return domain.ErrReorderTargetInvalid
		}
		insertIndex := newTargetIndex
		if placement == domain.PlaceAfter {
			insertIndex++
		} else if placement != domain.PlaceBefore {
			return domain.ErrReorderTargetInvalid
		}
		ordered = append(ordered, nodeModel{})
		copy(ordered[insertIndex+1:], ordered[insertIndex:])
		ordered[insertIndex] = moving
		if sameSiblingOrder(siblings, ordered) {
			return nil
		}
		if reorderedLockedSibling(resolver, all, siblings, ordered) {
			return domain.ErrGroupLockedReadOnly
		}
		if err := persistSiblingOrder(tx, p, source.ParentKey, ordered, source.ID, now); err != nil {
			return err
		}
		return schedule(txContext, p)
	})
}

func (r *Repository) Move(ctx context.Context, p, id, conversionID string, parent *string, confirm bool, now time.Time, schedule func(context.Context, string) error, invalidate func(context.Context, []string) error) error {
	return r.tx(ctx, p, func(txContext context.Context, tx *gorm.DB, project projectModel) error {
		all, err := loadLocked(tx, p)
		if err != nil {
			return err
		}
		moving, ok := all[id]
		if !ok {
			return domain.ErrNotFound
		}
		resolver, _, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		if _, err := requirePlanningOpen(resolver, id); err != nil {
			return err
		}
		if subtreeHasLocked(resolver, all, id) {
			return domain.ErrGroupLockedReadOnly
		}
		if moving.ParentID != nil {
			if parentModel, exists := all[*moving.ParentID]; exists && parentModel.GroupSchedulingSource == "override" {
				count, countErr := childCount(tx, p, parentModel.ID)
				if countErr != nil {
					return countErr
				}
				if count == 1 {
					return domain.ErrGroupOverrideMustReset
				}
			}
		}
		if parent != nil {
			if *parent == id || isDescendant(all, id, *parent) {
				return domain.ErrCycle
			}
			dest, ok := all[*parent]
			if !ok {
				return domain.ErrParentNotFound
			}
			if _, err := requirePlanningOpen(resolver, dest.ID); err != nil {
				return err
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
				if err := invalidateDependencyProjects(txContext, tx, conversionID, invalidate); err != nil {
					return err
				}
			}
		}
		if shiftedLockedSibling(resolver, all, moving.ParentKey, moving.Position, moving.ID) {
			return domain.ErrGroupLockedReadOnly
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
		return schedule(txContext, p)
	})
}
func (r *Repository) Delete(ctx context.Context, p, id string, now time.Time, schedule func(context.Context, string) error, invalidate func(context.Context, []string) error) error {
	return r.tx(ctx, p, func(txContext context.Context, tx *gorm.DB, project projectModel) error {
		m, err := findNode(tx, p, id, true)
		if err != nil {
			return err
		}
		resolver, all, err := loadGroupResolver(tx, project, true)
		if err != nil {
			return err
		}
		if _, err := requirePlanningOpen(resolver, id); err != nil {
			return err
		}
		if m.ParentID != nil {
			if parentModel, exists := all[*m.ParentID]; exists && parentModel.GroupSchedulingSource == "override" {
				parentChildren, countErr := childCount(tx, p, parentModel.ID)
				if countErr != nil {
					return countErr
				}
				if parentChildren == 1 {
					return domain.ErrGroupOverrideMustReset
				}
			}
		}
		if m.ActualStart != nil && m.ActualEnd != nil {
			return domain.ErrCompletedReadOnly
		}
		count, err := childCount(tx, p, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return domain.ErrHasChildren
		}
		if shiftedLockedSibling(resolver, all, m.ParentKey, m.Position, m.ID) {
			return domain.ErrGroupLockedReadOnly
		}
		dependencyProjects, err := dependencyProjectIDs(tx, id)
		if err != nil {
			return err
		}
		if err := tx.Where("blocking_task_id = ? OR blocked_task_id = ?", id, id).Delete(&dependencyLinkModel{}).Error; err != nil {
			return err
		}
		if err := tx.Where("id = ?", id).Delete(&nodeModel{}).Error; err != nil {
			return err
		}
		if err := compact(tx, p, m.ParentKey); err != nil {
			return err
		}
		if err := invalidateProjectsIfAutomatic(txContext, tx, dependencyProjects, invalidate); err != nil {
			return err
		}
		if !containsProjectID(dependencyProjects, p) {
			return schedule(txContext, p)
		}
		return nil
	})
}

func (r *Repository) tx(ctx context.Context, p string, fn func(context.Context, *gorm.DB, projectModel) error) error {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock schedule mutation: %w", err)
		}
		project, err := loadProject(tx, p, true)
		if err != nil {
			return err
		}
		if project.Status == "closed" {
			return domain.ErrProjectClosedReadOnly
		}
		if project.Status == "locked" {
			return domain.ErrProjectLockedReadOnly
		}
		return fn(sharedpersistence.WithTransaction(ctx, tx), tx, project)
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

func ensureCompletedPredecessors(db *gorm.DB, taskID string) error {
	var count int64
	err := db.Table("task_dependencies AS dependency").
		Joins("JOIN wbs_nodes AS predecessor ON predecessor.id = dependency.blocking_task_id").
		Where("dependency.blocked_task_id = ?", taskID).
		Where("predecessor.actual_start IS NULL OR predecessor.actual_end IS NULL").
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("validate completed predecessors: %w", err)
	}
	if count > 0 {
		return domain.ErrIncompletePredecessor
	}
	return nil
}

func earlierDate(existing, actual *time.Time) *time.Time {
	if actual == nil {
		return existing
	}
	if existing == nil || actual.Before(*existing) {
		value := schedulingdomain.DateOnly(*actual)
		return &value
	}
	value := schedulingdomain.DateOnly(*existing)
	return &value
}
func shiftSiblingPositionsForInsert(tx *gorm.DB, projectID, parentKey string, siblings []nodeModel, insertPosition int) error {
	if insertPosition < 1 || insertPosition > len(siblings)+1 {
		return domain.ErrCreatePositionInvalid
	}
	offset := len(siblings) + 1
	if err := tx.Model(&nodeModel{}).
		Where("project_id = ? AND parent_key = ? AND position >= ?", projectID, parentKey, insertPosition).
		Update("position", gorm.Expr("position + ?", offset)).Error; err != nil {
		return err
	}
	for _, sibling := range siblings {
		if sibling.Position < insertPosition {
			continue
		}
		if err := tx.Model(&nodeModel{}).Where("id = ?", sibling.ID).Update("position", sibling.Position+1).Error; err != nil {
			return err
		}
	}
	return nil
}

func persistSiblingOrder(tx *gorm.DB, projectID, parentKey string, ordered []nodeModel, sourceID string, now time.Time) error {
	offset := len(ordered) + 1
	if err := tx.Model(&nodeModel{}).
		Where("project_id = ? AND parent_key = ?", projectID, parentKey).
		Update("position", gorm.Expr("position + ?", offset)).Error; err != nil {
		return err
	}
	for index, sibling := range ordered {
		updates := map[string]any{"position": index + 1}
		if sibling.ID == sourceID {
			updates["updated_at"] = now
		}
		if err := tx.Model(&nodeModel{}).Where("id = ?", sibling.ID).Updates(updates).Error; err != nil {
			return err
		}
	}
	return nil
}

func sameSiblingOrder(left, right []nodeModel) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].ID != right[index].ID {
			return false
		}
	}
	return true
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
	return m.RoleID != nil || m.AssigneeID != nil || m.EffortMinutes != nil || m.ExecutionStart != nil || m.ExecutionEnd != nil || m.CommitmentStart != nil || m.CommitmentEnd != nil || m.ExecutionUnscheduledReason != nil || m.CommitmentUnscheduledReason != nil || m.ActualStart != nil || m.ActualEnd != nil
}
func clearExecutable(m *nodeModel) {
	m.RoleID = nil
	m.AssigneeID = nil
	m.EffortMinutes = nil
	m.LagDays = 0
	m.CapacityAllocationPercentage = 100
	m.ExecutionStart = nil
	m.ExecutionEnd = nil
	m.CommitmentStart = nil
	m.CommitmentEnd = nil
	m.ExecutionUnscheduledReason = nil
	m.CommitmentUnscheduledReason = nil
	m.ActualStart = nil
	m.ActualEnd = nil
}
func copyExecutable(dst *nodeModel, src nodeModel) {
	dst.RoleID = src.RoleID
	dst.AssigneeID = src.AssigneeID
	dst.EffortMinutes = src.EffortMinutes
	dst.LagDays = src.LagDays
	dst.CapacityAllocationPercentage = src.CapacityAllocationPercentage
	dst.ExecutionStart = src.ExecutionStart
	dst.ExecutionEnd = src.ExecutionEnd
	dst.CommitmentStart = src.CommitmentStart
	dst.CommitmentEnd = src.CommitmentEnd
	dst.ExecutionUnscheduledReason = src.ExecutionUnscheduledReason
	dst.CommitmentUnscheduledReason = src.CommitmentUnscheduledReason
	dst.ActualStart = src.ActualStart
	dst.ActualEnd = src.ActualEnd
}
func executableUpdates(m nodeModel, now time.Time) map[string]any {
	return map[string]any{"role_id": m.RoleID, "assignee_id": m.AssigneeID, "effort_minutes": m.EffortMinutes, "lag_days": m.LagDays, "capacity_allocation_percentage": m.CapacityAllocationPercentage, "execution_start": m.ExecutionStart, "execution_end": m.ExecutionEnd, "commitment_start": m.CommitmentStart, "commitment_end": m.CommitmentEnd, "execution_unscheduled_reason": m.ExecutionUnscheduledReason, "commitment_unscheduled_reason": m.CommitmentUnscheduledReason, "actual_start": m.ActualStart, "actual_end": m.ActualEnd, "updated_at": now}
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
			child := nodeModel{ID: conversionID, ProjectID: p, ParentID: &dest.ID, ParentKey: dest.ID, Name: name, NameKey: key, Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now}
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
func containsProjectID(ids []string, projectID string) bool {
	for _, id := range ids {
		if id == projectID {
			return true
		}
	}
	return false
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
	return invalidate(sharedpersistence.WithTransaction(ctx, tx), ids)
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

type groupReopenPlan struct {
	LockedProjects []schedulingimpact.Project
	OpenProjects   []schedulingimpact.Project
	LockedGroups   []schedulingimpact.Group
	OpenGroups     []schedulingimpact.Group
	StateSignature string
	Token          string
}

var errRollbackGroupReopenPreview = errors.New("rollback Group reopen preview")

func requiredGroupReopenPlan(ctx context.Context, tx *gorm.DB, rootProjectID, rootGroupID string, schedule func(context.Context, string) error) (groupReopenPlan, error) {
	var projects []projectModel
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status IN ?", []string{"open", "locked"}).Order("id ASC").Find(&projects).Error; err != nil {
		return groupReopenPlan{}, fmt.Errorf("load active Projects for Group reopen closure: %w", err)
	}
	projectByID := make(map[string]projectModel, len(projects))
	for _, project := range projects {
		projectByID[project.ID] = project
	}
	rootProject, exists := projectByID[rootProjectID]
	if !exists || rootProject.Status != "open" {
		return groupReopenPlan{}, domain.ErrGroupReopenRequired
	}
	var nodes []nodeModel
	projectIDs := make([]string, 0, len(projects))
	for _, project := range projects {
		projectIDs = append(projectIDs, project.ID)
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("project_id IN ?", projectIDs).Order("project_id ASC").Order("parent_key ASC").Order("position ASC").Find(&nodes).Error; err != nil {
		return groupReopenPlan{}, fmt.Errorf("load Groups for reopen closure: %w", err)
	}
	nodeByID := make(map[string]nodeModel, len(nodes))
	for _, node := range nodes {
		nodeByID[node.ID] = node
	}
	root, exists := nodeByID[rootGroupID]
	if !exists || root.ProjectID != rootProjectID || root.GroupLocalStatus != "locked" {
		return groupReopenPlan{}, domain.ErrGroupReopenRequired
	}
	resolver, err := newGroupResolver(rootProject, filterProjectNodes(nodes, rootProjectID))
	if err != nil {
		return groupReopenPlan{}, err
	}
	if !resolver.IsGrouping(rootGroupID) {
		return groupReopenPlan{}, domain.ErrGroupNotGroupingWBS
	}
	rootEffective, err := resolver.EffectiveFor(rootGroupID)
	if err != nil {
		return groupReopenPlan{}, err
	}
	if rootEffective.LockOwnerID != rootGroupID {
		return groupReopenPlan{}, domain.ErrGroupReopenRequired
	}

	reopenProjects := make(map[string]struct{})
	reopenGroups := map[string]struct{}{rootGroupID: {}}
	openProjects := make(map[string]schedulingimpact.Project)
	openGroups := make(map[string]schedulingimpact.Group)
	converged := false
	for iteration := 0; iteration < len(projects)+len(nodes)+2; iteration++ {
		impact, err := simulateGroupReopenImpact(ctx, tx, rootProjectID, rootGroupID, reopenProjects, reopenGroups, schedule)
		if err != nil {
			return groupReopenPlan{}, err
		}
		added := false
		for _, candidate := range impact.LockedProjects {
			project, exists := projectByID[candidate.ID]
			if !exists || project.Status != "locked" {
				continue
			}
			if _, already := reopenProjects[project.ID]; !already {
				reopenProjects[project.ID] = struct{}{}
				added = true
			}
		}
		for _, candidate := range impact.LockedGroups {
			group, exists := nodeByID[candidate.ID]
			if !exists || group.GroupLocalStatus != "locked" {
				continue
			}
			if addGroupAndLockingAncestors(group, nodeByID, projectByID, reopenGroups, reopenProjects) {
				added = true
			}
		}
		if added {
			continue
		}
		if len(impact.LockedProjects) > 0 || len(impact.LockedGroups) > 0 {
			return groupReopenPlan{}, fmt.Errorf("Group reopen closure simulation did not converge")
		}
		for _, candidate := range impact.OpenProjects {
			if _, reopening := reopenProjects[candidate.ID]; !reopening {
				openProjects[candidate.ID] = candidate
			}
		}
		for _, candidate := range impact.OpenGroups {
			if _, reopening := reopenGroups[candidate.ID]; !reopening {
				openGroups[candidate.ID] = candidate
			}
		}
		converged = true
		break
	}
	if !converged {
		return groupReopenPlan{}, fmt.Errorf("Group reopen closure exceeded the bounded fixed-point search")
	}

	plan := groupReopenPlan{}
	for _, project := range projects {
		if _, included := reopenProjects[project.ID]; included {
			plan.LockedProjects = append(plan.LockedProjects, schedulingimpact.Project{ID: project.ID, Name: project.Name, Status: project.Status, Version: project.ScheduleVersion})
		}
	}
	groupIDs := make([]string, 0, len(reopenGroups))
	for groupID := range reopenGroups {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Strings(groupIDs)
	for _, groupID := range groupIDs {
		group := nodeByID[groupID]
		project := projectByID[group.ProjectID]
		plan.LockedGroups = append(plan.LockedGroups, schedulingimpact.Group{ID: group.ID, ProjectID: group.ProjectID, Name: group.Name, Path: qualifiedGroupPath(project.Name, group.ID, nodeByID), Status: group.GroupLocalStatus, Version: group.GroupSchedulingVersion})
	}
	for _, value := range openProjects {
		plan.OpenProjects = append(plan.OpenProjects, value)
	}
	for _, value := range openGroups {
		plan.OpenGroups = append(plan.OpenGroups, value)
	}
	sort.Slice(plan.OpenProjects, func(i, j int) bool { return plan.OpenProjects[i].ID < plan.OpenProjects[j].ID })
	sort.Slice(plan.OpenGroups, func(i, j int) bool { return plan.OpenGroups[i].ID < plan.OpenGroups[j].ID })
	plan.StateSignature = groupReopenStateSignature(projects, nodes)
	plan.Token = groupReopenToken(rootProjectID, rootGroupID, plan)
	return plan, nil
}

func simulateGroupReopenImpact(ctx context.Context, tx *gorm.DB, rootProjectID, rootGroupID string, reopenProjects, reopenGroups map[string]struct{}, schedule func(context.Context, string) error) (schedulingimpact.Error, error) {
	var scheduleErr error
	transactionErr := tx.Transaction(func(simulation *gorm.DB) error {
		projectIDs := mapStringKeys(reopenProjects)
		if len(projectIDs) > 0 {
			if err := simulation.Model(&projectModel{}).Where("id IN ? AND status = ?", projectIDs, "locked").Updates(map[string]any{"status": "open", "locked_execution_snapshot": nil, "locked_commitment_snapshot": nil}).Error; err != nil {
				return err
			}
		}
		groupIDs := mapStringKeys(reopenGroups)
		if len(groupIDs) > 0 {
			if err := simulation.Model(&nodeModel{}).Where("id IN ? AND group_local_status = ?", groupIDs, "locked").Updates(map[string]any{"group_local_status": "open", "group_locked_automatic_scheduling": nil, "group_locked_scheduling_start_date": nil}).Error; err != nil {
				return err
			}
		}
		simulationContext := sharedpersistence.WithTransaction(ctx, simulation)
		simulationContext = schedulingimpact.WithGroupPreviewOperation(simulationContext, rootProjectID, rootGroupID, schedulingimpact.ModeOrdinary)
		scheduleErr = schedule(simulationContext, rootProjectID)
		return errRollbackGroupReopenPreview
	})
	if transactionErr != nil && !errors.Is(transactionErr, errRollbackGroupReopenPreview) {
		return schedulingimpact.Error{}, transactionErr
	}
	if scheduleErr == nil {
		return schedulingimpact.Error{}, nil
	}
	var impact schedulingimpact.Error
	if errors.As(scheduleErr, &impact) {
		return impact, nil
	}
	return schedulingimpact.Error{}, fmt.Errorf("simulate Group reopen schedule: %w", scheduleErr)
}

func addGroupAndLockingAncestors(group nodeModel, nodes map[string]nodeModel, projects map[string]projectModel, groups, projectSet map[string]struct{}) bool {
	added := false
	current := &group
	for current != nil {
		if current.GroupLocalStatus == "locked" {
			if _, exists := groups[current.ID]; !exists {
				groups[current.ID] = struct{}{}
				added = true
			}
		}
		if current.ParentID == nil {
			break
		}
		parent, exists := nodes[*current.ParentID]
		if !exists {
			break
		}
		current = &parent
	}
	project := projects[group.ProjectID]
	if project.Status == "locked" {
		if _, exists := projectSet[group.ProjectID]; !exists {
			projectSet[group.ProjectID] = struct{}{}
			added = true
		}
	}
	return added
}

func filterProjectNodes(nodes []nodeModel, projectID string) []nodeModel {
	values := make([]nodeModel, 0)
	for _, node := range nodes {
		if node.ProjectID == projectID {
			values = append(values, node)
		}
	}
	return values
}

func qualifiedGroupPath(projectName, groupID string, nodes map[string]nodeModel) string {
	group, exists := nodes[groupID]
	if !exists {
		return projectName + " / " + groupID
	}
	parts := []string{group.Name}
	current := group.ParentID
	seen := map[string]bool{group.ID: true}
	for current != nil && !seen[*current] {
		seen[*current] = true
		parent, exists := nodes[*current]
		if !exists {
			break
		}
		parts = append(parts, parent.Name)
		current = parent.ParentID
	}
	for left, right := 0, len(parts)-1; left < right; left, right = left+1, right-1 {
		parts[left], parts[right] = parts[right], parts[left]
	}
	return strings.Join(append([]string{projectName}, parts...), " / ")
}

func groupReopenStateSignature(projects []projectModel, nodes []nodeModel) string {
	parts := make([]string, 0, len(projects)+len(nodes))
	projectValues := append([]projectModel{}, projects...)
	sort.Slice(projectValues, func(i, j int) bool { return projectValues[i].ID < projectValues[j].ID })
	for _, project := range projectValues {
		parts = append(parts, strings.Join([]string{
			"project", project.ID, project.Status, fmt.Sprintf("%d", project.ScheduleVersion),
			fmt.Sprintf("%t", project.AutomaticScheduling), dateToken(project.SchedulingStartDate),
			fmt.Sprintf("%d", project.ProjectBuffer), fmt.Sprintf("%d", project.Priority),
		}, ":"))
	}
	nodeValues := append([]nodeModel{}, nodes...)
	sort.Slice(nodeValues, func(i, j int) bool { return nodeValues[i].ID < nodeValues[j].ID })
	for _, node := range nodeValues {
		parentID := ""
		if node.ParentID != nil {
			parentID = *node.ParentID
		}
		automatic := "nil"
		if node.GroupAutomaticScheduling != nil {
			automatic = fmt.Sprintf("%t", *node.GroupAutomaticScheduling)
		}
		lockedAutomatic := "nil"
		if node.GroupLockedAutomaticScheduling != nil {
			lockedAutomatic = fmt.Sprintf("%t", *node.GroupLockedAutomaticScheduling)
		}
		parts = append(parts, strings.Join([]string{
			"node", node.ID, node.ProjectID, parentID, node.ParentKey, fmt.Sprintf("%d", node.Position),
			node.GroupSchedulingSource, automatic, dateToken(node.GroupSchedulingStartDate), node.GroupLocalStatus,
			fmt.Sprintf("%d", node.GroupSchedulingVersion), lockedAutomatic, dateToken(node.GroupLockedSchedulingStartDate),
		}, ":"))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(digest[:])
}

func dateToken(value *time.Time) string {
	if value == nil {
		return "nil"
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func groupReopenToken(rootProjectID, rootGroupID string, plan groupReopenPlan) string {
	parts := []string{rootProjectID, rootGroupID, plan.StateSignature}
	for _, project := range plan.LockedProjects {
		parts = append(parts, fmt.Sprintf("locked-project:%s:%d", project.ID, project.Version))
	}
	for _, group := range plan.LockedGroups {
		parts = append(parts, fmt.Sprintf("locked-group:%s:%d", group.ID, group.Version))
	}
	for _, project := range plan.OpenProjects {
		parts = append(parts, fmt.Sprintf("open-project:%s:%d", project.ID, project.Version))
	}
	for _, group := range plan.OpenGroups {
		parts = append(parts, fmt.Sprintf("open-group:%s:%d", group.ID, group.Version))
	}
	digest := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(digest[:])
}

func mapStringKeys(values map[string]struct{}) []string {
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	sort.Strings(keys)
	return keys
}

func subtreeHasLocked(resolver *groupscheduling.Resolver, all map[string]nodeModel, root string) bool {
	for id := range all {
		if id != root && !isDescendant(all, root, id) {
			continue
		}
		effective, err := resolver.EffectiveFor(id)
		if err == nil && effective.Lifecycle == groupscheduling.LifecycleLocked {
			return true
		}
	}
	return false
}

func shiftedLockedSibling(resolver *groupscheduling.Resolver, all map[string]nodeModel, parentKey string, afterPosition int, excludeID string) bool {
	for id, sibling := range all {
		if id == excludeID || sibling.ParentKey != parentKey || sibling.Position <= afterPosition {
			continue
		}
		if subtreeHasLocked(resolver, all, id) {
			return true
		}
	}
	return false
}

func reorderedLockedSibling(resolver *groupscheduling.Resolver, all map[string]nodeModel, before, after []nodeModel) bool {
	positions := make(map[string]int, len(before))
	for index, sibling := range before {
		positions[sibling.ID] = index
	}
	for index, sibling := range after {
		if previous, ok := positions[sibling.ID]; ok && previous != index && subtreeHasLocked(resolver, all, sibling.ID) {
			return true
		}
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
func differentTimeline(a, b domain.Timeline) bool {
	return differentDate(a.Start, b.Start) || differentDate(a.End, b.End)
}
func differentDate(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a != nil || b != nil
	}
	return !a.Equal(*b)
}
func fromDomain(n domain.Node) nodeModel {
	m := nodeModel{ID: n.ID, ProjectID: n.ProjectID, ParentID: n.ParentID, ParentKey: parentKey(n.ParentID), Name: n.Name, NameKey: domain.NameKey(n.Name), Position: n.Position, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt}
	applyFields(&m, n.Executable)
	return m
}
func applyFields(m *nodeModel, f domain.ExecutableFields) {
	m.RoleID = f.RoleID
	m.AssigneeID = f.AssigneeID
	m.EffortMinutes = f.EffortMinutes
	m.LagDays = f.LagDays
	m.CapacityAllocationPercentage = f.CapacityAllocationPercentage
	m.ExecutionStart = f.ExecutionTimeline.Start
	m.ExecutionEnd = f.ExecutionTimeline.End
	m.CommitmentStart = f.CommitmentTimeline.Start
	m.CommitmentEnd = f.CommitmentTimeline.End
	m.ExecutionUnscheduledReason = f.ExecutionUnscheduledReason
	m.CommitmentUnscheduledReason = f.CommitmentUnscheduledReason
	m.ActualStart = f.ActualStart
	m.ActualEnd = f.ActualEnd
}
func toDomain(m nodeModel, children bool) domain.Node {
	source := m.GroupSchedulingSource
	if source == "" {
		source = "inherit"
	}
	status := m.GroupLocalStatus
	if status == "" {
		status = "open"
	}
	return domain.Node{ID: m.ID, ProjectID: m.ProjectID, ParentID: m.ParentID, Name: m.Name, Position: m.Position, HasChildren: children, Executable: domain.ExecutableFields{RoleID: m.RoleID, AssigneeID: m.AssigneeID, EffortMinutes: m.EffortMinutes, LagDays: m.LagDays, CapacityAllocationPercentage: m.CapacityAllocationPercentage, ExecutionTimeline: domain.Timeline{Start: m.ExecutionStart, End: m.ExecutionEnd}, CommitmentTimeline: domain.Timeline{Start: m.CommitmentStart, End: m.CommitmentEnd}, ExecutionUnscheduledReason: m.ExecutionUnscheduledReason, CommitmentUnscheduledReason: m.CommitmentUnscheduledReason, ActualStart: m.ActualStart, ActualEnd: m.ActualEnd}, Scheduling: domain.GroupScheduling{Version: m.GroupSchedulingVersion, Source: source, AutomaticScheduling: m.GroupAutomaticScheduling, SchedulingStartDate: m.GroupSchedulingStartDate, LocalStatus: status}, Children: []domain.Node{}, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt}
}
func buildTree(models []nodeModel, resolver *groupscheduling.Resolver) []domain.Node {
	children := map[string][]nodeModel{}
	for _, m := range models {
		children[m.ParentKey] = append(children[m.ParentKey], m)
	}
	var walk func(string) []domain.Node
	walk = func(key string) []domain.Node {
		out := []domain.Node{}
		for _, m := range children[key] {
			n := toDomain(m, len(children[m.ID]) > 0)
			if effective, err := resolver.EffectiveFor(m.ID); err == nil {
				decorateEffective(&n, effective)
			}
			if inherited, err := resolver.InheritedFor(m.ID); err == nil {
				decorateInherited(&n, inherited)
			}
			n.Children = walk(m.ID)
			out = append(out, n)
		}
		return out
	}
	return walk("")
}

func decorateEffective(node *domain.Node, value groupscheduling.Effective) {
	node.Scheduling.EffectiveAutomatic = value.AutomaticScheduling
	node.Scheduling.EffectiveStartDate = value.SchedulingStartDate
	node.Scheduling.AutomaticSourceID = value.AutomaticOwnerID
	node.Scheduling.AutomaticSourceName = value.AutomaticOwnerName
	node.Scheduling.StartDateSourceID = value.StartDateOwnerID
	node.Scheduling.StartDateSourceName = value.StartDateOwnerName
	node.Scheduling.EffectiveLifecycle = string(value.Lifecycle)
	node.Scheduling.LockOwnerID = value.LockOwnerID
	node.Scheduling.LockOwnerName = value.LockOwnerName
}

func decorateInherited(node *domain.Node, value groupscheduling.Effective) {
	node.Scheduling.InheritedAutomatic = value.AutomaticScheduling
	node.Scheduling.InheritedStartDate = value.SchedulingStartDate
	node.Scheduling.InheritedAutomaticSourceID = value.AutomaticOwnerID
	node.Scheduling.InheritedAutomaticSourceName = value.AutomaticOwnerName
	node.Scheduling.InheritedStartDateSourceID = value.StartDateOwnerID
	node.Scheduling.InheritedStartDateSourceName = value.StartDateOwnerName
}

func newGroupResolver(project projectModel, models []nodeModel) (*groupscheduling.Resolver, error) {
	nodes := make([]groupscheduling.Node, 0, len(models))
	for _, model := range models {
		source := groupscheduling.Source(model.GroupSchedulingSource)
		if source == "" {
			source = groupscheduling.SourceInherit
		}
		status := groupscheduling.LocalStatus(model.GroupLocalStatus)
		if status == "" {
			status = groupscheduling.StatusOpen
		}
		nodes = append(nodes, groupscheduling.Node{ID: model.ID, ParentID: model.ParentID, Name: model.Name, Source: source, AutomaticSchedulingOverride: model.GroupAutomaticScheduling, SchedulingStartDateOverride: model.GroupSchedulingStartDate, LocalStatus: status, LockedAutomaticScheduling: model.GroupLockedAutomaticScheduling, LockedSchedulingStartDate: model.GroupLockedSchedulingStartDate})
	}
	return groupscheduling.New(groupscheduling.Project{ID: project.ID, Name: project.Name, Status: project.Status, AutomaticScheduling: project.AutomaticScheduling, SchedulingStartDate: project.SchedulingStartDate}, nodes)
}

func loadDecoratedNode(tx *gorm.DB, project projectModel, id string) (*domain.Node, error) {
	resolver, all, err := loadGroupResolver(tx, project, false)
	if err != nil {
		return nil, err
	}
	model, exists := all[id]
	if !exists {
		return nil, domain.ErrNotFound
	}
	node := toDomain(model, resolver.IsGrouping(id))
	effective, err := resolver.EffectiveFor(id)
	if err != nil {
		return nil, err
	}
	decorateEffective(&node, effective)
	inherited, err := resolver.InheritedFor(id)
	if err != nil {
		return nil, err
	}
	decorateInherited(&node, inherited)
	return &node, nil
}

func loadGroupResolver(tx *gorm.DB, project projectModel, lock bool) (*groupscheduling.Resolver, map[string]nodeModel, error) {
	var models []nodeModel
	query := tx.Where("project_id = ?", project.ID)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.Order("parent_key ASC").Order("position ASC").Order("id ASC").Find(&models).Error; err != nil {
		return nil, nil, err
	}
	resolver, err := newGroupResolver(project, models)
	if err != nil {
		return nil, nil, err
	}
	byID := make(map[string]nodeModel, len(models))
	for _, model := range models {
		byID[model.ID] = model
	}
	return resolver, byID, nil
}

func requirePlanningOpen(resolver *groupscheduling.Resolver, id string) (groupscheduling.Effective, error) {
	effective, err := resolver.EffectiveFor(id)
	if err != nil {
		return effective, err
	}
	if effective.Lifecycle == groupscheduling.LifecycleClosed {
		return effective, domain.ErrProjectClosedReadOnly
	}
	if effective.Lifecycle == groupscheduling.LifecycleLocked {
		if effective.LockOwnerID == resolver.ProjectID() {
			return effective, domain.ErrProjectLockedReadOnly
		}
		return effective, domain.ErrGroupLockedReadOnly
	}
	return effective, nil
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
