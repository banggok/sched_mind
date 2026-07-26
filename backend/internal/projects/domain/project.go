package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

const MaxNameLength = 100

type Status string

const (
	StatusOpen   Status = "open"
	StatusLocked Status = "locked"
	StatusClosed Status = "closed"
)

type PriorityDirection string

const (
	PriorityUp   PriorityDirection = "up"
	PriorityDown PriorityDirection = "down"
)

type Project struct {
	ID                       string
	Name                     string
	Status                   Status
	StartDate                *time.Time
	EndDate                  *time.Time
	AutoCalculateDate        bool
	AutoDependencyByAssignee bool
	Priority                 int
	ClosedAt                 *time.Time
	LockedExecutionSnapshot  *string
	LockedCommitmentSnapshot *string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func NewProject(id, name string, priority int, now time.Time) (*Project, error) {
	normalized, err := NormalizeName(name)
	if err != nil {
		return nil, err
	}
	if priority < 1 {
		return nil, ErrPriorityInvalid
	}
	return &Project{ID: id, Name: normalized, Status: StatusOpen, AutoCalculateDate: true, AutoDependencyByAssignee: true, Priority: priority, CreatedAt: now, UpdatedAt: now}, nil
}

func Rehydrate(id, name string, status Status, startDate, endDate *time.Time, autoCalculateDate, autoDependencyByAssignee bool, priority int, closedAt *time.Time, executionSnapshot, commitmentSnapshot *string, createdAt, updatedAt time.Time) (*Project, error) {
	if _, err := NormalizeName(name); err != nil {
		return nil, err
	}
	if !status.Valid() {
		return nil, ErrStatusInvalid
	}
	if priority < 1 {
		return nil, ErrPriorityInvalid
	}
	return &Project{ID: id, Name: name, Status: status, StartDate: startDate, EndDate: endDate, AutoCalculateDate: autoCalculateDate, AutoDependencyByAssignee: autoDependencyByAssignee, Priority: priority, ClosedAt: closedAt, LockedExecutionSnapshot: executionSnapshot, LockedCommitmentSnapshot: commitmentSnapshot, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func NormalizeName(name string) (string, error) {
	value := strings.TrimSpace(name)
	if value == "" {
		return "", ErrNameRequired
	}
	if utf8.RuneCountInString(value) > MaxNameLength {
		return "", ErrNameTooLong
	}
	return value, nil
}

func NormalizedNameKey(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

func ParseStatus(value string) (Status, error) {
	status := Status(value)
	if !status.Valid() {
		return "", ErrStatusInvalid
	}
	return status, nil
}

func (status Status) Valid() bool {
	return status == StatusOpen || status == StatusLocked || status == StatusClosed
}

func ParsePriorityDirection(value string) (PriorityDirection, error) {
	direction := PriorityDirection(value)
	if direction != PriorityUp && direction != PriorityDown {
		return "", ErrPriorityDirectionInvalid
	}
	return direction, nil
}

func (project *Project) Rename(name string, now time.Time) error {
	if project.Status == StatusClosed {
		return ErrClosedReadOnly
	}
	value, err := NormalizeName(name)
	if err != nil {
		return err
	}
	project.Name, project.UpdatedAt = value, now
	return nil
}

func (project *Project) ChangeStatus(target Status, hasLeaves, hasUnfinishedLeaves bool, executionSnapshot, commitmentSnapshot *string, now time.Time) error {
	if !target.Valid() {
		return ErrStatusInvalid
	}
	allowed := (project.Status == StatusOpen && target == StatusLocked) ||
		((project.Status == StatusOpen || project.Status == StatusLocked) && target == StatusClosed) ||
		(project.Status == StatusClosed && target == StatusOpen)
	if !allowed {
		return ErrStatusTransitionNotAllowed
	}
	if target == StatusLocked && !hasLeaves {
		return ErrCannotLockWithoutTasks
	}
	if target == StatusClosed {
		if !hasLeaves {
			return ErrCannotCloseWithoutTasks
		}
		if hasUnfinishedLeaves {
			return ErrCannotCloseWithActiveTasks
		}
		closedAt := now
		project.ClosedAt = &closedAt
	}
	if target == StatusLocked {
		project.LockedExecutionSnapshot = cloneString(executionSnapshot)
		project.LockedCommitmentSnapshot = cloneString(commitmentSnapshot)
	}
	if target == StatusOpen {
		project.ClosedAt = nil
	}
	project.Status, project.UpdatedAt = target, now
	return nil
}

func (project Project) CanDelete(hasChildren bool) error {
	if hasChildren {
		return ErrHasChildren
	}
	return nil
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
