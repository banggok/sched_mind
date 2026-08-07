package application

import (
	"context"

	"github.com/banggok/sched_mind/backend/internal/roles/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type MemberUsage struct {
	ID   string
	Name string
}

type Repository interface {
	List(context.Context, listing.Query) (listing.Page[domain.Role], error)
	ListMembers(context.Context, string, listing.Query) (listing.Page[MemberUsage], error)
	FindByID(context.Context, string) (*domain.Role, error)
	NameExists(context.Context, string, string) (bool, error)
	Create(context.Context, domain.Role) error
	Update(context.Context, domain.Role) error
	Delete(context.Context, string) error
}
