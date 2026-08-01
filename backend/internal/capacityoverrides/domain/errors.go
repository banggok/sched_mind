package domain

import "errors"

var (
	ErrStartDateRequired        = errors.New("start date is required")
	ErrEndDateRequired          = errors.New("end date is required")
	ErrInvalidDateRange         = errors.New("end date must not be earlier than start date")
	ErrCapacityRequired         = errors.New("capacity is required")
	ErrCapacityNegative         = errors.New("capacity must not be negative")
	ErrCapacityExceedsLimit     = errors.New("capacity must not exceed 24 hours")
	ErrCapacityInvalidIncrement = errors.New("capacity must use 0.5-hour increments")
	ErrDescriptionRequired      = errors.New("description is required")
	ErrDescriptionTooLong       = errors.New("description must not exceed 100 characters")
	ErrDuplicate                = errors.New("capacity override duplicates an existing record")
	ErrNotFound                 = errors.New("capacity override not found")
	ErrTeamMemberNotFound       = errors.New("team member not found")
)
