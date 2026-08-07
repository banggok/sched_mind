package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

type Direction string

type Placement string

const (
	MoveUp   Direction = "up"
	MoveDown Direction = "down"

	PlaceBefore Placement = "before"
	PlaceAfter  Placement = "after"
)

type Timeline struct {
	Start *time.Time
	End   *time.Time
}

type ExecutableFields struct {
	RoleID                       *string
	AssigneeID                   *string
	EffortMinutes                *int
	LagDays                      int
	CapacityAllocationPercentage int
	ExecutionTimeline            Timeline
	CommitmentTimeline           Timeline
	ExecutionUnscheduledReason   *string
	CommitmentUnscheduledReason  *string
	ActualStart                  *time.Time
	ActualEnd                    *time.Time
}

type GroupScheduling struct {
	Version                      int64
	Source                       string
	AutomaticScheduling          *bool
	SchedulingStartDate          *time.Time
	LocalStatus                  string
	EffectiveAutomatic           bool
	EffectiveStartDate           *time.Time
	InheritedAutomatic           bool
	InheritedStartDate           *time.Time
	InheritedAutomaticSourceID   string
	InheritedAutomaticSourceName string
	InheritedStartDateSourceID   string
	InheritedStartDateSourceName string
	AutomaticSourceID            string
	AutomaticSourceName          string
	StartDateSourceID            string
	StartDateSourceName          string
	EffectiveLifecycle           string
	LockOwnerID                  string
	LockOwnerName                string
}

type Node struct {
	ID          string
	ProjectID   string
	ParentID    *string
	Name        string
	Position    int
	HasChildren bool
	Executable  ExecutableFields
	Scheduling  GroupScheduling
	Children    []Node
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func NormalizeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ErrNameRequired
	}
	if utf8.RuneCountInString(value) > 200 {
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
	return &Node{ID: id, ProjectID: projectID, ParentID: cloneString(parentID), Name: name, Position: position, Executable: EmptyExecutable(), Scheduling: GroupScheduling{Source: "inherit", LocalStatus: "open", EffectiveLifecycle: "open"}, Children: []Node{}, CreatedAt: now, UpdatedAt: now}, nil
}

func (node Node) IsExecutable() bool { return !node.HasChildren }
func (node Node) HasExecutableData() bool {
	f := node.Executable
	return f.RoleID != nil || f.AssigneeID != nil || f.EffortMinutes != nil || f.LagDays != 0 || f.CapacityAllocationPercentage != 100 || f.ExecutionTimeline.Start != nil || f.ExecutionTimeline.End != nil || f.CommitmentTimeline.Start != nil || f.CommitmentTimeline.End != nil || f.ExecutionUnscheduledReason != nil || f.CommitmentUnscheduledReason != nil || f.ActualStart != nil || f.ActualEnd != nil
}

func (node Node) Completed() bool {
	return node.Executable.ActualStart != nil && node.Executable.ActualEnd != nil
}

func (node *Node) Rename(name string, now time.Time) error {
	if node.Completed() {
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
	if fields.CapacityAllocationPercentage < 1 || fields.CapacityAllocationPercentage > 100 {
		return ErrCapacityAllocationInvalid
	}
	if fields.LagDays < 0 {
		return ErrLagInvalid
	}
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
	if node.Completed() {
		return ErrCompletedReadOnly
	}
	if err := ValidateExecutable(fields, automaticScheduling, projectOpen); err != nil {
		return err
	}
	if automaticScheduling {
		fields.ExecutionTimeline = node.Executable.ExecutionTimeline
		fields.CommitmentTimeline = node.Executable.CommitmentTimeline
		fields.ExecutionUnscheduledReason = node.Executable.ExecutionUnscheduledReason
		fields.CommitmentUnscheduledReason = node.Executable.CommitmentUnscheduledReason
	}
	node.Executable, node.UpdatedAt = cloneFields(fields), now
	return nil
}

func (node *Node) Complete(actualStart, actualEnd time.Time, now time.Time) error {
	if !node.IsExecutable() {
		return ErrExecutableOnly
	}
	if node.Completed() {
		return ErrCompletedReadOnly
	}
	start := dateOnly(actualStart)
	end := dateOnly(actualEnd)
	if end.Before(start) {
		return ErrActualDateOrder
	}
	node.Executable.ActualStart = &start
	node.Executable.ActualEnd = &end
	node.UpdatedAt = now
	return nil
}

func (node *Node) Reopen(now time.Time) error {
	if !node.IsExecutable() {
		return ErrExecutableOnly
	}
	if !node.Completed() {
		return ErrTaskNotCompleted
	}
	node.Executable.ActualStart = nil
	node.Executable.ActualEnd = nil
	node.UpdatedAt = now
	return nil
}

func EmptyExecutable() ExecutableFields { return ExecutableFields{CapacityAllocationPercentage: 100} }

func cloneFields(value ExecutableFields) ExecutableFields {
	value.RoleID = cloneString(value.RoleID)
	value.AssigneeID = cloneString(value.AssigneeID)
	value.EffortMinutes = cloneInt(value.EffortMinutes)
	value.ExecutionTimeline = cloneTimeline(value.ExecutionTimeline)
	value.CommitmentTimeline = cloneTimeline(value.CommitmentTimeline)
	value.ExecutionUnscheduledReason = cloneString(value.ExecutionUnscheduledReason)
	value.CommitmentUnscheduledReason = cloneString(value.CommitmentUnscheduledReason)
	if value.ActualStart != nil {
		v := dateOnly(*value.ActualStart)
		value.ActualStart = &v
	}
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
