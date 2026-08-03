package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"github.com/banggok/sched_mind/backend/internal/teammembers/application"
	"github.com/banggok/sched_mind/backend/internal/teammembers/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	database *gorm.DB
}

func New(database *gorm.DB) *Repository {
	return &Repository{database: database}
}

func (repository *Repository) List(
	ctx context.Context,
	query listing.Query,
) (listing.Page[application.TeamMemberRecord], error) {
	var models []teamMemberModel
	statement := repository.database.WithContext(ctx).Model(&teamMemberModel{})
	if search := strings.ToLower(strings.TrimSpace(query.Search)); search != "" {
		statement = statement.Where("LOWER(name) LIKE ?", sharedpersistence.EscapeLike(search)+"%")
	}
	var total int64
	if err := statement.Count(&total).Error; err != nil {
		return listing.Page[application.TeamMemberRecord]{}, fmt.Errorf("count team member models: %w", err)
	}
	if err := statement.Preload("Role").Order("LOWER(name) ASC").Order("id ASC").
		Limit(query.PageSize).Offset((query.Page - 1) * query.PageSize).
		Find(&models).Error; err != nil {
		return listing.Page[application.TeamMemberRecord]{}, fmt.Errorf("list team member models: %w", err)
	}
	records := make([]application.TeamMemberRecord, 0, len(models))
	for _, model := range models {
		record, err := toRecord(model)
		if err != nil {
			return listing.Page[application.TeamMemberRecord]{}, err
		}
		if record == nil {
			return listing.Page[application.TeamMemberRecord]{}, errors.New("map team member model: result is nil")
		}
		records = append(records, *record)
	}
	return listing.Page[application.TeamMemberRecord]{
		Items: records, Page: query.Page, PageSize: query.PageSize, Total: total,
	}, nil
}

func (repository *Repository) FindByID(
	ctx context.Context,
	id string,
) (*application.TeamMemberRecord, error) {
	var model teamMemberModel
	err := repository.database.WithContext(ctx).
		Preload("Role").
		First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find team member model: %w", err)
	}
	return toRecord(model)
}

func (repository *Repository) RoleExists(
	ctx context.Context,
	roleID string,
) (bool, error) {
	var count int64
	if err := repository.database.WithContext(ctx).
		Model(&roleModel{}).
		Where("id = ?", roleID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("count role: %w", err)
	}
	return count > 0, nil
}

func (repository *Repository) Create(
	ctx context.Context,
	member domain.TeamMember,
) error {
	if err := repository.database.WithContext(ctx).
		Create(fromDomain(member)).Error; err != nil {
		return mapConstraintError(err)
	}
	return nil
}

func (repository *Repository) Update(ctx context.Context, member domain.TeamMember) error {
	return repository.update(repository.database.WithContext(ctx), member)
}

func (repository *Repository) UpdateWithSchedule(
	ctx context.Context,
	member domain.TeamMember,
	capacityChanged bool,
	schedule func(context.Context) error,
) error {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	return sharedpersistence.Transaction(ctx, repository.database).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock team member capacity mutation: %w", err)
		}
		var current teamMemberModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ?", member.ID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("lock team member for update: %w", err)
		}
		if err := repository.update(tx, member); err != nil {
			return err
		}
		if capacityChanged && schedule != nil {
			if err := schedule(sharedpersistence.WithTransaction(ctx, tx)); err != nil {
				return fmt.Errorf("recalculate member schedule: %w", err)
			}
		}
		return nil
	})
}

func (repository *Repository) update(database *gorm.DB, member domain.TeamMember) error {
	result := database.Model(&teamMemberModel{}).Where("id = ?", member.ID).Updates(map[string]any{
		"name":              member.Name,
		"role_id":           member.RoleID,
		"daily_capacity":    member.DailyCapacity.Decimal(),
		"buffer_percentage": member.BufferPercentage.Decimal(),
		"updated_at":        member.UpdatedAt,
	})
	if result.Error != nil {
		return mapConstraintError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (repository *Repository) DeleteIfNoActiveTask(ctx context.Context, id string) error {
	return repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var member teamMemberModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&member, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("lock team member for delete: %w", err)
		}
		var activeAssignments int64
		if err := tx.Table("wbs_nodes AS task").
			Joins("JOIN projects AS project ON project.id = task.project_id").
			Where("task.assignee_id = ?", id).
			Where("project.status <> ?", "closed").
			Where("NOT EXISTS (?)", tx.Table("wbs_nodes AS child").Select("1").
				Where("child.project_id = task.project_id AND child.parent_id = task.id")).
			Count(&activeAssignments).Error; err != nil {
			return fmt.Errorf("count active task assignments: %w", err)
		}
		if activeAssignments > 0 {
			return domain.ErrAssignedToTask
		}
		if err := tx.Where("team_member_id = ?", id).
			Delete(&capacityOverrideModel{}).Error; err != nil {
			return fmt.Errorf("soft delete capacity overrides: %w", err)
		}
		result := tx.Delete(&member)
		if result.Error != nil {
			return mapConstraintError(result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}

func fromDomain(member domain.TeamMember) *teamMemberModel {
	return &teamMemberModel{
		ID:               member.ID,
		Name:             member.Name,
		RoleID:           member.RoleID,
		DailyCapacity:    member.DailyCapacity.Decimal(),
		BufferPercentage: member.BufferPercentage.Decimal(),
		CreatedAt:        member.CreatedAt,
		UpdatedAt:        member.UpdatedAt,
	}
}

func toRecord(model teamMemberModel) (*application.TeamMemberRecord, error) {
	dailyValue, err := strconv.ParseFloat(model.DailyCapacity, 64)
	if err != nil {
		return nil, fmt.Errorf("parse team member daily capacity: %w", err)
	}
	dailyCapacity, err := domain.NewDailyCapacity(dailyValue)
	if err != nil {
		return nil, fmt.Errorf("rehydrate team member daily capacity: %w", err)
	}
	bufferValue, err := strconv.ParseFloat(model.BufferPercentage, 64)
	if err != nil {
		return nil, fmt.Errorf("parse team member buffer: %w", err)
	}
	buffer, err := domain.NewBufferPercentage(bufferValue)
	if err != nil {
		return nil, fmt.Errorf("rehydrate team member buffer: %w", err)
	}
	member, err := domain.RehydrateTeamMember(
		model.ID,
		model.Name,
		model.RoleID,
		dailyCapacity,
		buffer,
		model.CreatedAt,
		model.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("rehydrate team member: %w", err)
	}
	if member == nil {
		return nil, errors.New("rehydrate team member: domain returned nil without error")
	}
	return &application.TeamMemberRecord{
		Member:   *member,
		RoleName: model.Role.Name,
	}, nil
}

func mapConstraintError(err error) error {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "team_members_role_fk") ||
		strings.Contains(message, "foreign key") {
		return domain.ErrRoleNotFound
	}
	return err
}
