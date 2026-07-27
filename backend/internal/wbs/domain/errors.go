package domain

import "errors"

var (
	ErrNotFound              = errors.New("WBS was not found")
	ErrParentNotFound        = errors.New("parent WBS was not found")
	ErrProjectNotFound       = errors.New("project was not found")
	ErrNameRequired          = errors.New("WBS name is required")
	ErrNameTooLong           = errors.New("WBS name must be 100 characters or fewer")
	ErrNameExists            = errors.New("a sibling WBS with this name already exists")
	ErrHasChildren           = errors.New("move or delete this WBS's children first")
	ErrCycle                 = errors.New("WBS cannot be moved under itself or its descendant")
	ErrProjectMismatch       = errors.New("WBS parent must belong to the same project")
	ErrConversionRequired    = errors.New("confirm moving executable data to the first child")
	ErrExecutableOnly        = errors.New("executable fields are available only on a leaf WBS")
	ErrManualTimeline        = errors.New("manual timeline is available only for an open project with automatic scheduling off")
	ErrTimelinePair          = errors.New("timeline start and end must both be provided")
	ErrTimelineOrder         = errors.New("timeline end must be on or after start")
	ErrEffortInvalid         = errors.New("effort must be at least 0.5 hours in 0.5 hour increments")
	ErrRoleAssigneeMismatch  = errors.New("assignee role does not match the selected role")
	ErrCompletedReadOnly     = errors.New("completed executable WBS is read-only")
	ErrProjectClosedReadOnly = errors.New("closed project is read-only")
	ErrMoveNotAllowed        = errors.New("WBS cannot move further in that direction")
)
