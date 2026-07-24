package gormrepo

import (
	"context"
	"errors"
	"strings"

	"github.com/banggok/sched_mind/backend/internal/roles/domain"
	"gorm.io/gorm"
)

type Repository struct {
	database *gorm.DB
}

func New(database *gorm.DB) *Repository {
	return &Repository{database: database}
}

func (repository *Repository) List(ctx context.Context) ([]domain.Role, error) {
	var models []roleModel
	if err := repository.database.WithContext(ctx).
		Order("LOWER(name) ASC").
		Find(&models).Error; err != nil {
		return nil, err
	}

	roles := make([]domain.Role, 0, len(models))
	for _, model := range models {
		roles = append(roles, toDomain(model))
	}

	return roles, nil
}

func (repository *Repository) FindByID(ctx context.Context, id string) (*domain.Role, error) {
	var model roleModel
	err := repository.database.WithContext(ctx).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	role := toDomain(model)
	return &role, nil
}

func (repository *Repository) NameExists(
	ctx context.Context,
	normalizedName string,
	excludeID string,
) (bool, error) {
	query := repository.database.WithContext(ctx).
		Model(&roleModel{}).
		Where("LOWER(name) = ?", normalizedName)
	if excludeID != "" {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (repository *Repository) Create(ctx context.Context, role domain.Role) error {
	err := repository.database.WithContext(ctx).Create(fromDomain(role)).Error
	return mapConstraintError(err)
}

func (repository *Repository) Update(ctx context.Context, role domain.Role) error {
	result := repository.database.WithContext(ctx).
		Model(&roleModel{}).
		Where("id = ?", role.ID).
		Updates(map[string]any{
			"name":       role.Name,
			"updated_at": role.UpdatedAt,
		})
	if result.Error != nil {
		return mapConstraintError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (repository *Repository) IsInUse(ctx context.Context, id string) (bool, error) {
	var count int64
	if err := repository.database.WithContext(ctx).
		Model(&teamMemberModel{}).
		Where("role_id = ?", id).
		Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (repository *Repository) Delete(ctx context.Context, id string) error {
	result := repository.database.WithContext(ctx).
		Where("id = ?", id).
		Delete(&roleModel{})
	if result.Error != nil {
		return mapConstraintError(result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func toDomain(model roleModel) domain.Role {
	return domain.RehydrateRole(
		model.ID,
		model.Name,
		model.CreatedAt,
		model.UpdatedAt,
	)
}

func fromDomain(role domain.Role) roleModel {
	return roleModel{
		ID:        role.ID,
		Name:      role.Name,
		CreatedAt: role.CreatedAt,
		UpdatedAt: role.UpdatedAt,
	}
}

func mapConstraintError(err error) error {
	if err == nil {
		return nil
	}

	message := strings.ToLower(err.Error())
	if strings.Contains(message, "unique") ||
		strings.Contains(message, "duplicate key") {
		return domain.ErrNameExists
	}
	if strings.Contains(message, "foreign key") {
		return domain.ErrInUse
	}

	return err
}
