package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/sprints/domain"
)

type WriteInput struct {
	Name      string
	StartDate time.Time
	EndDate   time.Time
	MemberIDs []string
	TaskIDs   []string
	Version   int64
}

type CandidateInput struct {
	StartDate       time.Time
	EndDate         time.Time
	MemberIDs       []string
	ExcludedTaskIDs []string
}

type Service struct {
	store Store
	now   func() time.Time
	newID func() (string, error)
}

func NewService(store Store) *Service {
	return NewServiceWithDependencies(store, time.Now, identity.NewUUID)
}

func NewServiceWithDependencies(store Store, now func() time.Time, newID func() (string, error)) *Service {
	return &Service{store: store, now: now, newID: newID}
}

func (service *Service) List(ctx context.Context, query listing.Query) (listing.Page[domain.Sprint], error) {
	page, err := service.store.List(ctx, query)
	if err != nil {
		return listing.Page[domain.Sprint]{}, fmt.Errorf("list sprints: %w", err)
	}
	return page, nil
}

func (service *Service) Get(ctx context.Context, id string) (*domain.Sprint, error) {
	sprint, err := service.store.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get sprint: %w", err)
	}
	if sprint == nil {
		return nil, ErrNotFound
	}
	return sprint, nil
}

func (service *Service) Detail(ctx context.Context, id string) (*Detail, error) {
	detail, err := service.store.Detail(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get sprint detail: %w", err)
	}
	if detail == nil {
		return nil, ErrNotFound
	}
	return detail, nil
}

func (service *Service) Suggest(ctx context.Context, input SuggestionInput) (*Suggestion, error) {
	if _, err := domain.NewSprint("suggestion", "Suggestion", input.StartDate, input.EndDate, input.MemberIDs, nil, service.now()); err != nil {
		return nil, err
	}
	suggestion, err := service.store.Suggest(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("suggest sprint tasks: %w", err)
	}
	if suggestion == nil {
		return nil, errors.New("suggest sprint tasks: repository returned nil without error")
	}
	return suggestion, nil
}

func (service *Service) Candidates(ctx context.Context, id string, query listing.Query) (listing.Page[TaskProjection], error) {
	page, err := service.store.Candidates(ctx, id, query)
	if err != nil {
		return listing.Page[TaskProjection]{}, fmt.Errorf("list sprint task candidates: %w", err)
	}
	return page, nil
}

func (service *Service) DraftCandidates(ctx context.Context, input CandidateInput, query listing.Query) (listing.Page[TaskProjection], error) {
	if _, err := domain.NewSprint("candidate-draft", "Candidate Draft", input.StartDate, input.EndDate, input.MemberIDs, input.ExcludedTaskIDs, service.now()); err != nil {
		return listing.Page[TaskProjection]{}, err
	}
	page, err := service.store.DraftCandidates(ctx, input, query)
	if err != nil {
		return listing.Page[TaskProjection]{}, fmt.Errorf("list draft sprint task candidates: %w", err)
	}
	return page, nil
}

func (service *Service) Create(ctx context.Context, input WriteInput) (*domain.Sprint, error) {
	id, err := service.newID()
	if err != nil {
		return nil, fmt.Errorf("create sprint ID: %w", err)
	}
	sprint, err := domain.NewSprint(id, input.Name, input.StartDate, input.EndDate, input.MemberIDs, input.TaskIDs, service.now())
	if err != nil {
		return nil, err
	}
	if sprint == nil {
		return nil, errors.New("create sprint: domain returned nil without error")
	}
	if err := service.store.Create(ctx, *sprint); err != nil {
		return nil, fmt.Errorf("create sprint: %w", err)
	}
	return service.Get(ctx, sprint.ID)
}

func (service *Service) Update(ctx context.Context, id string, input WriteInput) (*domain.Sprint, error) {
	sprint, err := service.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if sprint.Version != input.Version {
		return nil, ErrVersionConflict
	}
	previousTasks := make(map[string]struct{}, len(sprint.TaskIDs))
	for _, taskID := range sprint.TaskIDs {
		previousTasks[taskID] = struct{}{}
	}
	if err := sprint.Update(input.Name, input.StartDate, input.EndDate, input.MemberIDs, input.TaskIDs, service.now()); err != nil {
		return nil, err
	}
	if err := service.store.Update(ctx, *sprint, input.Version, previousTasks); err != nil {
		return nil, fmt.Errorf("update sprint: %w", err)
	}
	return service.Get(ctx, id)
}

func (service *Service) Start(ctx context.Context, id string, version int64) (*domain.Sprint, error) {
	sprint, err := service.store.Start(ctx, id, version, service.now())
	if err != nil {
		return nil, fmt.Errorf("start sprint: %w", err)
	}
	if sprint == nil {
		return nil, ErrNotFound
	}
	return sprint, nil
}

func (service *Service) Delete(ctx context.Context, id string, version int64) error {
	if err := service.store.Delete(ctx, id, version); err != nil {
		return fmt.Errorf("delete sprint: %w", err)
	}
	return nil
}
