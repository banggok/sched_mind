package domain

import (
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxFilterNameLength = 100
	SystemFilterName    = "All Active Projects"
)

var (
	ErrFilterNameRequired = errors.New("saved filter name is required")
	ErrFilterNameTooLong  = errors.New("saved filter name must not exceed 100 characters")
	ErrFilterNameReserved = errors.New("saved filter name is reserved")
	ErrFilterNameExists   = errors.New("saved filter name already exists")
	ErrFilterNotFound     = errors.New("saved filter not found")
	ErrFilterConflict     = errors.New("saved filter changed concurrently")
	ErrProjectionInvalid  = errors.New("portfolio projection is invalid")
	ErrProjectLimit       = errors.New("too many projects selected")
	ErrDateRangeInvalid   = errors.New("portfolio date range is invalid")
)

type Projection string

const (
	Execution  Projection = "execution"
	Commitment Projection = "commitment"
)

func ParseProjection(value string) (Projection, error) {
	projection := Projection(strings.ToLower(strings.TrimSpace(value)))
	if projection != Execution && projection != Commitment {
		return "", ErrProjectionInvalid
	}
	return projection, nil
}

func NormalizeFilterName(value string) (string, string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", "", ErrFilterNameRequired
	}
	if utf8.RuneCountInString(name) > MaxFilterNameLength {
		return "", "", ErrFilterNameTooLong
	}
	key := strings.ToLower(name)
	if key == strings.ToLower(SystemFilterName) {
		return "", "", ErrFilterNameReserved
	}
	return name, key, nil
}

type ProjectOption struct {
	ID              string
	Name            string
	Status          string
	Priority        int
	ScheduleVersion int64
}

type Row struct {
	ID                 string
	ProjectID          string
	ParentID           *string
	Kind               string
	Name               string
	WBSNumber          string
	Depth              int
	Position           int
	Status             string
	RoleID             *string
	RoleName           *string
	AssigneeID         *string
	AssigneeName       *string
	EffortMinutes      *int
	Start              *time.Time
	End                *time.Time
	UnscheduledReason  *string
	IncompleteEffort   bool
	IncompleteSchedule bool
	HasChildren        bool
	Completed          bool
}

type Dependency struct {
	ID             string
	BlockingTaskID string
	BlockedTaskID  string
	Source         string
}

type Holiday struct {
	Date        time.Time
	Description string
}

type Portfolio struct {
	Projection       Projection
	Projects         []ProjectOption
	Rows             []Row
	Dependencies     []Dependency
	Holidays         []Holiday
	WorkingDayAnchor *time.Time
}

type SavedFilter struct {
	ID         string
	Name       string
	ProjectIDs []string
	Version    int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
