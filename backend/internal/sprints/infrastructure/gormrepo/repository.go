package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/sprints/application"
	"github.com/banggok/sched_mind/backend/internal/sprints/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const sprintMutationAdvisoryLockKey int64 = 76081

var sprintMutationMutex sync.Mutex

type Repository struct{ database *gorm.DB }

func New(database *gorm.DB) *Repository { return &Repository{database: database} }

func (repository *Repository) List(ctx context.Context, query listing.Query) (listing.Page[domain.Sprint], error) {
	statement := repository.database.WithContext(ctx).Model(&sprintModel{})
	if search := strings.ToLower(strings.TrimSpace(query.Search)); search != "" {
		statement = statement.Where("name_key LIKE ?", search+"%")
	}
	var total int64
	if err := statement.Count(&total).Error; err != nil {
		return listing.Page[domain.Sprint]{}, fmt.Errorf("count sprints: %w", err)
	}
	var models []sprintModel
	if err := statement.Order("start_date DESC").Order("end_date DESC").Order("name_key ASC").Order("id ASC").
		Limit(query.PageSize).Offset((query.Page - 1) * query.PageSize).Find(&models).Error; err != nil {
		return listing.Page[domain.Sprint]{}, fmt.Errorf("list sprints: %w", err)
	}
	items := make([]domain.Sprint, 0, len(models))
	for _, model := range models {
		items = append(items, toDomain(model, nil, nil))
	}
	return listing.Page[domain.Sprint]{Items: items, Page: query.Page, PageSize: query.PageSize, Total: total}, nil
}

func (repository *Repository) Find(ctx context.Context, id string) (*domain.Sprint, error) {
	return repository.find(repository.database.WithContext(ctx), id)
}

func (repository *Repository) find(database *gorm.DB, id string) (*domain.Sprint, error) {
	var model sprintModel
	if err := database.First(&model, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, application.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("find sprint: %w", err)
	}
	var members []sprintMemberModel
	if err := database.Where("sprint_id = ?", id).Order("member_id ASC").Find(&members).Error; err != nil {
		return nil, fmt.Errorf("find sprint members: %w", err)
	}
	var tasks []sprintTaskModel
	if err := database.Where("sprint_id = ?", id).Order("task_id ASC").Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("find sprint tasks: %w", err)
	}
	memberIDs := make([]string, 0, len(members))
	for _, member := range members {
		memberIDs = append(memberIDs, member.MemberID)
	}
	taskIDs := make([]string, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.TaskID)
	}
	sprint := toDomain(model, memberIDs, taskIDs)
	return &sprint, nil
}

func (repository *Repository) Create(ctx context.Context, sprint domain.Sprint) error {
	return repository.mutate(ctx, func(tx *gorm.DB) error {
		if err := validateMembers(tx, sprint.MemberIDs); err != nil {
			return err
		}
		if err := validateTasks(tx, sprint.TaskIDs); err != nil {
			return err
		}
		if err := ensureNoOverlap(tx, sprint.ID, sprint.StartDate, sprint.EndDate, sprint.MemberIDs); err != nil {
			return err
		}
		if err := tx.Create(fromDomain(sprint)).Error; err != nil {
			return mapWriteError(err)
		}
		return replaceRelations(tx, sprint.ID, sprint.MemberIDs, sprint.TaskIDs)
	})
}

func (repository *Repository) Update(ctx context.Context, sprint domain.Sprint, oldVersion int64, priorTasks map[string]struct{}) error {
	return repository.mutate(ctx, func(tx *gorm.DB) error {
		var current sprintModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", sprint.ID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return application.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("lock sprint: %w", err)
		}
		if current.Version != oldVersion {
			return application.ErrVersionConflict
		}
		if err := validateMembers(tx, sprint.MemberIDs); err != nil {
			return err
		}
		newTasks := make([]string, 0, len(sprint.TaskIDs))
		for _, taskID := range sprint.TaskIDs {
			if _, retained := priorTasks[taskID]; !retained {
				newTasks = append(newTasks, taskID)
			}
		}
		if err := validateTasks(tx, newTasks); err != nil {
			return err
		}
		if err := ensureNoOverlap(tx, sprint.ID, sprint.StartDate, sprint.EndDate, sprint.MemberIDs); err != nil {
			return err
		}
		result := tx.Model(&sprintModel{}).Where("id = ? AND version = ?", sprint.ID, oldVersion).Updates(map[string]any{
			"name": sprint.Name, "name_key": sprint.NormalizedName, "start_date": sprint.StartDate, "end_date": sprint.EndDate,
			"version": sprint.Version, "updated_at": sprint.UpdatedAt,
		})
		if result.Error != nil {
			return mapWriteError(result.Error)
		}
		if result.RowsAffected != 1 {
			return application.ErrVersionConflict
		}
		return replaceRelations(tx, sprint.ID, sprint.MemberIDs, sprint.TaskIDs)
	})
}

func (repository *Repository) Start(ctx context.Context, id string, version int64, now time.Time) (*domain.Sprint, error) {
	var started *domain.Sprint
	err := repository.mutate(ctx, func(tx *gorm.DB) error {
		var current sprintModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return application.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("lock sprint for start: %w", err)
		}
		if current.Version != version {
			return application.ErrVersionConflict
		}
		if current.Status == string(domain.StatusStarted) {
			return domain.ErrAlreadyStarted
		}
		result := tx.Model(&sprintModel{}).Where("id = ? AND version = ?", id, version).Updates(map[string]any{
			"status": string(domain.StatusStarted), "started_at": now, "updated_at": now, "version": version + 1,
		})
		if result.Error != nil {
			return fmt.Errorf("start sprint: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return application.ErrVersionConflict
		}
		value, err := repository.find(tx, id)
		started = value
		return err
	})
	return started, err
}

func (repository *Repository) Delete(ctx context.Context, id string, version int64) error {
	return repository.mutate(ctx, func(tx *gorm.DB) error {
		var current sprintModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return application.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("lock sprint for delete: %w", err)
		}
		if current.Version != version {
			return application.ErrVersionConflict
		}
		if err := tx.Where("sprint_id = ?", id).Delete(&sprintMemberModel{}).Error; err != nil {
			return fmt.Errorf("delete sprint members: %w", err)
		}
		if err := tx.Where("sprint_id = ?", id).Delete(&sprintTaskModel{}).Error; err != nil {
			return fmt.Errorf("delete sprint tasks: %w", err)
		}
		result := tx.Where("id = ? AND version = ?", id, version).Delete(&sprintModel{})
		if result.Error != nil {
			return fmt.Errorf("delete sprint: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return application.ErrVersionConflict
		}
		return nil
	})
}

func (repository *Repository) mutate(ctx context.Context, operation func(*gorm.DB) error) error {
	if repository.database.Dialector.Name() == "postgres" {
		return repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", sprintMutationAdvisoryLockKey).Error; err != nil {
				return fmt.Errorf("lock sprint mutation: %w", err)
			}
			return operation(tx)
		})
	}
	sprintMutationMutex.Lock()
	defer sprintMutationMutex.Unlock()
	return repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return operation(tx)
	})
}

func validateMembers(tx *gorm.DB, memberIDs []string) error {
	var count int64
	if err := tx.Model(&memberModel{}).Where("id IN ? AND deleted_at IS NULL", memberIDs).Count(&count).Error; err != nil {
		return fmt.Errorf("validate sprint members: %w", err)
	}
	if count != int64(len(memberIDs)) {
		return application.ErrMemberNotFound
	}
	return nil
}

func validateTasks(tx *gorm.DB, taskIDs []string) error {
	if len(taskIDs) == 0 {
		return nil
	}
	var count int64
	if err := tx.Table("wbs_nodes").Where("id IN ?", taskIDs).Count(&count).Error; err != nil {
		return fmt.Errorf("validate sprint task existence: %w", err)
	}
	if count != int64(len(taskIDs)) {
		return application.ErrTaskNotFound
	}
	if err := tx.Table("wbs_nodes").Where("id IN ?", taskIDs).
		Where("execution_start IS NULL OR execution_end IS NULL").Count(&count).Error; err != nil {
		return fmt.Errorf("validate sprint task schedule: %w", err)
	}
	if count > 0 {
		return application.ErrTaskUnscheduled
	}
	query := tx.Table("wbs_nodes AS task").Joins("JOIN projects AS project ON project.id = task.project_id").
		Where("task.id IN ?", taskIDs).Where("task.assignee_id IS NOT NULL").
		Where("task.execution_start IS NOT NULL AND task.execution_end IS NOT NULL").
		Where("task.actual_start IS NULL AND task.actual_end IS NULL").
		Where("project.status IN ?", []string{"open", "locked"}).
		Where("NOT EXISTS (?)", tx.Table("wbs_nodes AS child").Select("1").Where("child.project_id = task.project_id AND child.parent_id = task.id"))
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("validate sprint tasks: %w", err)
	}
	if count != int64(len(taskIDs)) {
		return application.ErrTaskNotEligible
	}
	return nil
}

func ensureNoOverlap(tx *gorm.DB, excludedID string, startDate, endDate time.Time, memberIDs []string) error {
	type overlapRow struct {
		SprintID, SprintName, MemberID, MemberName string
		StartDate, EndDate                         time.Time
	}
	var rows []overlapRow
	query := tx.Table("sprints AS sprint").Joins("JOIN sprint_members AS member ON member.sprint_id = sprint.id").
		Joins("JOIN team_members AS team_member ON team_member.id = member.member_id").
		Where("sprint.id <> ?", excludedID).Where("sprint.start_date <= ? AND ? <= sprint.end_date", endDate, startDate).
		Where("member.member_id IN ?", memberIDs).
		Select("sprint.id AS sprint_id, sprint.name AS sprint_name, sprint.start_date, sprint.end_date, member.member_id, team_member.name AS member_name").
		Order("sprint.start_date ASC").Order("sprint.end_date ASC").Order("sprint.name_key ASC").Order("sprint.id ASC").Order("member.member_id ASC")
	if err := query.Scan(&rows).Error; err != nil {
		return fmt.Errorf("check sprint overlap: %w", err)
	}
	if len(rows) > 0 {
		first := rows[0]
		conflict := &application.MemberOverlapError{
			SprintID: first.SprintID, SprintName: first.SprintName, StartDate: first.StartDate, EndDate: first.EndDate,
		}
		for _, row := range rows {
			if row.SprintID != first.SprintID {
				break
			}
			conflict.Members = append(conflict.Members, application.OverlapMember{ID: row.MemberID, Name: row.MemberName})
		}
		return conflict
	}
	return nil
}

func replaceRelations(tx *gorm.DB, sprintID string, memberIDs, taskIDs []string) error {
	if err := tx.Where("sprint_id = ?", sprintID).Delete(&sprintMemberModel{}).Error; err != nil {
		return fmt.Errorf("replace sprint members: %w", err)
	}
	if err := tx.Where("sprint_id = ?", sprintID).Delete(&sprintTaskModel{}).Error; err != nil {
		return fmt.Errorf("replace sprint tasks: %w", err)
	}
	for _, memberID := range memberIDs {
		if err := tx.Create(&sprintMemberModel{SprintID: sprintID, MemberID: memberID}).Error; err != nil {
			return mapWriteError(err)
		}
	}
	for _, taskID := range taskIDs {
		if err := tx.Create(&sprintTaskModel{SprintID: sprintID, TaskID: taskID}).Error; err != nil {
			return mapWriteError(err)
		}
	}
	return nil
}

func fromDomain(sprint domain.Sprint) sprintModel {
	return sprintModel{ID: sprint.ID, Name: sprint.Name, NameKey: sprint.NormalizedName, StartDate: sprint.StartDate, EndDate: sprint.EndDate,
		Status: string(sprint.Status), Version: sprint.Version, StartedAt: sprint.StartedAt, CreatedAt: sprint.CreatedAt, UpdatedAt: sprint.UpdatedAt}
}

func toDomain(model sprintModel, memberIDs, taskIDs []string) domain.Sprint {
	return domain.Sprint{ID: model.ID, Name: model.Name, NormalizedName: model.NameKey, StartDate: model.StartDate, EndDate: model.EndDate,
		Status: domain.Status(model.Status), Version: model.Version, StartedAt: model.StartedAt, CreatedAt: model.CreatedAt, UpdatedAt: model.UpdatedAt,
		MemberIDs: memberIDs, TaskIDs: taskIDs}
}

func mapWriteError(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(strings.ToLower(err.Error()), "unique") {
		return application.ErrNameConflict
	}
	return fmt.Errorf("persist sprint: %w", err)
}
