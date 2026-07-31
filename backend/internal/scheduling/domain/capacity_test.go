package domain

import (
	"errors"
	"math/big"
	"testing"
	"time"
)

func TestCapacityMinutesRoundsEachFinalTimelineCapacityIndependently_AC6_AC7_AC8(t *testing.T) {
	resolved, err := ParseDecimal("8")
	if err != nil {
		t.Fatal(err)
	}

	execution, err := CapacityMinutes(resolved, "30", 20, Execution)
	if err != nil {
		t.Fatal(err)
	}
	commitment, err := CapacityMinutes(resolved, "30", 20, Commitment)
	if err != nil {
		t.Fatal(err)
	}

	if execution.Cmp(big.NewRat(330, 1)) != 0 {
		t.Fatalf("Execution capacity = %s minutes, want 330", execution.RatString())
	}
	if commitment.Cmp(big.NewRat(270, 1)) != 0 {
		t.Fatalf("Commitment capacity = %s minutes, want 270", commitment.RatString())
	}
}

func TestCapacityMinutesRoundsHalfHourTiesUp_AC8(t *testing.T) {
	resolved, err := ParseDecimal("7.5")
	if err != nil {
		t.Fatal(err)
	}

	capacity, err := CapacityMinutes(resolved, "10", 0, Execution)
	if err != nil {
		t.Fatal(err)
	}
	if capacity.Cmp(big.NewRat(420, 1)) != 0 {
		t.Fatalf("capacity = %s minutes, want rounded 420", capacity.RatString())
	}
}

func TestCapacityMinutesRejectsInvalidPercentages_AC6_AC7(t *testing.T) {
	resolved := big.NewRat(8, 1)
	for _, test := range []struct {
		name          string
		memberBuffer  string
		projectBuffer int
	}{
		{name: "negative member buffer", memberBuffer: "-1", projectBuffer: 20},
		{name: "member buffer reaches one hundred", memberBuffer: "100", projectBuffer: 20},
		{name: "project buffer over one hundred", memberBuffer: "20", projectBuffer: 101},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := CapacityMinutes(resolved, test.memberBuffer, test.projectBuffer, Commitment)
			if !errors.Is(err, ErrDataIntegrity) {
				t.Fatalf("CapacityMinutes() error = %v, want ErrDataIntegrity", err)
			}
		})
	}
}

func TestDateHelpersUseUTCDateOnlyAndWeekend_AC5_AC13(t *testing.T) {
	input := time.Date(2026, time.August, 8, 23, 15, 0, 0, time.FixedZone("local", 7*60*60))
	date := DateOnly(input)
	if got := date.Format(time.RFC3339); got != "2026-08-08T00:00:00Z" {
		t.Fatalf("DateOnly() = %s", got)
	}
	if !IsWeekend(date) {
		t.Fatal("Saturday must be treated as weekend")
	}
}
