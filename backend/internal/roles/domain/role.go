package domain

import (
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const MaxNameLength = 100

type Role struct {
	ID        string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewRole(id, name string, now time.Time) (*Role, error) {
	normalizedName, err := NormalizeName(name)
	if err != nil {
		return nil, err
	}

	return &Role{
		ID:        id,
		Name:      normalizedName,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func RehydrateRole(id, name string, createdAt, updatedAt time.Time) Role {
	return Role{
		ID:        id,
		Name:      name,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

func (role *Role) Rename(name string, now time.Time) error {
	normalizedName, err := NormalizeName(name)
	if err != nil {
		return err
	}

	role.Name = normalizedName
	role.UpdatedAt = now

	return nil
}

func NormalizeName(name string) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", ErrNameRequired
	}

	if utf8.RuneCountInString(trimmed) > MaxNameLength {
		return "", ErrNameTooLong
	}

	for _, character := range trimmed {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			continue
		}

		switch character {
		case ' ', '-', '/', '(', ')':
			continue
		default:
			return "", ErrNameInvalid
		}
	}

	return trimmed, nil
}

func NormalizedNameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
