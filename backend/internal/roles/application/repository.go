package application

import (
	"context"

	"github.com/banggok/sched_mind/backend/internal/roles/domain"
)

type Repository interface {
	List(context.Context) ([]domain.Role, error)
	FindByID(context.Context, string) (*domain.Role, error)
	NameExists(context.Context, string, string) (bool, error)
	Create(context.Context, domain.Role) error
	Update(context.Context, domain.Role) error
	IsInUse(context.Context, string) (bool, error)
	Delete(context.Context, string) error
}
