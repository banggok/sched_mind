package gormrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/portfolio/application"
	"github.com/banggok/sched_mind/backend/internal/portfolio/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ database *gorm.DB }

func New(database *gorm.DB) *Repository { return &Repository{database: database} }

type projectRow struct {
	ID              string
	Name            string
	Status          string
	Priority        int
	ScheduleVersion int64
}

type nodeRow struct {
	ID                string
	ProjectID         string
	ParentID          *string
	Name              string
	Position          int
	RoleID            *string
	RoleName          *string
	AssigneeID        *string
	AssigneeName      *string
	EffortMinutes     *int
	Start             *time.Time
	End               *time.Time
	UnscheduledReason *string
	ActualStart       *time.Time
	ActualEnd         *time.Time
}

type dependencyRow struct {
	ID             string
	BlockingTaskID string
	BlockedTaskID  string
}

type holidayRow struct {
	Date        time.Time
	Description string
}

func (repository *Repository) ActiveProjects(ctx context.Context) ([]domain.ProjectOption, error) {
	var rows []projectRow
	if err := repository.database.WithContext(ctx).
		Table("projects").
		Select("id, name, status, priority, schedule_version").
		Where("status IN ?", []string{"open", "locked"}).
		Order("priority ASC").Order("id ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("query active projects: %w", err)
	}
	values := make([]domain.ProjectOption, 0, len(rows))
	for _, row := range rows {
		values = append(values, domain.ProjectOption{ID: row.ID, Name: row.Name, Status: row.Status, Priority: row.Priority, ScheduleVersion: row.ScheduleVersion})
	}
	return values, nil
}

func (repository *Repository) Portfolio(ctx context.Context, query application.PortfolioQuery) (*domain.Portfolio, error) {
	portfolio := &domain.Portfolio{
		Projection:   query.Projection,
		Projects:     []domain.ProjectOption{},
		Rows:         []domain.Row{},
		Dependencies: []domain.Dependency{},
		Holidays:     []domain.Holiday{},
	}
	if err := repository.loadHolidays(ctx, query.From, query.To, portfolio); err != nil {
		return nil, err
	}
	if len(query.ProjectIDs) == 0 {
		return portfolio, nil
	}

	var projects []projectRow
	if err := repository.database.WithContext(ctx).
		Table("projects").
		Select("id, name, status, priority, schedule_version").
		Where("id IN ? AND status IN ?", query.ProjectIDs, []string{"open", "locked"}).
		Order("priority ASC").Order("id ASC").
		Scan(&projects).Error; err != nil {
		return nil, fmt.Errorf("query selected portfolio projects: %w", err)
	}
	selectedIDs := make([]string, 0, len(projects))
	for _, project := range projects {
		portfolio.Projects = append(portfolio.Projects, domain.ProjectOption{ID: project.ID, Name: project.Name, Status: project.Status, Priority: project.Priority, ScheduleVersion: project.ScheduleVersion})
		selectedIDs = append(selectedIDs, project.ID)
	}
	if len(selectedIDs) == 0 {
		return portfolio, nil
	}

	startColumn, endColumn, reasonColumn := "execution_start", "execution_end", "execution_unscheduled_reason"
	if query.Projection == domain.Commitment {
		startColumn, endColumn, reasonColumn = "commitment_start", "commitment_end", "commitment_unscheduled_reason"
	}
	selectClause := fmt.Sprintf("node.id, node.project_id, node.parent_id, node.name, node.position, node.role_id, role.name AS role_name, node.assignee_id, member.name AS assignee_name, node.effort_minutes, node.%s AS start, node.%s AS end, node.%s AS unscheduled_reason, node.actual_start, node.actual_end", startColumn, endColumn, reasonColumn)
	var nodes []nodeRow
	if err := repository.database.WithContext(ctx).
		Table("wbs_nodes AS node").
		Select(selectClause).
		Joins("LEFT JOIN roles AS role ON role.id = node.role_id").
		Joins("LEFT JOIN team_members AS member ON member.id = node.assignee_id").
		Where("node.project_id IN ?", selectedIDs).
		Order("node.project_id ASC").Order("node.parent_key ASC").Order("node.position ASC").Order("node.id ASC").
		Scan(&nodes).Error; err != nil {
		return nil, fmt.Errorf("query portfolio WBS: %w", err)
	}
	portfolio.Rows, portfolio.WorkingDayAnchor = composeRows(projects, nodes)

	var dependencies []dependencyRow
	if err := repository.database.WithContext(ctx).
		Table("task_dependencies AS dependency").
		Select("dependency.id, dependency.blocking_task_id, dependency.blocked_task_id").
		Joins("JOIN wbs_nodes AS blocking ON blocking.id = dependency.blocking_task_id").
		Joins("JOIN wbs_nodes AS blocked ON blocked.id = dependency.blocked_task_id").
		Where("blocking.project_id IN ? AND blocked.project_id IN ?", selectedIDs, selectedIDs).
		Order("dependency.blocking_task_id ASC").Order("dependency.blocked_task_id ASC").Order("dependency.id ASC").
		Scan(&dependencies).Error; err != nil {
		return nil, fmt.Errorf("query portfolio dependencies: %w", err)
	}
	for _, dependency := range dependencies {
		portfolio.Dependencies = append(portfolio.Dependencies, domain.Dependency{ID: dependency.ID, BlockingTaskID: dependency.BlockingTaskID, BlockedTaskID: dependency.BlockedTaskID})
	}
	return portfolio, nil
}

func (repository *Repository) loadHolidays(ctx context.Context, from, to time.Time, portfolio *domain.Portfolio) error {
	var rows []holidayRow
	if err := repository.database.WithContext(ctx).
		Table("public_holiday_dates AS holiday_date").
		Select("holiday_date.date, holiday.description").
		Joins("JOIN public_holidays AS holiday ON holiday.id = holiday_date.public_holiday_id").
		Where("holiday_date.date BETWEEN ? AND ?", from, to).
		Order("holiday_date.date ASC").Order("holiday.id ASC").
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("query portfolio holidays: %w", err)
	}
	seen := make(map[string]struct{})
	for _, row := range rows {
		key := row.Date.Format("2006-01-02")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		portfolio.Holidays = append(portfolio.Holidays, domain.Holiday{Date: row.Date, Description: row.Description})
	}
	return nil
}

type aggregate struct {
	effort             int
	hasKnownEffort     bool
	incompleteEffort   bool
	start              *time.Time
	end                *time.Time
	incompleteSchedule bool
}

func composeRows(projects []projectRow, nodes []nodeRow) ([]domain.Row, *time.Time) {
	byProject := make(map[string][]nodeRow)
	for _, node := range nodes {
		byProject[node.ProjectID] = append(byProject[node.ProjectID], node)
	}
	rows := make([]domain.Row, 0, len(nodes)+len(projects))
	var anchor *time.Time
	for _, project := range projects {
		projectNodes := byProject[project.ID]
		byParent := make(map[string][]nodeRow)
		for _, node := range projectNodes {
			key := ""
			if node.ParentID != nil {
				key = *node.ParentID
			}
			byParent[key] = append(byParent[key], node)
		}
		for key := range byParent {
			sort.Slice(byParent[key], func(left, right int) bool {
				if byParent[key][left].Position != byParent[key][right].Position {
					return byParent[key][left].Position < byParent[key][right].Position
				}
				return byParent[key][left].ID < byParent[key][right].ID
			})
		}

		aggregates := make(map[string]aggregate, len(projectNodes))
		var calculate func(nodeRow) aggregate
		calculate = func(node nodeRow) aggregate {
			if value, exists := aggregates[node.ID]; exists {
				return value
			}
			children := byParent[node.ID]
			value := aggregate{}
			if len(children) > 0 {
				for _, child := range children {
					value = mergeAggregate(value, calculate(child))
				}
			} else {
				if node.EffortMinutes == nil {
					value.incompleteEffort = true
				} else {
					value.effort = *node.EffortMinutes
					value.hasKnownEffort = true
				}
				if node.Start != nil && node.End != nil {
					value.start = cloneDate(node.Start)
					value.end = cloneDate(node.End)
					if anchor == nil || node.Start.Before(*anchor) {
						anchor = cloneDate(node.Start)
					}
				} else {
					value.incompleteSchedule = true
				}
				if node.UnscheduledReason != nil {
					value.incompleteSchedule = true
				}
			}
			aggregates[node.ID] = value
			return value
		}

		projectAggregate := aggregate{}
		for _, root := range byParent[""] {
			projectAggregate = mergeAggregate(projectAggregate, calculate(root))
		}
		var projectEffort *int
		if projectAggregate.hasKnownEffort {
			copy := projectAggregate.effort
			projectEffort = &copy
		}
		rows = append(rows, domain.Row{
			ID: project.ID, ProjectID: project.ID, Kind: "project", Name: project.Name, Status: project.Status,
			EffortMinutes: projectEffort, Start: cloneDate(projectAggregate.start), End: cloneDate(projectAggregate.end),
			IncompleteEffort: projectAggregate.incompleteEffort, IncompleteSchedule: projectAggregate.incompleteSchedule,
			HasChildren: len(projectNodes) > 0,
		})

		var appendNode func(nodeRow, string, int)
		appendNode = func(node nodeRow, number string, depth int) {
			children := byParent[node.ID]
			value := aggregates[node.ID]
			kind := "task"
			if len(children) > 0 {
				kind = "group"
			}
			var effort *int
			if value.hasKnownEffort {
				copy := value.effort
				effort = &copy
			}
			row := domain.Row{
				ID: node.ID, ProjectID: node.ProjectID, ParentID: node.ParentID, Kind: kind,
				Name: node.Name, WBSNumber: number, Depth: depth, Position: node.Position,
				RoleID: node.RoleID, RoleName: node.RoleName,
				AssigneeID: node.AssigneeID, AssigneeName: node.AssigneeName, EffortMinutes: effort,
				Start: cloneDate(value.start), End: cloneDate(value.end), UnscheduledReason: node.UnscheduledReason,
				IncompleteEffort: value.incompleteEffort, IncompleteSchedule: value.incompleteSchedule,
				HasChildren: len(children) > 0,
				Completed:   node.ActualStart != nil && node.ActualEnd != nil,
			}
			if kind != "task" {
				row.AssigneeID = nil
				row.AssigneeName = nil
				row.UnscheduledReason = nil
			}
			rows = append(rows, row)
			for index, child := range children {
				appendNode(child, fmt.Sprintf("%s.%d", number, index+1), depth+1)
			}
		}
		for index, root := range byParent[""] {
			appendNode(root, fmt.Sprintf("%d", index+1), 1)
		}
	}
	return rows, anchor
}

func mergeAggregate(left, right aggregate) aggregate {
	left.effort += right.effort
	left.hasKnownEffort = left.hasKnownEffort || right.hasKnownEffort
	left.incompleteEffort = left.incompleteEffort || right.incompleteEffort
	left.incompleteSchedule = left.incompleteSchedule || right.incompleteSchedule
	if right.start != nil && (left.start == nil || right.start.Before(*left.start)) {
		left.start = cloneDate(right.start)
	}
	if right.end != nil && (left.end == nil || right.end.After(*left.end)) {
		left.end = cloneDate(right.end)
	}
	return left
}

func cloneDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	year, month, day := value.Date()
	copy := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	return &copy
}

func (repository *Repository) ListSavedFilters(ctx context.Context) ([]domain.SavedFilter, error) {
	var models []savedFilterModel
	if err := repository.database.WithContext(ctx).Order("name_key ASC").Order("id ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("query saved filters: %w", err)
	}
	values := make([]domain.SavedFilter, 0, len(models))
	for _, model := range models {
		value, err := savedFilterToDomain(model)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func (repository *Repository) CreateSavedFilter(ctx context.Context, value domain.SavedFilter) (*domain.SavedFilter, error) {
	name, key, err := domain.NormalizeFilterName(value.Name)
	if err != nil {
		return nil, err
	}
	var created *domain.SavedFilter
	err = repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		projectIDs, err := activeProjectIDs(tx, value.ProjectIDs)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(projectIDs)
		if err != nil {
			return fmt.Errorf("encode saved filter projects: %w", err)
		}
		model := savedFilterModel{ID: value.ID, Name: name, NameKey: key, ProjectIDs: payload, Version: 1, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
		if err := tx.Create(&model).Error; err != nil {
			if isNameConflict(err) {
				return domain.ErrFilterNameExists
			}
			return fmt.Errorf("insert saved filter: %w", err)
		}
		result, err := savedFilterToDomain(model)
		if err != nil {
			return err
		}
		created = &result
		return nil
	})
	return created, err
}

func (repository *Repository) UpdateSavedFilter(ctx context.Context, id string, version int64, projectIDs []string, now time.Time) (*domain.SavedFilter, error) {
	var updated *domain.SavedFilter
	err := repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model savedFilterModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrFilterNotFound
		}
		if err != nil {
			return fmt.Errorf("load saved filter for update: %w", err)
		}
		if model.Version != version {
			return domain.ErrFilterConflict
		}
		activeIDs, err := activeProjectIDs(tx, projectIDs)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(activeIDs)
		if err != nil {
			return fmt.Errorf("encode saved filter projects: %w", err)
		}
		result := tx.Model(&savedFilterModel{}).Where("id = ? AND version = ?", id, version).Updates(map[string]any{"project_ids": payload, "version": version + 1, "updated_at": now})
		if result.Error != nil {
			return fmt.Errorf("update saved filter: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return domain.ErrFilterConflict
		}
		model.ProjectIDs = payload
		model.Version++
		model.UpdatedAt = now
		value, err := savedFilterToDomain(model)
		if err != nil {
			return err
		}
		updated = &value
		return nil
	})
	return updated, err
}

func (repository *Repository) DeleteSavedFilter(ctx context.Context, id string, version int64) error {
	return repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var model savedFilterModel
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&model, "id = ?", id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrFilterNotFound
		}
		if err != nil {
			return fmt.Errorf("load saved filter for delete: %w", err)
		}
		if model.Version != version {
			return domain.ErrFilterConflict
		}
		result := tx.Where("id = ? AND version = ?", id, version).Delete(&savedFilterModel{})
		if result.Error != nil {
			return fmt.Errorf("delete saved filter: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return domain.ErrFilterConflict
		}
		return nil
	})
}

func activeProjectIDs(database *gorm.DB, requested []string) ([]string, error) {
	if len(requested) == 0 {
		return []string{}, nil
	}
	var ids []string
	if err := database.Table("projects").Where("id IN ? AND status IN ?", requested, []string{"open", "locked"}).Order("id ASC").Pluck("id", &ids).Error; err != nil {
		return nil, fmt.Errorf("validate saved filter projects: %w", err)
	}
	return ids, nil
}

func savedFilterToDomain(model savedFilterModel) (domain.SavedFilter, error) {
	projectIDs := []string{}
	if len(model.ProjectIDs) > 0 {
		if err := json.Unmarshal(model.ProjectIDs, &projectIDs); err != nil {
			return domain.SavedFilter{}, fmt.Errorf("decode saved filter projects: %w", err)
		}
	}
	return domain.SavedFilter{ID: model.ID, Name: model.Name, ProjectIDs: projectIDs, Version: model.Version, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt}, nil
}

func isNameConflict(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "portfolio_saved_filters_name") || strings.Contains(message, "unique constraint")
}

var _ application.Repository = (*Repository)(nil)
