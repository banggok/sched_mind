package application

import (
	"context"
	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"time"
)

type ListQuery struct {
	listing.Query
	EffectiveDate *time.Time
}

type Repository interface {
	List(context.Context, string, ListQuery) (listing.Page[domain.CapacityOverride], error)
	Find(context.Context, string, string) (*domain.CapacityOverride, error)
	Create(context.Context, domain.CapacityOverride) error
	Update(context.Context, domain.CapacityOverride) error
	Delete(context.Context, string, string) error
}
