package application

import (
	"context"
	"time"

	"github.com/banggok/sched_mind/backend/internal/publicholidays/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type ListQuery struct {
	listing.Query
	HolidayDate *time.Time
	Today       time.Time
}

type Repository interface {
	List(context.Context, ListQuery) (listing.Page[domain.PublicHoliday], error)
	Find(context.Context, string) (*domain.PublicHoliday, error)
	Create(context.Context, domain.PublicHoliday) error
	Update(context.Context, domain.PublicHoliday) error
	Delete(context.Context, string) error
	ExistsOn(context.Context, time.Time) (bool, error)
	DatesBetween(context.Context, time.Time, time.Time) ([]time.Time, error)
}
