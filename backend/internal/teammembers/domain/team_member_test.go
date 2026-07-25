package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestTeamMemberValidationAndUpdate(t *testing.T) {
	t.Parallel()
	daily, _ := NewDailyCapacity(8)
	now := time.Date(2026, 7, 25, 10, 0, 0, 0, time.UTC)
	member, err := NewTeamMember(
		"member-1",
		"  Harry  ",
		"role-1",
		daily,
		DefaultBufferPercentage(),
		now,
	)
	if err != nil {
		t.Fatalf("NewTeamMember() error = %v", err)
	}
	if member.Name != "Harry" {
		t.Fatalf("Name = %q, want Harry", member.Name)
	}

	updatedAt := now.Add(time.Hour)
	updatedCapacity, _ := NewDailyCapacity(7.5)
	buffer, _ := NewBufferPercentage(10)
	if err := member.Update(
		"Harry Wijaya",
		"role-2",
		updatedCapacity,
		buffer,
		updatedAt,
	); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if member.ID != "member-1" || !member.CreatedAt.Equal(now) {
		t.Fatal("Update() changed immutable identity or creation time")
	}
	if member.RoleID != "role-2" || member.CommitmentCapacity() != 7 {
		t.Fatal("Update() did not apply role or capacity values")
	}
}

func TestTeamMemberRejectsInvalidIdentityFields(t *testing.T) {
	t.Parallel()
	daily, _ := NewDailyCapacity(8)
	tests := []struct {
		name       string
		memberName string
		roleID     string
		wantError  error
	}{
		{name: "blank name", memberName: " ", roleID: "role", wantError: ErrNameRequired},
		{name: "long name", memberName: strings.Repeat("a", 101), roleID: "role", wantError: ErrNameTooLong},
		{name: "blank role", memberName: "Harry", roleID: " ", wantError: ErrRoleRequired},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			member, err := NewTeamMember(
				"id",
				test.memberName,
				test.roleID,
				daily,
				DefaultBufferPercentage(),
				time.Now(),
			)
			if !errors.Is(err, test.wantError) {
				t.Fatalf("NewTeamMember() error = %v, want %v", err, test.wantError)
			}
			if member != nil {
				t.Fatalf("NewTeamMember() result = %#v, want nil", member)
			}
		})
	}
}
