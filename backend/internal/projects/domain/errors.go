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
	ErrLockedReadOnly             = errors.New("locked project is read-only except for Project Name and Actual Date completion")
	ErrCannotLockUnscheduled      = errors.New("project cannot lock while unfinished tasks are not fully scheduled")
	ErrCannotLockWithoutTasks     = errors.New("project cannot lock without tasks")
	ErrCannotCloseWithActiveTasks = errors.New("project cannot close while tasks are unfinished")
	ErrCannotCloseWithoutTasks    = errors.New("project cannot close without tasks")
	ErrSettingsReadOnly           = errors.New("project settings can only be changed while the project is open")
	ErrProjectBufferInvalid       = errors.New("project buffer must be between 0 and 100")
	ErrHasChildren                = errors.New("project with children cannot be deleted")
	ErrPriorityInvalid            = errors.New("project priority must be a positive integer")
	ErrPriorityDirectionInvalid   = errors.New("project priority direction must be up or down")
	ErrPriorityMoveNotAllowed     = errors.New("project priority cannot move in that direction")
	ErrBulkReopenRequired         = errors.New("project reopen requires related locked projects")
	ErrBulkReopenStale            = errors.New("project reopen plan is stale")
)

type ReopenProject struct {
	ID      string
	Name    string
	Version int64
}

type BulkReopenRequiredError struct {
	RootProjectID string
	Locked        []ReopenProject
	Open          []ReopenProject
	Token         string
}

func (err BulkReopenRequiredError) Error() string { return ErrBulkReopenRequired.Error() }
func (err BulkReopenRequiredError) Unwrap() error { return ErrBulkReopenRequired }
