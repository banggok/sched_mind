package gormrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/projects/application"
	"github.com/banggok/sched_mind/backend/internal/projects/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
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
	if value := escapeLike(domain.NormalizedNameKey(search)); value != "" {
		base = base.Where("name_key LIKE ?", value+"%")
	}
	return base
}

func (r *Repository) listSegment(ctx context.Context, search string, active bool, offset, limit int) ([]projectModel, error) {
	statement := r.listBase(ctx, search)
	if active {
		statement = statement.Where("status IN ?", activeStatuses()).Order("priority ASC").Order("CASE WHEN start_date IS NULL THEN 1 ELSE 0 END ASC").Order("start_date ASC").Order("CASE WHEN end_date IS NULL THEN 1 ELSE 0 END ASC").Order("end_date ASC").Order("CASE WHEN status = 'locked' THEN 0 ELSE 1 END ASC").Order("id ASC")
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

func (r *Repository) CreateNext(ctx context.Context, id, name string, automaticScheduling bool, projectBuffer int, now time.Time) (*domain.Project, error) {
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
			if err := value.UpdateSettings(automaticScheduling, projectBuffer, now); err != nil {
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

func (r *Repository) UpdateDetails(ctx context.Context, id, name string, automaticScheduling bool, projectBuffer int, now time.Time, schedule func(context.Context, string) error) (*domain.Project, error) {
	var changed *domain.Project
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		value, err := find(tx, id, true)
		if err != nil {
			return err
		}
		if value == nil {
			return errors.New("update project: find returned nil")
		}
		wasAutomatic := value.AutomaticScheduling
		if err := value.Rename(name, now); err != nil {
			return err
		}
		if value.AutomaticScheduling != automaticScheduling || value.ProjectBuffer != projectBuffer {
			if err := value.UpdateSettings(automaticScheduling, projectBuffer, now); err != nil {
				return err
			}
		}
		result := tx.Model(&projectModel{}).Where("id = ?", id).Updates(map[string]interface{}{"name": value.Name, "name_key": domain.NormalizedNameKey(value.Name), "automatic_scheduling": value.AutomaticScheduling, "project_buffer": value.ProjectBuffer, "updated_at": value.UpdatedAt})
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return domain.ErrNameExists
		}
		if result.Error != nil {
			return result.Error
		}
		if !wasAutomatic && automaticScheduling {
			if err := schedule(ctx, id); err != nil {
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

func (r *Repository) ChangeStatus(ctx context.Context, id string, target domain.Status, now time.Time) (*domain.Project, error) {
	var changed *domain.Project
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		value, err := find(tx, id, true)
		if err != nil {
			return err
		}
		if value == nil {
			return errors.New("change status: find returned nil")
		}
		// WBS is not implemented. Every Project created in this story therefore has
		// zero leaves, so lock and close are rejected without inventing rows.
		if err := value.ChangeStatus(target, false, false, nil, nil, now); err != nil {
			return err
		}
		if err := tx.Model(&projectModel{}).Where("id = ?", value.ID).Updates(statusUpdates(*value)).Error; err != nil {
			return err
		}
		changed = value
		return nil
	})
	if err != nil {
		return nil, err
	}
	return changed, nil
}

func (r *Repository) MovePriority(ctx context.Context, id string, direction domain.PriorityDirection, now time.Time, schedule func(context.Context) error) (*domain.Project, error) {
	var changed *domain.Project
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
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
		if err := schedule(ctx); err != nil {
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

func (r *Repository) UpdateSettings(ctx context.Context, id string, automaticScheduling bool, projectBuffer int, now time.Time, schedule func(context.Context, string) error) (*domain.Project, error) {
	var changed *domain.Project
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		value, err := find(tx, id, true)
		if err != nil {
			return err
		}
		if value == nil {
			return errors.New("update settings: find returned nil")
		}
		wasAutomatic := value.AutomaticScheduling
		if err := value.UpdateSettings(automaticScheduling, projectBuffer, now); err != nil {
			return err
		}
		if err := tx.Model(&projectModel{}).Where("id = ?", id).Updates(map[string]interface{}{"automatic_scheduling": value.AutomaticScheduling, "project_buffer": value.ProjectBuffer, "updated_at": value.UpdatedAt}).Error; err != nil {
			return err
		}
		if !wasAutomatic && automaticScheduling {
			if err := schedule(ctx, id); err != nil {
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
	return &projectModel{ID: value.ID, Name: value.Name, NameKey: domain.NormalizedNameKey(value.Name), Status: string(value.Status), StartDate: value.StartDate, EndDate: value.EndDate, AutoCalculateDate: value.AutoCalculateDate, AutoDependencyByAssignee: value.AutoDependencyByAssignee, AutomaticScheduling: value.AutomaticScheduling, ProjectBuffer: value.ProjectBuffer, Priority: value.Priority, ClosedAt: value.ClosedAt, LockedExecutionSnapshot: value.LockedExecutionSnapshot, LockedCommitmentSnapshot: value.LockedCommitmentSnapshot, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func toDomain(model projectModel) (*domain.Project, error) {
	return domain.Rehydrate(model.ID, model.Name, domain.Status(model.Status), model.StartDate, model.EndDate, model.AutoCalculateDate, model.AutoDependencyByAssignee, model.AutomaticScheduling, model.ProjectBuffer, model.Priority, model.ClosedAt, model.LockedExecutionSnapshot, model.LockedCommitmentSnapshot, model.CreatedAt, model.UpdatedAt)
}
func statusUpdates(value domain.Project) map[string]interface{} {
	return map[string]interface{}{"status": value.Status, "closed_at": value.ClosedAt, "locked_execution_snapshot": value.LockedExecutionSnapshot, "locked_commitment_snapshot": value.LockedCommitmentSnapshot, "updated_at": value.UpdatedAt}
}
func escapeLike(value string) string {
	return strings.NewReplacer("\\", "\\\\", "%", "\\%", "_", "\\_").Replace(value)
}

var _ application.Store = (*Repository)(nil)
