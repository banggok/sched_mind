package application

import (
	"errors"
	"time"
)

var (
	ErrNotFound        = errors.New("sprint not found")
	ErrNameConflict    = errors.New("sprint name conflicts with an existing sprint")
	ErrVersionConflict = errors.New("sprint version conflict")
	ErrMemberOverlap   = errors.New("sprint overlaps for one or more selected members")
	ErrMemberNotFound  = errors.New("sprint member is unavailable")
	ErrTaskNotFound    = errors.New("sprint task is unavailable")
	ErrTaskUnscheduled = errors.New("sprint task is not scheduled")
	ErrTaskNotEligible = errors.New("sprint task is not eligible")
)

type OverlapMember struct {
	ID   string
	Name string
}

type MemberOverlapError struct {
	SprintID   string
	SprintName string
	StartDate  time.Time
	EndDate    time.Time
	Members    []OverlapMember
}

func (err *MemberOverlapError) Error() string { return ErrMemberOverlap.Error() }
func (err *MemberOverlapError) Unwrap() error { return ErrMemberOverlap }
