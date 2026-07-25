package domain

import "errors"

var (
	ErrNameRequired              = errors.New("team member name is required")
	ErrNameTooLong               = errors.New("team member name must not exceed 100 characters")
	ErrRoleRequired              = errors.New("role is required")
	ErrRoleNotFound              = errors.New("role not found")
	ErrDailyCapacityRequired     = errors.New("daily capacity is required")
	ErrDailyCapacityNotPositive  = errors.New("daily capacity must be greater than 0")
	ErrDailyCapacityExceedsLimit = errors.New(
		"daily capacity must not exceed 24 hours",
	)
	ErrDailyCapacityInvalidIncrement = errors.New(
		"daily capacity must use 0.5-hour increments",
	)
	ErrBufferOutOfRange = errors.New("buffer must be between 0 and less than 100")
	ErrBufferPrecision  = errors.New("buffer must not exceed two decimal places")
	ErrNotFound         = errors.New("team member not found")
	ErrAssignedToTask   = errors.New(
		"team member is assigned to one or more tasks and cannot be deleted",
	)
	ErrHasCapacityOverride = errors.New(
		"team member has capacity overrides and cannot be deleted",
	)
)
