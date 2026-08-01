package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/application"
	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
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
	return r.CreateWithSchedule(ctx, value, func(context.Context, string) error { return nil })
}

func (r *Repository) CreateWithSchedule(ctx context.Context, value domain.CapacityOverride, schedule func(context.Context, string) error) error {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	return sharedpersistence.Transaction(ctx, r.database).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock capacity override mutation: %w", err)
		}
		if err := r.ensureMember(ctx, tx, value.TeamMemberID, true); err != nil {
			return err
		}
		if err := ensureNoDuplicate(tx, value); err != nil {
			return err
		}
		before, err := resolvedCapacityWindow(tx, value.TeamMemberID, value.StartDate, value.EndDate)
		if err != nil {
			return err
		}
		if err := tx.Create(fromDomain(value)).Error; err != nil {
			if isDuplicateConstraint(err) {
				return domain.ErrDuplicate
			}
			return fmt.Errorf("create capacity override model: %w", err)
		}
		after, err := resolvedCapacityWindow(tx, value.TeamMemberID, value.StartDate, value.EndDate)
		if err != nil {
			return err
		}
		if !sameResolvedCapacity(before, after) {
			if err := schedule(sharedpersistence.WithTransaction(ctx, tx), value.TeamMemberID); err != nil {
				return fmt.Errorf("recalculate affected member schedule: %w", err)
			}
		}
		return nil
	})
}

func (r *Repository) Update(ctx context.Context, value domain.CapacityOverride) error {
	return r.UpdateWithSchedule(ctx, value, func(context.Context, string) error { return nil })
}

func (r *Repository) UpdateWithSchedule(ctx context.Context, value domain.CapacityOverride, schedule func(context.Context, string) error) error {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	return sharedpersistence.Transaction(ctx, r.database).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock capacity override mutation: %w", err)
		}
		if err := r.ensureMember(ctx, tx, value.TeamMemberID, true); err != nil {
			return err
		}
		var current capacityOverrideModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ? AND team_member_id = ?", value.ID, value.TeamMemberID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("find capacity override for update: %w", err)
		}
		if err := ensureNoDuplicate(tx, value); err != nil {
			return err
		}
		from, to := dateUnion(current.StartDate, current.EndDate, value.StartDate, value.EndDate)
		before, err := resolvedCapacityWindow(tx, value.TeamMemberID, from, to)
		if err != nil {
			return err
		}
		result := tx.Model(&capacityOverrideModel{}).Where("id = ? AND team_member_id = ?", value.ID, value.TeamMemberID).Updates(map[string]any{"description": value.Description, "start_date": value.StartDate, "end_date": value.EndDate, "capacity": value.Capacity.Decimal(), "updated_at": value.UpdatedAt})
		if result.Error != nil {
			if isDuplicateConstraint(result.Error) {
				return domain.ErrDuplicate
			}
			return fmt.Errorf("update capacity override model: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		after, err := resolvedCapacityWindow(tx, value.TeamMemberID, from, to)
		if err != nil {
			return err
		}
		if !sameResolvedCapacity(before, after) {
			if err := schedule(sharedpersistence.WithTransaction(ctx, tx), value.TeamMemberID); err != nil {
				return fmt.Errorf("recalculate affected member schedule: %w", err)
			}
		}
		return nil
	})
}

func (r *Repository) Delete(ctx context.Context, memberID, id string) error {
	return r.DeleteWithSchedule(ctx, memberID, id, func(context.Context, string) error { return nil })
}

func (r *Repository) DeleteWithSchedule(ctx context.Context, memberID, id string, schedule func(context.Context, string) error) error {
	ctx, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	return sharedpersistence.Transaction(ctx, r.database).Transaction(func(tx *gorm.DB) error {
		if err := sharedpersistence.LockScheduleMutation(tx); err != nil {
			return fmt.Errorf("lock capacity override mutation: %w", err)
		}
		if err := r.ensureMember(ctx, tx, memberID, true); err != nil {
			return err
		}
		var current capacityOverrideModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, "id = ? AND team_member_id = ?", id, memberID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.ErrNotFound
		} else if err != nil {
			return fmt.Errorf("find capacity override for delete: %w", err)
		}
		before, err := resolvedCapacityWindow(tx, memberID, current.StartDate, current.EndDate)
		if err != nil {
			return err
		}
		result := tx.Unscoped().Where("id = ? AND team_member_id = ?", id, memberID).Delete(&capacityOverrideModel{})
		if result.Error != nil {
			return fmt.Errorf("delete capacity override: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return domain.ErrNotFound
		}
		after, err := resolvedCapacityWindow(tx, memberID, current.StartDate, current.EndDate)
		if err != nil {
			return err
		}
		if !sameResolvedCapacity(before, after) {
			if err := schedule(sharedpersistence.WithTransaction(ctx, tx), memberID); err != nil {
				return fmt.Errorf("recalculate affected member schedule: %w", err)
			}
		}
		return nil
	})
}

func resolvedCapacityWindow(tx *gorm.DB, memberID string, from, to time.Time) (map[string]string, error) {
	var member teamMemberModel
	if err := tx.Select("id", "daily_capacity").First(&member, "id = ?", memberID).Error; err != nil {
		return nil, fmt.Errorf("load member daily capacity for resolution: %w", err)
	}
	var models []capacityOverrideModel
	if err := tx.Where("team_member_id = ? AND deleted_at IS NULL AND start_date <= ? AND end_date >= ?", memberID, to, from).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("load capacity overrides for resolution: %w", err)
	}
	var holidayRows []publicHolidayDateModel
	if err := tx.Where("date BETWEEN ? AND ?", dateOnly(from), dateOnly(to)).Find(&holidayRows).Error; err != nil {
		return nil, fmt.Errorf("load public holidays for capacity resolution: %w", err)
	}
	holidays := make(map[string]struct{}, len(holidayRows))
	for _, holiday := range holidayRows {
		holidays[dateOnly(holiday.Date).Format("2006-01-02")] = struct{}{}
	}
	resolved := make(map[string]string)
	for day := dateOnly(from); !day.After(dateOnly(to)); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		if day.Weekday() == time.Saturday || day.Weekday() == time.Sunday {
			resolved[key] = "0"
			continue
		}
		if _, publicHoliday := holidays[key]; publicHoliday {
			resolved[key] = "0"
			continue
		}
		minimum := ""
		for _, model := range models {
			if day.Before(dateOnly(model.StartDate)) || day.After(dateOnly(model.EndDate)) {
				continue
			}
			if minimum == "" || decimalLess(model.Capacity, minimum) {
				minimum = model.Capacity
			}
		}
		if minimum == "" {
			minimum = member.DailyCapacity
		}
		resolved[key] = minimum
	}
	return resolved, nil
}

func sameResolvedCapacity(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		other, exists := right[key]
		if !exists || decimalLess(value, other) || decimalLess(other, value) {
			return false
		}
	}
	return true
}

func decimalLess(left, right string) bool {
	leftValue, leftOK := new(big.Rat).SetString(left)
	rightValue, rightOK := new(big.Rat).SetString(right)
	if !leftOK || !rightOK {
		return left < right
	}
	return leftValue.Cmp(rightValue) < 0
}

func dateUnion(firstStart, firstEnd, secondStart, secondEnd time.Time) (time.Time, time.Time) {
	from := firstStart
	if secondStart.Before(from) {
		from = secondStart
	}
	to := firstEnd
	if secondEnd.After(to) {
		to = secondEnd
	}
	return from, to
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
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
func ensureNoDuplicate(tx *gorm.DB, value domain.CapacityOverride) error {
	var count int64
	err := tx.Model(&capacityOverrideModel{}).
		Where("team_member_id = ? AND id <> ? AND start_date = ? AND end_date = ? AND capacity = ?", value.TeamMemberID, value.ID, value.StartDate, value.EndDate, value.Capacity.Decimal()).
		Count(&count).Error
	if err != nil {
		return fmt.Errorf("check capacity override duplicate: %w", err)
	}
	if count > 0 {
		return domain.ErrDuplicate
	}
	return nil
}
func isDuplicateConstraint(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "capacity_overrides_active_duplicate_idx") || strings.Contains(message, "unique constraint")
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
