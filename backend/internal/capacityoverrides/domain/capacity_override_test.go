package domain

import (
	"errors"
	"testing"
	"time"
)

func date(value string) time.Time { result, _ := time.Parse("2006-01-02", value); return result }
func TestCapacityValidation(t *testing.T) {
	tests := []struct {
		value float64
		err   error
	}{{0, nil}, {.5, nil}, {24, nil}, {-.5, ErrCapacityNegative}, {24.5, ErrCapacityExceedsLimit}, {7.2, ErrCapacityInvalidIncrement}}
	for _, test := range tests {
		_, err := NewCapacity(test.value)
		if !errors.Is(err, test.err) {
			t.Errorf("NewCapacity(%v) error=%v want %v", test.value, err, test.err)
		}
	}
}
func TestPeriodRulesAndUpdateOwnership(t *testing.T) {
	capacity, _ := NewCapacity(4)
	now := time.Now()
	value, err := New("id", "member", " Training ", date("2026-07-03"), date("2026-07-03"), capacity, now)
	if err != nil || value == nil {
		t.Fatalf("new: %v", err)
	}
	if value.Description != "Training" {
		t.Fatalf("description=%q", value.Description)
	}
	if err := value.Update("Training", date("2026-07-04"), date("2026-07-03"), capacity, now); !errors.Is(err, ErrInvalidDateRange) {
		t.Fatalf("error=%v", err)
	}
	if value.TeamMemberID != "member" {
		t.Fatal("ownership changed")
	}
}

func TestDescriptionRules(t *testing.T) {
	capacity, _ := NewCapacity(4)
	now := time.Now()
	if _, err := New("id", "member", "  ", now, now, capacity, now); !errors.Is(err, ErrDescriptionRequired) {
		t.Fatalf("blank description error=%v", err)
	}
	if _, err := New("id", "member", string(make([]rune, 101)), now, now, capacity, now); !errors.Is(err, ErrDescriptionTooLong) {
		t.Fatalf("long description error=%v", err)
	}
}
func TestOverlapAndResolution(t *testing.T) {
	if !Overlaps(date("2026-07-03"), date("2026-07-04"), date("2026-07-04"), date("2026-07-05")) {
		t.Fatal("inclusive boundary must overlap")
	}
	if Overlaps(date("2026-07-03"), date("2026-07-04"), date("2026-07-05"), date("2026-07-06")) {
		t.Fatal("adjacent periods must not overlap")
	}
	capacity, _ := NewCapacity(4)
	if ResolveDailyCapacity(true, &capacity, 8) != 0 || ResolveDailyCapacity(false, &capacity, 8) != 4 || ResolveDailyCapacity(false, nil, 8) != 8 {
		t.Fatal("resolution precedence invalid")
	}
}

func TestEffectiveDateUsesInclusiveBoundaries(t *testing.T) {
	start, end := date("2026-07-26"), date("2026-07-28")
	for _, value := range []string{"2026-07-26", "2026-07-27", "2026-07-28"} {
		if !IsEffectiveOn(start, end, date(value)) {
			t.Fatalf("%s should match", value)
		}
	}
	for _, value := range []string{"2026-07-25", "2026-07-29"} {
		if IsEffectiveOn(start, end, date(value)) {
			t.Fatalf("%s should not match", value)
		}
	}
}
