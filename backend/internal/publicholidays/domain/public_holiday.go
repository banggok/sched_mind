package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type PublicHoliday struct {
	ID          string
	StartDate   time.Time
	EndDate     time.Time
	Description string
	Dates       []time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func New(id string, startDate, endDate time.Time, description string, now time.Time) (*PublicHoliday, error) {
	description, err := normalizeDescription(description)
	if err != nil {
		return nil, err
	}
	dates, err := workingDates(startDate, endDate)
	if err != nil {
		return nil, err
	}
	return &PublicHoliday{ID: id, StartDate: startDate, EndDate: endDate, Description: description, Dates: dates, CreatedAt: now, UpdatedAt: now}, nil
}

func Rehydrate(id string, startDate, endDate time.Time, description string, dates []time.Time, createdAt, updatedAt time.Time) (*PublicHoliday, error) {
	holiday, err := New(id, startDate, endDate, description, createdAt)
	if err != nil {
		return nil, err
	}
	if len(dates) > 0 {
		holiday.Dates = append([]time.Time(nil), dates...)
	}
	holiday.UpdatedAt = updatedAt
	return holiday, nil
}

func (holiday *PublicHoliday) Update(startDate, endDate time.Time, description string, now time.Time) error {
	description, err := normalizeDescription(description)
	if err != nil {
		return err
	}
	dates, err := workingDates(startDate, endDate)
	if err != nil {
		return err
	}
	holiday.StartDate = startDate
	holiday.EndDate = endDate
	holiday.Description = description
	holiday.Dates = dates
	holiday.UpdatedAt = now
	return nil
}

func workingDates(startDate, endDate time.Time) ([]time.Time, error) {
	if startDate.IsZero() {
		return nil, ErrStartDateRequired
	}
	if endDate.IsZero() {
		return nil, ErrEndDateRequired
	}
	if endDate.Before(startDate) {
		return nil, ErrInvalidDateRange
	}
	dates := make([]time.Time, 0)
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		if date.Weekday() != time.Saturday && date.Weekday() != time.Sunday {
			dates = append(dates, date)
		}
	}
	if len(dates) == 0 {
		return nil, ErrNoWorkingDates
	}
	return dates, nil
}

func normalizeDescription(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrDescriptionRequired
	}
	if utf8.RuneCountInString(value) > 100 {
		return "", ErrDescriptionTooLong
	}
	return value, nil
}

func IsWeekend(date time.Time) bool {
	return date.Weekday() == time.Saturday || date.Weekday() == time.Sunday
}
