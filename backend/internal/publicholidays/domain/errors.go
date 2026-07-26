package domain

import "errors"

var (
	ErrStartDateRequired   = errors.New("start date is required")
	ErrEndDateRequired     = errors.New("end date is required")
	ErrInvalidDateRange    = errors.New("end date must not be earlier than start date")
	ErrNoWorkingDates      = errors.New("date range must include at least one weekday")
	ErrDescriptionRequired = errors.New("description is required")
	ErrDescriptionTooLong  = errors.New("description must not exceed 100 characters")
	ErrDateAlreadyExists   = errors.New("a public holiday already exists on one or more selected dates")
	ErrNotFound            = errors.New("public holiday not found")
)
