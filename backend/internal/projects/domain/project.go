package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

const MaxNameLength = 100
const DefaultProjectBuffer = 20

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
	AutomaticScheduling      bool
	SchedulingStartDate      *time.Time
	ProjectBuffer            int
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
	return &Project{ID: id, Name: normalized, Status: StatusOpen, AutoCalculateDate: true, AutoDependencyByAssignee: true, AutomaticScheduling: true, ProjectBuffer: DefaultProjectBuffer, Priority: priority, CreatedAt: now, UpdatedAt: now}, nil
}

func Rehydrate(id, name string, status Status, startDate, endDate *time.Time, autoCalculateDate, autoDependencyByAssignee, automaticScheduling bool, schedulingStartDate *time.Time, projectBuffer, priority int, closedAt *time.Time, executionSnapshot, commitmentSnapshot *string, createdAt, updatedAt time.Time) (*Project, error) {
	if _, err := NormalizeName(name); err != nil {
		return nil, err
	}
	if !status.Valid() {
		return nil, ErrStatusInvalid
	}
	if priority < 1 {
		return nil, ErrPriorityInvalid
	}
	if err := ValidateProjectBuffer(projectBuffer); err != nil {
		return nil, err
	}
	return &Project{ID: id, Name: name, Status: status, StartDate: startDate, EndDate: endDate, AutoCalculateDate: autoCalculateDate, AutoDependencyByAssignee: autoDependencyByAssignee, AutomaticScheduling: automaticScheduling, SchedulingStartDate: cloneDate(schedulingStartDate), ProjectBuffer: projectBuffer, Priority: priority, ClosedAt: closedAt, LockedExecutionSnapshot: executionSnapshot, LockedCommitmentSnapshot: commitmentSnapshot, CreatedAt: createdAt, UpdatedAt: updatedAt}, nil
}

func ValidateProjectBuffer(value int) error {
	if value < 0 || value > 100 {
		return ErrProjectBufferInvalid
	}
	return nil
}

func (project *Project) UpdateSettings(automaticScheduling bool, schedulingStartDate *time.Time, projectBuffer int, now time.Time) error {
	if project.Status != StatusOpen {
		return ErrSettingsReadOnly
	}
	if err := ValidateProjectBuffer(projectBuffer); err != nil {
		return err
	}
	project.AutomaticScheduling = automaticScheduling
	project.SchedulingStartDate = cloneDate(schedulingStartDate)
	project.ProjectBuffer = projectBuffer
	project.UpdatedAt = now
	return nil
}

func cloneDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	date := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	return &date
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
