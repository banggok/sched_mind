package domain

import (
	"errors"
	"time"
)

var (
	ErrNotFound         = errors.New("dependency not found")
	ErrTaskNotFound     = errors.New("task not found")
	ErrSelfReference    = errors.New("dependency self reference")
	ErrAlreadyExists    = errors.New("dependency already exists")
	ErrExecutableNeeded = errors.New("executable task required")
	ErrCycle            = errors.New("dependency cycle detected")
	ErrClosedProject    = errors.New("closed project task not allowed")
	ErrLockedProject    = errors.New("locked project dependency is read-only")
	ErrCompletedBlocked = errors.New("completed task cannot be blocked")
	ErrCompletedHistory = errors.New("completed dependency history read only")
	ErrInvalidDirection = errors.New("invalid dependency direction")
)

type Direction string

const (
	BlockedBy Direction = "blockedBy"
	Blocks    Direction = "blocks"
)

func (d Direction) Valid() bool { return d == BlockedBy || d == Blocks }

type Dependency struct {
	ID, BlockingTaskID, BlockedTaskID string
	CreatedAt, UpdatedAt              time.Time
}

func New(id, blockingTaskID, blockedTaskID string, now time.Time) (*Dependency, error) {
	return newDependency(id, blockingTaskID, blockedTaskID, now)
}

func Rehydrate(id, blockingTaskID, blockedTaskID string, createdAt, updatedAt time.Time) (*Dependency, error) {
	value, err := newDependency(id, blockingTaskID, blockedTaskID, createdAt)
	if err != nil {
		return nil, err
	}
	value.UpdatedAt = updatedAt
	return value, nil
}

func newDependency(id, blockingTaskID, blockedTaskID string, now time.Time) (*Dependency, error) {
	if id == "" || blockingTaskID == "" || blockedTaskID == "" {
		return nil, ErrTaskNotFound
	}
	if blockingTaskID == blockedTaskID {
		return nil, ErrSelfReference
	}
	return &Dependency{
		ID:             id,
		BlockingTaskID: blockingTaskID,
		BlockedTaskID:  blockedTaskID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

type Task struct {
	ID, Name, ProjectID, ProjectName string
	HierarchyPath                    string
	ActualStart                      *time.Time
	ActualEnd                        *time.Time
	ExpectedStart                    *time.Time
}

type Item struct {
	Dependency Dependency
	Task       Task
}

type Detail struct {
	BlockedBy []Item
	Blocks    []Item
}

type CandidatePage struct {
	Items      []Task
	Page       int
	PageSize   int
	TotalItems int64
}

type CycleStep struct{ TaskID, TaskName, ProjectName string }

type CycleError struct{ Path []CycleStep }

func (e *CycleError) Error() string { return ErrCycle.Error() }
func (e *CycleError) Unwrap() error { return ErrCycle }
