package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/application"
	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Repository struct{ database *gorm.DB }

func New(database *gorm.DB) *Repository { return &Repository{database: database} }
func (r *Repository) List(ctx context.Context, memberID string, q application.ListQuery) (listing.Page[domain.CapacityOverride], error) {
	if err := r.ensureMember(ctx, r.database, memberID, false); err != nil {
		return listing.Page[domain.CapacityOverride]{}, err
	}
	statement := r.database.WithContext(ctx).Model(&capacityOverrideModel{}).Where("team_member_id = ?", memberID)
	if q.EffectiveDate != nil {
		statement = statement.Where(
			"start_date <= ? AND end_date >= ?",
			*q.EffectiveDate,
			*q.EffectiveDate,
		)
	}
	var total int64
	if err := statement.Count(&total).Error; err != nil {
		return listing.Page[domain.CapacityOverride]{}, fmt.Errorf("count capacity overrides: %w", err)
	}
	var models []capacityOverrideModel
	if err := statement.Order("start_date ASC").Order("end_date ASC").Order("id ASC").Limit(q.PageSize).Offset((q.Page - 1) * q.PageSize).Find(&models).Error; err != nil {
		return listing.Page[domain.CapacityOverride]{}, fmt.Errorf("list capacity overrides: %w", err)
	}
	items := make([]domain.CapacityOverride, 0, len(models))
	for _, model := range models {
		item, err := toDomain(model)
		if err != nil {
			return listing.Page[domain.CapacityOverride]{}, err
		}
		items = append(items, *item)
	}
	return listing.Page[domain.CapacityOverride]{Items: items, Page: q.Page, PageSize: q.PageSize, Total: total}, nil
}
func (r *Repository) Find(ctx context.Context, memberID, id string) (*domain.CapacityOverride, error) {
	if err := r.ensureMember(ctx, r.database, memberID, false); err != nil {
		return nil, err
	}
	var model capacityOverrideModel
	err := r.database.WithContext(ctx).First(&model, "id = ? AND team_member_id = ?", id, memberID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find capacity override: %w", err)
	}
	return toDomain(model)
}
func (r *Repository) Create(ctx context.Context, value domain.CapacityOverride) error {
	return r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.ensureMember(ctx, tx, value.TeamMemberID, true); err != nil {
			return err
		}
		if err := ensureNoOverlap(tx, value); err != nil {
			return err
		}
		if err := tx.Create(fromDomain(value)).Error; err != nil {
			return fmt.Errorf("create capacity override model: %w", err)
		}
		return nil
	})
}
func (r *Repository) Update(ctx context.Context, value domain.CapacityOverride) error {
	return r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := r.ensureMember(ctx, tx, value.TeamMemberID, true); err != nil {
			return err
		}
		var current capacityOverrideModel
		if err := tx.First(&current, "id = ? AND team_member_id = ?", value.ID, value.TeamMemberID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("find capacity override for update: %w", err)
		}
		if err := ensureNoOverlap(tx, value); err != nil {
			return err
		}
		result := tx.Model(&capacityOverrideModel{}).Where("id = ? AND team_member_id = ?", value.ID, value.TeamMemberID).Updates(map[string]any{"description": value.Description, "start_date": value.StartDate, "end_date": value.EndDate, "capacity": value.Capacity.Decimal(), "updated_at": value.UpdatedAt})
		if result.Error != nil {
			return fmt.Errorf("update capacity override model: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
}
func (r *Repository) Delete(ctx context.Context, memberID, id string) error {
	if err := r.ensureMember(ctx, r.database, memberID, false); err != nil {
		return err
	}
	result := r.database.WithContext(ctx).Unscoped().Where("id = ? AND team_member_id = ?", id, memberID).Delete(&capacityOverrideModel{})
	if result.Error != nil {
		return fmt.Errorf("delete capacity override: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (r *Repository) ensureMember(ctx context.Context, db *gorm.DB, id string, lock bool) error {
	query := db.WithContext(ctx)
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var model teamMemberModel
	err := query.First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.ErrTeamMemberNotFound
	}
	if err != nil {
		return fmt.Errorf("find team member: %w", err)
	}
	return nil
}
func ensureNoOverlap(tx *gorm.DB, value domain.CapacityOverride) error {
	var count int64
	err := tx.Model(&capacityOverrideModel{}).Where("team_member_id = ? AND id <> ? AND start_date <= ? AND end_date >= ?", value.TeamMemberID, value.ID, value.EndDate, value.StartDate).Count(&count).Error
	if err != nil {
		return fmt.Errorf("check capacity override overlap: %w", err)
	}
	if count > 0 {
		return domain.ErrOverlaps
	}
	return nil
}
func fromDomain(value domain.CapacityOverride) *capacityOverrideModel {
	return &capacityOverrideModel{ID: value.ID, TeamMemberID: value.TeamMemberID, Description: value.Description, StartDate: value.StartDate, EndDate: value.EndDate, Capacity: value.Capacity.Decimal(), CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func toDomain(model capacityOverrideModel) (*domain.CapacityOverride, error) {
	hours, err := strconv.ParseFloat(model.Capacity, 64)
	if err != nil {
		return nil, fmt.Errorf("parse capacity override capacity: %w", err)
	}
	capacity, err := domain.NewCapacity(hours)
	if err != nil {
		return nil, fmt.Errorf("rehydrate capacity override capacity: %w", err)
	}
	value, err := domain.Rehydrate(model.ID, model.TeamMemberID, model.Description, model.StartDate, model.EndDate, capacity, model.CreatedAt, model.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("rehydrate capacity override: %w", err)
	}
	if value == nil {
		return nil, errors.New("rehydrate capacity override: domain returned nil")
	}
	return value, nil
}

var _ application.Repository = (*Repository)(nil)
