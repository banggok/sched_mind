package application

import (
	"context"
	"time"

	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/sprints/domain"
)

type Store interface {
	List(context.Context, listing.Query) (listing.Page[domain.Sprint], error)
	Find(context.Context, string) (*domain.Sprint, error)
	Detail(context.Context, string) (*Detail, error)
	Suggest(context.Context, SuggestionInput) (*Suggestion, error)
	Candidates(context.Context, string, listing.Query) (listing.Page[TaskProjection], error)
	DraftCandidates(context.Context, CandidateInput, listing.Query) (listing.Page[TaskProjection], error)
	Create(context.Context, domain.Sprint) error
	Update(context.Context, domain.Sprint, int64, map[string]struct{}) error
	Start(context.Context, string, int64, time.Time) (*domain.Sprint, error)
	Delete(context.Context, string, int64) error
}
