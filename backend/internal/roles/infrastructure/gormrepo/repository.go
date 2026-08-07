package gormrepo

import (
	"context"
	"errors"
	"strings"

	"github.com/banggok/sched_mind/backend/internal/roles/application"
	"github.com/banggok/sched_mind/backend/internal/roles/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct {
	database *gorm.DB
}

func New(database *gorm.DB) *Repository {
	return &Repository{database: database}
}

func (repository *Repository) List(ctx context.Context, query listing.Query) (listing.Page[domain.Role], error) {
	var models []roleModel
	statement := repository.database.WithContext(ctx).Model(&roleModel{})
	if search := strings.ToLower(strings.TrimSpace(query.Search)); search != "" {
		statement = statement.Where("LOWER(name) LIKE ?", sharedpersistence.EscapeLike(search)+"%")
	}
	var total int64
	if err := statement.Count(&total).Error; err != nil {
		return listing.Page[domain.Role]{}, err
	}
	if err := statement.Order("LOWER(name) ASC").Order("id ASC").
		Limit(query.PageSize).Offset((query.Page - 1) * query.PageSize).
		Find(&models).Error; err != nil {
		return listing.Page[domain.Role]{}, err
	}

	roles := make([]domain.Role, 0, len(models))
	for _, model := range models {
		roles = append(roles, toDomain(model))
	}

	return listing.Page[domain.Role]{
		Items: roles, Page: query.Page, PageSize: query.PageSize, Total: total,
	}, nil
}

func (repository *Repository) ListMembers(
	ctx context.Context,
	roleID string,
	query listing.Query,
) (listing.Page[application.MemberUsage], error) {
	statement := repository.database.WithContext(ctx).Model(&teamMemberModel{}).Where("role_id = ?", roleID)
	if search := strings.ToLower(strings.TrimSpace(query.Search)); search != "" {
		statement = statement.Where("LOWER(name) LIKE ?", sharedpersistence.EscapeLike(search)+"%")
	}

	var total int64
	if err := statement.Count(&total).Error; err != nil {
		return listing.Page[application.MemberUsage]{}, err
	}

	var members []teamMemberModel
	if err := statement.Select("id, name").
		Order("LOWER(name) ASC").Order("id ASC").
		Limit(query.PageSize).Offset((query.Page - 1) * query.PageSize).
		Find(&members).Error; err != nil {
		return listing.Page[application.MemberUsage]{}, err
	}

	items := make([]application.MemberUsage, 0, len(members))
	for _, member := range members {
		items = append(items, application.MemberUsage{ID: member.ID, Name: member.Name})
	}
	return listing.Page[application.MemberUsage]{
		Items: items, Page: query.Page, PageSize: query.PageSize, Total: total,
	}, nil
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

func (repository *Repository) Delete(ctx context.Context, id string) error {
	return repository.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var role roleModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&role, "id = ?", id).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		} else if err != nil {
			return err
		}

		var activeMembers int64
		if err := tx.Model(&teamMemberModel{}).Where("role_id = ?", id).Count(&activeMembers).Error; err != nil {
			return err
		}
		if activeMembers > 0 {
			return domain.ErrInUse
		}

		var taskReferences int64
		if err := tx.Model(&wbsNodeModel{}).Where("role_id = ?", id).Count(&taskReferences).Error; err != nil {
			return err
		}
		if taskReferences > 0 {
			return domain.ErrInUseByTask
		}

		if err := tx.Unscoped().Model(&teamMemberModel{}).
			Where("role_id = ? AND deleted_at IS NOT NULL", id).
			Update("role_id", nil).Error; err != nil {
			return err
		}

		result := tx.Delete(&role)
		if result.Error != nil {
			return mapConstraintError(result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
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
