package domain

import (
	"errors"
	"testing"
)

func TestDailyCapacityValidation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		value     float64
		wantError error
	}{
		{name: "zero", value: 0, wantError: ErrDailyCapacityNotPositive},
		{name: "negative", value: -1, wantError: ErrDailyCapacityNotPositive},
		{name: "over limit", value: 24.5, wantError: ErrDailyCapacityExceedsLimit},
		{name: "invalid increment", value: 7.2, wantError: ErrDailyCapacityInvalidIncrement},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewDailyCapacity(test.value)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("NewDailyCapacity() error = %v, want %v", err, test.wantError)
			}
		})
	}
	for _, value := range []float64{0.5, 6, 7.5, 8, 24} {
		if _, err := NewDailyCapacity(value); err != nil {
			t.Fatalf("NewDailyCapacity(%v) error = %v", value, err)
		}
	}
}

func TestBufferAndBaseExecutionCapacity(t *testing.T) {
	t.Parallel()
	daily, err := NewDailyCapacity(8)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		buffer float64
		want   float64
	}{
		{buffer: 0, want: 8},
		{buffer: 20, want: 6.5},
		{buffer: 25, want: 6},
	}
	for _, test := range tests {
		buffer, err := NewBufferPercentage(test.buffer)
		if err != nil {
			t.Fatalf("NewBufferPercentage(%v) error = %v", test.buffer, err)
		}
		if got := BaseExecutionCapacity(daily, buffer); got != test.want {
			t.Fatalf("BaseExecutionCapacity() = %v, want %v", got, test.want)
		}
	}

	sevenAndHalf, err := NewDailyCapacity(7.5)
	if err != nil {
		t.Fatal(err)
	}
	if got := BaseExecutionCapacity(sevenAndHalf, DefaultBufferPercentage()); got != 6 {
		t.Fatalf("BaseExecutionCapacity() = %v, want 6", got)
	}
	tenPercent, err := NewBufferPercentage(10)
	if err != nil {
		t.Fatal(err)
	}
	if got := BaseExecutionCapacity(sevenAndHalf, tenPercent); got != 7 {
		t.Fatalf("BaseExecutionCapacity() rounded = %v, want 7", got)
	}
}
