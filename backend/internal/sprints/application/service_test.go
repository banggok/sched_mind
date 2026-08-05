package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/sprints/domain"
)

type sprintStoreStub struct {
	sprint     *domain.Sprint
	created    *domain.Sprint
	updated    *domain.Sprint
	oldVersion int64
	priorTasks map[string]struct{}
}

func (store *sprintStoreStub) List(context.Context, listing.Query) (listing.Page[domain.Sprint], error) {
	return listing.Page[domain.Sprint]{}, nil
}
func (store *sprintStoreStub) Find(context.Context, string) (*domain.Sprint, error) {
	if store.sprint == nil {
		return nil, nil
	}
	copy := *store.sprint
	copy.MemberIDs = append([]string(nil), store.sprint.MemberIDs...)
	copy.TaskIDs = append([]string(nil), store.sprint.TaskIDs...)
	return &copy, nil
}
func (store *sprintStoreStub) Detail(context.Context, string) (*Detail, error) { return nil, nil }
func (store *sprintStoreStub) Suggest(context.Context, SuggestionInput) (*Suggestion, error) {
	return &Suggestion{}, nil
}
func (store *sprintStoreStub) Candidates(context.Context, string, listing.Query) (listing.Page[TaskProjection], error) {
	return listing.Page[TaskProjection]{}, nil
}
func (store *sprintStoreStub) DraftCandidates(context.Context, CandidateInput, listing.Query) (listing.Page[TaskProjection], error) {
	return listing.Page[TaskProjection]{}, nil
}
func (store *sprintStoreStub) Create(_ context.Context, sprint domain.Sprint) error {
	store.created = &sprint
	store.sprint = &sprint
	return nil
}
func (store *sprintStoreStub) Update(_ context.Context, sprint domain.Sprint, oldVersion int64, priorTasks map[string]struct{}) error {
	store.updated = &sprint
	store.oldVersion = oldVersion
	store.priorTasks = priorTasks
	store.sprint = &sprint
	return nil
}
func (store *sprintStoreStub) Start(context.Context, string, int64, time.Time) (*domain.Sprint, error) {
	return nil, errors.New("not used")
}
func (store *sprintStoreStub) Delete(context.Context, string, int64) error { return nil }

func TestCreatePersistsNormalizedPlannedAggregate_AC45And66(t *testing.T) {
	now := time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC)
	store := &sprintStoreStub{}
	service := NewServiceWithDependencies(store, func() time.Time { return now }, func() (string, error) { return "sprint-1", nil })

	created, err := service.Create(context.Background(), WriteInput{
		Name: "  Sprint One ", StartDate: now, EndDate: now.AddDate(0, 0, 7),
		MemberIDs: []string{"member-1"}, TaskIDs: []string{"task-1"},
	})
	if err != nil {
		t.Fatalf("create sprint: %v", err)
	}
	if store.created == nil || created.ID != "sprint-1" || created.Name != "Sprint One" || created.Status != domain.StatusPlanned || created.Version != 1 {
		t.Fatalf("unexpected created aggregate: %#v", created)
	}
}

func TestUpdateRejectsStaleVersionBeforePersistence_AC46And72(t *testing.T) {
	now := time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC)
	existing, err := domain.NewSprint("sprint-1", "Sprint", now, now, []string{"member-1"}, []string{"task-1"}, now)
	if err != nil {
		t.Fatal(err)
	}
	store := &sprintStoreStub{sprint: existing}
	service := NewServiceWithDependencies(store, func() time.Time { return now }, func() (string, error) { return "unused", nil })

	_, err = service.Update(context.Background(), existing.ID, WriteInput{
		Name: existing.Name, StartDate: existing.StartDate, EndDate: existing.EndDate,
		MemberIDs: existing.MemberIDs, TaskIDs: existing.TaskIDs, Version: 99,
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("got %v, want version conflict", err)
	}
	if store.updated != nil {
		t.Fatal("stale update reached persistence")
	}
}

func TestUpdatePassesPriorTaskSetForDriftAwareAtomicReplacement_AC46To58(t *testing.T) {
	now := time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC)
	existing, err := domain.NewSprint("sprint-1", "Sprint", now, now, []string{"member-1"}, []string{"retained", "removed"}, now)
	if err != nil {
		t.Fatal(err)
	}
	store := &sprintStoreStub{sprint: existing}
	service := NewServiceWithDependencies(store, func() time.Time { return now.Add(time.Hour) }, func() (string, error) { return "unused", nil })

	updated, err := service.Update(context.Background(), existing.ID, WriteInput{
		Name: "Renamed", StartDate: existing.StartDate, EndDate: existing.EndDate,
		MemberIDs: []string{"member-1"}, TaskIDs: []string{"retained", "new"}, Version: 1,
	})
	if err != nil {
		t.Fatalf("update sprint: %v", err)
	}
	if store.oldVersion != 1 || updated.Version != 2 {
		t.Fatalf("unexpected versions: old=%d updated=%d", store.oldVersion, updated.Version)
	}
	if _, ok := store.priorTasks["retained"]; !ok {
		t.Fatal("existing membership was not supplied for drift-aware validation")
	}
	if _, ok := store.priorTasks["new"]; ok {
		t.Fatal("new task was incorrectly treated as retained membership")
	}
}
