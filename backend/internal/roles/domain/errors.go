package domain

import "errors"

var (
	ErrNameRequired = errors.New("role name is required")
	ErrNameTooLong  = errors.New("role name must not exceed 100 characters")
	ErrNameInvalid  = errors.New("role name contains unsupported characters")
	ErrNameExists   = errors.New("role name already exists")
	ErrNotFound     = errors.New("role not found")
	ErrInUse        = errors.New("role is assigned to one or more team members and cannot be deleted")
)
