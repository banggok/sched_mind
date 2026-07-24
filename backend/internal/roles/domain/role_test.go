package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestNormalizeName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		input     string
		want      string
		wantError error
	}{
		{name: "trims whitespace", input: "  Backend Engineer  ", want: "Backend Engineer"},
		{name: "allows supported punctuation", input: "QA / Automation (L2)", want: "QA / Automation (L2)"},
		{name: "requires a value", input: "   ", wantError: ErrNameRequired},
		{name: "rejects unsupported punctuation", input: "Backend!", wantError: ErrNameInvalid},
		{name: "accepts exactly one hundred characters", input: strings.Repeat("a", 100), want: strings.Repeat("a", 100)},
		{name: "rejects more than one hundred characters", input: strings.Repeat("a", 101), wantError: ErrNameTooLong},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeName(test.input)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("NormalizeName() error = %v, want %v", err, test.wantError)
			}
			if got != test.want {
				t.Fatalf("NormalizeName() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestRenamePreservesIdentityAndCreationTime(t *testing.T) {
	t.Parallel()

	createdAt := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)
	role, err := NewRole("role-id", "Backend", createdAt)
	if err != nil {
		t.Fatalf("NewRole() error = %v", err)
	}
	if role == nil {
		t.Fatal("NewRole() result = nil, want role")
	}

	if err := role.Rename(" Backend Engineer ", updatedAt); err != nil {
		t.Fatalf("Rename() error = %v", err)
	}

	if role.ID != "role-id" {
		t.Fatalf("ID = %q, want role-id", role.ID)
	}
	if role.Name != "Backend Engineer" {
		t.Fatalf("Name = %q, want Backend Engineer", role.Name)
	}
	if !role.CreatedAt.Equal(createdAt) {
		t.Fatalf("CreatedAt = %v, want %v", role.CreatedAt, createdAt)
	}
	if !role.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("UpdatedAt = %v, want %v", role.UpdatedAt, updatedAt)
	}
}

func TestNewRoleReturnsNilOnValidationError(t *testing.T) {
	t.Parallel()

	role, err := NewRole("role-id", " ", time.Now())
	if !errors.Is(err, ErrNameRequired) {
		t.Fatalf("NewRole() error = %v, want ErrNameRequired", err)
	}
	if role != nil {
		t.Fatalf("NewRole() result = %#v, want nil on error", role)
	}
}
