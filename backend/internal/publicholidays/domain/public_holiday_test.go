package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func day(value string) time.Time { result, _ := time.Parse("2006-01-02", value); return result }
func TestRangeSkipsWeekendsAndUpdatePreservesIdentity(t *testing.T) {
	now := time.Now()
	holiday, err := New("id", day("2026-08-07"), day("2026-08-11"), " Retreat ", now)
	if err != nil || holiday == nil {
		t.Fatal(err)
	}
	if len(holiday.Dates) != 3 || holiday.Description != "Retreat" {
		t.Fatalf("holiday=%+v", holiday)
	}
	created := holiday.CreatedAt
	updated := now.Add(time.Hour)
	if err := holiday.Update(day("2026-08-12"), day("2026-08-13"), "Training", updated); err != nil {
		t.Fatal(err)
	}
	if holiday.ID != "id" || !holiday.CreatedAt.Equal(created) || !holiday.UpdatedAt.Equal(updated) {
		t.Fatalf("holiday=%+v", holiday)
	}
}
func TestRangeAndDescriptionValidation(t *testing.T) {
	now := time.Now()
	tests := []struct {
		start, end  time.Time
		description string
		want        error
	}{{time.Time{}, now, "Holiday", ErrStartDateRequired}, {now, time.Time{}, "Holiday", ErrEndDateRequired}, {day("2026-08-18"), day("2026-08-17"), "Holiday", ErrInvalidDateRange}, {day("2026-08-08"), day("2026-08-09"), "Holiday", ErrNoWorkingDates}, {now, now, " ", ErrDescriptionRequired}, {now, now, strings.Repeat("a", 101), ErrDescriptionTooLong}}
	for _, test := range tests {
		if _, err := New("id", test.start, test.end, test.description, now); !errors.Is(err, test.want) {
			t.Fatalf("err=%v want=%v", err, test.want)
		}
	}
}
func TestWeekendIsDefaultHoliday(t *testing.T) {
	if !IsWeekend(day("2026-08-08")) || IsWeekend(day("2026-08-10")) {
		t.Fatal("weekend classification invalid")
	}
}
