package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"github.com/banggok/sched_mind/backend/internal/publicholidays/application"
	"github.com/banggok/sched_mind/backend/internal/publicholidays/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"gorm.io/gorm"
	"time"
)

type Repository struct{ database *gorm.DB }

func New(database *gorm.DB) *Repository { return &Repository{database: database} }

func (r *Repository) List(ctx context.Context, q application.ListQuery) (listing.Page[domain.PublicHoliday], error) {
	base := r.database.WithContext(ctx).Model(&publicHolidayModel{})
	if q.HolidayDate != nil {
		base = base.Where("id IN (?)", r.database.Model(&publicHolidayDateModel{}).Select("public_holiday_id").Where("date = ?", *q.HolidayDate))
	}
	var activeCount, pastCount int64
	if err := base.Where("end_date >= ?", q.Today).Count(&activeCount).Error; err != nil {
		return listing.Page[domain.PublicHoliday]{}, fmt.Errorf("count current public holidays: %w", err)
	}
	base = r.database.WithContext(ctx).Model(&publicHolidayModel{})
	if q.HolidayDate != nil {
		base = base.Where("id IN (?)", r.database.Model(&publicHolidayDateModel{}).Select("public_holiday_id").Where("date = ?", *q.HolidayDate))
	}
	if err := base.Where("end_date < ?", q.Today).Count(&pastCount).Error; err != nil {
		return listing.Page[domain.PublicHoliday]{}, fmt.Errorf("count past public holidays: %w", err)
	}
	offset := (q.Page - 1) * q.PageSize
	models := make([]publicHolidayModel, 0, q.PageSize)
	if offset < int(activeCount) {
		part, err := r.listSegment(ctx, q, true, offset, q.PageSize)
		if err != nil {
			return listing.Page[domain.PublicHoliday]{}, err
		}
		models = append(models, part...)
		if len(models) < q.PageSize {
			part, err = r.listSegment(ctx, q, false, 0, q.PageSize-len(models))
			if err != nil {
				return listing.Page[domain.PublicHoliday]{}, err
			}
			models = append(models, part...)
		}
	} else {
		part, err := r.listSegment(ctx, q, false, offset-int(activeCount), q.PageSize)
		if err != nil {
			return listing.Page[domain.PublicHoliday]{}, err
		}
		models = append(models, part...)
	}
	items := make([]domain.PublicHoliday, 0, len(models))
	for _, model := range models {
		item, err := toDomain(model)
		if err != nil {
			return listing.Page[domain.PublicHoliday]{}, err
		}
		items = append(items, *item)
	}
	return listing.Page[domain.PublicHoliday]{Items: items, Page: q.Page, PageSize: q.PageSize, Total: activeCount + pastCount}, nil
}
func (r *Repository) listSegment(ctx context.Context, q application.ListQuery, current bool, offset, limit int) ([]publicHolidayModel, error) {
	statement := r.database.WithContext(ctx).Model(&publicHolidayModel{})
	if q.HolidayDate != nil {
		statement = statement.Where("id IN (?)", r.database.Model(&publicHolidayDateModel{}).Select("public_holiday_id").Where("date = ?", *q.HolidayDate))
	}
	if current {
		statement = statement.Where("end_date >= ?", q.Today)
	} else {
		statement = statement.Where("end_date < ?", q.Today)
	}
	var models []publicHolidayModel
	if err := statement.Order("start_date ASC").Order("end_date ASC").Order("id ASC").Offset(offset).Limit(limit).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list public holiday segment: %w", err)
	}
	return models, nil
}
func (r *Repository) Find(ctx context.Context, id string) (*domain.PublicHoliday, error) {
	var model publicHolidayModel
	err := r.database.WithContext(ctx).First(&model, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find public holiday: %w", err)
	}
	return toDomain(model)
}
func (r *Repository) Create(ctx context.Context, value domain.PublicHoliday) error {
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(fromDomain(value)).Error; err != nil {
			return err
		}
		return tx.Create(dateModels(value)).Error
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrDateAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("create public holiday: %w", err)
	}
	return nil
}
func (r *Repository) Update(ctx context.Context, value domain.PublicHoliday) error {
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&publicHolidayModel{}).Where("id = ?", value.ID).Updates(map[string]any{"start_date": value.StartDate, "end_date": value.EndDate, "description": value.Description, "updated_at": value.UpdatedAt})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		if err := tx.Where("public_holiday_id = ?", value.ID).Delete(&publicHolidayDateModel{}).Error; err != nil {
			return err
		}
		return tx.Create(dateModels(value)).Error
	})
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return domain.ErrDateAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("update public holiday: %w", err)
	}
	return nil
}
func (r *Repository) Delete(ctx context.Context, id string) error {
	err := r.database.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("public_holiday_id = ?", id).Delete(&publicHolidayDateModel{}).Error; err != nil {
			return err
		}
		result := tx.Where("id = ?", id).Delete(&publicHolidayModel{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("delete public holiday: %w", err)
	}
	return nil
}
func (r *Repository) ExistsOn(ctx context.Context, date time.Time) (bool, error) {
	var count int64
	if err := r.database.WithContext(ctx).Model(&publicHolidayDateModel{}).Where("date = ?", date).Limit(1).Count(&count).Error; err != nil {
		return false, fmt.Errorf("check public holiday date: %w", err)
	}
	return count > 0, nil
}
func (r *Repository) DatesBetween(ctx context.Context, startDate, endDate time.Time) ([]time.Time, error) {
	var models []publicHolidayDateModel
	if err := r.database.WithContext(ctx).Where("date >= ? AND date <= ?", startDate, endDate).Order("date ASC").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list public holiday dates: %w", err)
	}
	dates := make([]time.Time, 0, len(models))
	for _, model := range models {
		dates = append(dates, model.Date)
	}
	return dates, nil
}
func fromDomain(value domain.PublicHoliday) *publicHolidayModel {
	return &publicHolidayModel{ID: value.ID, StartDate: value.StartDate, EndDate: value.EndDate, Description: value.Description, CreatedAt: value.CreatedAt, UpdatedAt: value.UpdatedAt}
}
func dateModels(value domain.PublicHoliday) []publicHolidayDateModel {
	models := make([]publicHolidayDateModel, 0, len(value.Dates))
	for _, date := range value.Dates {
		models = append(models, publicHolidayDateModel{PublicHolidayID: value.ID, Date: date})
	}
	return models
}
func toDomain(model publicHolidayModel) (*domain.PublicHoliday, error) {
	value, err := domain.Rehydrate(model.ID, model.StartDate, model.EndDate, model.Description, nil, model.CreatedAt, model.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("rehydrate public holiday: %w", err)
	}
	if value == nil {
		return nil, errors.New("rehydrate public holiday: domain returned nil")
	}
	return value, nil
}

var _ application.Repository = (*Repository)(nil)
