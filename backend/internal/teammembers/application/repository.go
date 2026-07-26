package application

import (
	"context"

	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/teammembers/domain"
)

type TeamMemberRecord struct {
	Member   domain.TeamMember
	RoleName string
}

type Repository interface {
	List(context.Context, listing.Query) (listing.Page[TeamMemberRecord], error)
	FindByID(context.Context, string) (*TeamMemberRecord, error)
	RoleExists(context.Context, string) (bool, error)
	Create(context.Context, domain.TeamMember) error
	Update(context.Context, domain.TeamMember) error
	DeleteIfNoActiveTask(context.Context, string) error
}
