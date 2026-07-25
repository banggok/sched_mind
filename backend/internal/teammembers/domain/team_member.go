package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type TeamMember struct {
	ID               string
	Name             string
	RoleID           string
	DailyCapacity    DailyCapacity
	BufferPercentage BufferPercentage
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewTeamMember(
	id string,
	name string,
	roleID string,
	dailyCapacity DailyCapacity,
	buffer BufferPercentage,
	now time.Time,
) (*TeamMember, error) {
	normalizedName, err := NormalizeName(name)
	if err != nil {
		return nil, err
	}
	normalizedRoleID := strings.TrimSpace(roleID)
	if normalizedRoleID == "" {
		return nil, ErrRoleRequired
	}
	return &TeamMember{
		ID:               id,
		Name:             normalizedName,
		RoleID:           normalizedRoleID,
		DailyCapacity:    dailyCapacity,
		BufferPercentage: buffer,
		CreatedAt:        now,
		UpdatedAt:        now,
	}, nil
}

func RehydrateTeamMember(
	id string,
	name string,
	roleID string,
	dailyCapacity DailyCapacity,
	buffer BufferPercentage,
	createdAt time.Time,
	updatedAt time.Time,
) (*TeamMember, error) {
	member, err := NewTeamMember(
		id,
		name,
		roleID,
		dailyCapacity,
		buffer,
		createdAt,
	)
	if err != nil {
		return nil, err
	}
	member.UpdatedAt = updatedAt
	return member, nil
}

func (member *TeamMember) Update(
	name string,
	roleID string,
	dailyCapacity DailyCapacity,
	buffer BufferPercentage,
	now time.Time,
) error {
	normalizedName, err := NormalizeName(name)
	if err != nil {
		return err
	}
	normalizedRoleID := strings.TrimSpace(roleID)
	if normalizedRoleID == "" {
		return ErrRoleRequired
	}
	member.Name = normalizedName
	member.RoleID = normalizedRoleID
	member.DailyCapacity = dailyCapacity
	member.BufferPercentage = buffer
	member.UpdatedAt = now
	return nil
}

func NormalizeName(name string) (string, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", ErrNameRequired
	}
	if utf8.RuneCountInString(normalized) > 100 {
		return "", ErrNameTooLong
	}
	return normalized, nil
}

func (member TeamMember) CommitmentCapacity() float64 {
	return CommitmentCapacity(member.DailyCapacity, member.BufferPercentage)
}
