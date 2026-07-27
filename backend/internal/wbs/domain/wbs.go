package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type Direction string

const (
	MoveUp   Direction = "up"
	MoveDown Direction = "down"
)

type Timeline struct {
	Start *time.Time
	End   *time.Time
}

type ExecutableFields struct {
	RoleID             *string
	AssigneeID         *string
	EffortMinutes      *int
	ExecutionTimeline  Timeline
	CommitmentTimeline Timeline
	ActualEnd          *time.Time
}

type Node struct {
	ID          string
	ProjectID   string
	ParentID    *string
	Name        string
	Position    int
	HasChildren bool
	Executable  ExecutableFields
	Children    []Node
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NormalizeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrNameRequired
	}
	if utf8.RuneCountInString(value) > 100 {
		return "", ErrNameTooLong
	}
	return value, nil
}

func NameKey(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func New(id, projectID string, parentID *string, name string, position int, now time.Time) (*Node, error) {
	name, err := NormalizeName(name)
	if err != nil {
		return nil, err
	}
	return &Node{ID: id, ProjectID: projectID, ParentID: cloneString(parentID), Name: name, Position: position, Children: []Node{}, CreatedAt: now, UpdatedAt: now}, nil
}

func (node Node) IsExecutable() bool { return !node.HasChildren }
func (node Node) HasExecutableData() bool {
	f := node.Executable
	return f.RoleID != nil || f.AssigneeID != nil || f.EffortMinutes != nil || f.ExecutionTimeline.Start != nil || f.ExecutionTimeline.End != nil || f.CommitmentTimeline.Start != nil || f.CommitmentTimeline.End != nil || f.ActualEnd != nil
}

func (node *Node) Rename(name string, now time.Time) error {
	if node.Executable.ActualEnd != nil {
		return ErrCompletedReadOnly
	}
	value, err := NormalizeName(name)
	if err != nil {
		return err
	}
	node.Name, node.UpdatedAt = value, now
	return nil
}

func ValidateExecutable(fields ExecutableFields, automaticScheduling bool, projectOpen bool) error {
	if fields.EffortMinutes != nil && (*fields.EffortMinutes < 30 || *fields.EffortMinutes%30 != 0) {
		return ErrEffortInvalid
	}
	if err := validateTimeline(fields.ExecutionTimeline); err != nil {
		return err
	}
	if err := validateTimeline(fields.CommitmentTimeline); err != nil {
		return err
	}
	hasManual := fields.ExecutionTimeline.Start != nil || fields.CommitmentTimeline.Start != nil
	if hasManual && (automaticScheduling || !projectOpen) {
		return ErrManualTimeline
	}
	return nil
}

func validateTimeline(value Timeline) error {
	if (value.Start == nil) != (value.End == nil) {
		return ErrTimelinePair
	}
	if value.Start != nil && value.End.Before(*value.Start) {
		return ErrTimelineOrder
	}
	return nil
}

func (node *Node) UpdateExecutable(fields ExecutableFields, automaticScheduling bool, projectOpen bool, now time.Time) error {
	if !node.IsExecutable() {
		return ErrExecutableOnly
	}
	if node.Executable.ActualEnd != nil {
		return ErrCompletedReadOnly
	}
	if err := ValidateExecutable(fields, automaticScheduling, projectOpen); err != nil {
		return err
	}
	node.Executable, node.UpdatedAt = cloneFields(fields), now
	return nil
}

func (node *Node) Complete(actualEnd time.Time, now time.Time) error {
	if !node.IsExecutable() {
		return ErrExecutableOnly
	}
	if node.Executable.ActualEnd != nil {
		return ErrCompletedReadOnly
	}
	value := dateOnly(actualEnd)
	node.Executable.ActualEnd, node.UpdatedAt = &value, now
	return nil
}

func EmptyExecutable() ExecutableFields { return ExecutableFields{} }

func cloneFields(value ExecutableFields) ExecutableFields {
	value.RoleID = cloneString(value.RoleID)
	value.AssigneeID = cloneString(value.AssigneeID)
	value.EffortMinutes = cloneInt(value.EffortMinutes)
	value.ExecutionTimeline = cloneTimeline(value.ExecutionTimeline)
	value.CommitmentTimeline = cloneTimeline(value.CommitmentTimeline)
	if value.ActualEnd != nil {
		v := dateOnly(*value.ActualEnd)
		value.ActualEnd = &v
	}
	return value
}
func cloneTimeline(value Timeline) Timeline {
	return Timeline{Start: cloneDate(value.Start), End: cloneDate(value.End)}
}
func cloneDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	v := dateOnly(*value)
	return &v
}
func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}
func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}
func dateOnly(value time.Time) time.Time {
	y, m, d := value.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
