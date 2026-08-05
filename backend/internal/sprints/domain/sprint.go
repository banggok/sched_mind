package domain

import (
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

type Status string

const (
	StatusPlanned Status = "planned"
	StatusStarted Status = "started"
)

type Sprint struct {
	ID             string
	Name           string
	NormalizedName string
	StartDate      time.Time
	EndDate        time.Time
	Status         Status
	Version        int64
	MemberIDs      []string
	TaskIDs        []string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	StartedAt      *time.Time
}

func NewSprint(id, name string, startDate, endDate time.Time, memberIDs, taskIDs []string, now time.Time) (*Sprint, error) {
	sprint := &Sprint{ID: id, Status: StatusPlanned, Version: 1, CreatedAt: now, UpdatedAt: now}
	if err := sprint.replace(name, startDate, endDate, memberIDs, taskIDs); err != nil {
		return nil, err
	}
	return sprint, nil
}

func (sprint *Sprint) Update(name string, startDate, endDate time.Time, memberIDs, taskIDs []string, now time.Time) error {
	if err := sprint.replace(name, startDate, endDate, memberIDs, taskIDs); err != nil {
		return err
	}
	sprint.Version++
	sprint.UpdatedAt = now
	return nil
}

func (sprint *Sprint) Start(now time.Time) error {
	if sprint.Status == StatusStarted {
		return ErrAlreadyStarted
	}
	sprint.Status = StatusStarted
	sprint.StartedAt = &now
	sprint.UpdatedAt = now
	sprint.Version++
	return nil
}

func (sprint *Sprint) replace(name string, startDate, endDate time.Time, memberIDs, taskIDs []string) error {
	trimmedName := strings.TrimSpace(name)
	if trimmedName == "" {
		return ErrNameRequired
	}
	if utf8.RuneCountInString(trimmedName) > 200 {
		return ErrNameTooLong
	}
	startDate = dateOnly(startDate)
	endDate = dateOnly(endDate)
	if endDate.Before(startDate) {
		return ErrDateRangeInvalid
	}
	members := uniqueIDs(memberIDs)
	if len(members) == 0 {
		return ErrMemberRequired
	}
	sprint.Name = trimmedName
	sprint.NormalizedName = strings.ToLower(trimmedName)
	sprint.StartDate = startDate
	sprint.EndDate = endDate
	sprint.MemberIDs = members
	sprint.TaskIDs = uniqueIDs(taskIDs)
	return nil
}

func Overlaps(startDate, endDate time.Time, memberIDs []string, other Sprint) bool {
	if dateOnly(other.StartDate).After(dateOnly(endDate)) || dateOnly(startDate).After(dateOnly(other.EndDate)) {
		return false
	}
	for _, memberID := range uniqueIDs(memberIDs) {
		if slices.Contains(other.MemberIDs, memberID) {
			return true
		}
	}
	return false
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func uniqueIDs(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	unique := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		unique = append(unique, id)
	}
	return unique
}
