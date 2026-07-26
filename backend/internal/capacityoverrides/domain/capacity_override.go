package domain

import (
	"math"
	"strconv"
	"time"
)

type Capacity struct{ halfHours uint8 }

func NewCapacity(hours float64) (Capacity, error) {
	if math.IsNaN(hours) || math.IsInf(hours, 0) || hours < 0 {
		return Capacity{}, ErrCapacityNegative
	}
	if hours > 24 {
		return Capacity{}, ErrCapacityExceedsLimit
	}
	halfHours := hours * 2
	if math.Abs(halfHours-math.Round(halfHours)) > 1e-9 {
		return Capacity{}, ErrCapacityInvalidIncrement
	}
	return Capacity{halfHours: uint8(math.Round(halfHours))}, nil
}

func (capacity Capacity) Hours() float64 { return float64(capacity.halfHours) / 2 }
func (capacity Capacity) Decimal() string {
	return strconv.FormatFloat(capacity.Hours(), 'f', 1, 64)
}

type CapacityOverride struct {
	ID, TeamMemberID     string
	StartDate, EndDate   time.Time
	Capacity             Capacity
	CreatedAt, UpdatedAt time.Time
}

func New(id, teamMemberID string, startDate, endDate time.Time, capacity Capacity, now time.Time) (*CapacityOverride, error) {
	if startDate.IsZero() {
		return nil, ErrStartDateRequired
	}
	if endDate.IsZero() {
		return nil, ErrEndDateRequired
	}
	if endDate.Before(startDate) {
		return nil, ErrInvalidDateRange
	}
	return &CapacityOverride{ID: id, TeamMemberID: teamMemberID, StartDate: startDate, EndDate: endDate, Capacity: capacity, CreatedAt: now, UpdatedAt: now}, nil
}

func Rehydrate(id, teamMemberID string, startDate, endDate time.Time, capacity Capacity, createdAt, updatedAt time.Time) (*CapacityOverride, error) {
	override, err := New(id, teamMemberID, startDate, endDate, capacity, createdAt)
	if err != nil {
		return nil, err
	}
	override.UpdatedAt = updatedAt
	return override, nil
}

func (override *CapacityOverride) Update(startDate, endDate time.Time, capacity Capacity, now time.Time) error {
	if startDate.IsZero() {
		return ErrStartDateRequired
	}
	if endDate.IsZero() {
		return ErrEndDateRequired
	}
	if endDate.Before(startDate) {
		return ErrInvalidDateRange
	}
	override.StartDate, override.EndDate, override.Capacity, override.UpdatedAt = startDate, endDate, capacity, now
	return nil
}

func Overlaps(leftStart, leftEnd, rightStart, rightEnd time.Time) bool {
	return !leftStart.After(rightEnd) && !leftEnd.Before(rightStart)
}

func IsEffectiveOn(startDate, endDate, effectiveDate time.Time) bool {
	return !startDate.After(effectiveDate) && !endDate.Before(effectiveDate)
}

func ResolveDailyCapacity(publicHoliday bool, override *Capacity, baseHours float64) float64 {
	if publicHoliday {
		return 0
	}
	if override != nil {
		return override.Hours()
	}
	return baseHours
}
