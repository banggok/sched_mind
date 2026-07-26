package domain

import "errors"

var (
	ErrNameRequired               = errors.New("project name is required")
	ErrNameTooLong                = errors.New("project name must not exceed 100 characters")
	ErrNameExists                 = errors.New("project name already exists")
	ErrNotFound                   = errors.New("project not found")
	ErrStatusInvalid              = errors.New("project status is invalid")
	ErrStatusTransitionNotAllowed = errors.New("project status transition is not allowed")
	ErrClosedReadOnly             = errors.New("closed project is read-only")
	ErrCannotLockWithoutTasks     = errors.New("project cannot lock without tasks")
	ErrCannotCloseWithActiveTasks = errors.New("project cannot close while tasks are unfinished")
	ErrCannotCloseWithoutTasks    = errors.New("project cannot close without tasks")
	ErrHasChildren                = errors.New("project with children cannot be deleted")
	ErrPriorityInvalid            = errors.New("project priority must be a positive integer")
	ErrPriorityDirectionInvalid   = errors.New("project priority direction must be up or down")
	ErrPriorityMoveNotAllowed     = errors.New("project priority cannot move in that direction")
)
