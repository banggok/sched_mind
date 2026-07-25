package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/teammembers/application"
	"github.com/banggok/sched_mind/backend/internal/teammembers/domain"
	"gorm.io/gorm"
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
		statement = statement.Where("LOWER(name) LIKE ?", escapeLike(search)+"%")
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

func escapeLike(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `%`, `\%`)
	return strings.ReplaceAll(value, `_`, `\_`)
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

func (repository *Repository) Update(
	ctx context.Context,
	member domain.TeamMember,
) error {
	result := repository.database.WithContext(ctx).
		Model(&teamMemberModel{}).
		Where("id = ?", member.ID).
		Updates(map[string]any{
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

func (repository *Repository) IsAssignedToTask(
	ctx context.Context,
	id string,
) (bool, error) {
	var count int64
	if err := repository.database.WithContext(ctx).
		Model(&executableLeafModel{}).
		Where("assignee_id = ?", id).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("count executable leaf assignments: %w", err)
	}
	return count > 0, nil
}

func (repository *Repository) HasCapacityOverride(
	ctx context.Context,
	id string,
) (bool, error) {
	var count int64
	if err := repository.database.WithContext(ctx).
		Model(&capacityOverrideModel{}).
		Where("team_member_id = ?", id).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("count capacity overrides: %w", err)
	}
	return count > 0, nil
}

func (repository *Repository) Delete(ctx context.Context, id string) error {
	result := repository.database.WithContext(ctx).
		Delete(&teamMemberModel{}, "id = ?", id)
	if result.Error != nil {
		return mapConstraintError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
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
