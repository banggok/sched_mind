package domain

import "errors"

var (
	ErrNameRequired     = errors.New("sprint name is required")
	ErrNameTooLong      = errors.New("sprint name must not exceed 200 characters")
	ErrDateRangeInvalid = errors.New("sprint end date must not be before start date")
	ErrMemberRequired   = errors.New("at least one sprint member is required")
	ErrAlreadyStarted   = errors.New("sprint is already started")
)
