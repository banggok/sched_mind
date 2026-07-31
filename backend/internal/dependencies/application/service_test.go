package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/dependencies/domain"
)

type storeStub struct {
	created            domain.Dependency
	createErr          error
	deleted            string
	deleteErr          error
	kept               string
	keepErr            error
	affectedProjectIDs []string
}

func (s *storeStub) List(context.Context, string) (*domain.Detail, error) {
	return &domain.Detail{BlockedBy: []domain.Item{}, Blocks: []domain.Item{}}, nil
}

func (s *storeStub) Candidates(context.Context, string, domain.Direction, string, int, int) (*domain.CandidatePage, error) {
	return &domain.CandidatePage{}, nil
}

func (s *storeStub) Create(
	ctx context.Context,
	v domain.Dependency,
	callback func(context.Context, []string) error,
) (*domain.Dependency, error) {
	s.created = v

	if s.createErr != nil {
		return nil, s.createErr
	}

	projectIDs := s.affectedProjectIDs
	if len(projectIDs) == 0 {
		projectIDs = []string{"project"}
	}

	if err := callback(ctx, projectIDs); err != nil {
		return nil, err
	}

	return &v, nil
}

func (s *storeStub) Delete(ctx context.Context, id string, _ time.Time, callback func(context.Context, []string) error) error {
	s.deleted = id
	if s.deleteErr != nil {
		return s.deleteErr
	}
	return callback(ctx, []string{"project"})
}

func (s *storeStub) KeepAsManual(ctx context.Context, id string, now time.Time, callback func(context.Context, []string) error) (*domain.Dependency, error) {
	s.kept = id
	if s.keepErr != nil {
		return nil, s.keepErr
	}
	if err := callback(ctx, []string{"project"}); err != nil {
		return nil, err
	}
	return &domain.Dependency{ID: id, BlockingTaskID: "a", BlockedTaskID: "b", ManualOwned: true, AutomaticOwned: true, CreatedAt: now, UpdatedAt: now}, nil
}

type schedulerStub struct {
	calls      int
	projectIDs []string
	err        error
}

func (s *schedulerStub) InvalidatePortfolio(
	_ context.Context,
	projectIDs []string,
) error {
	s.calls++
	s.projectIDs = append([]string(nil), projectIDs...)
	return s.err
}

func TestCreateOrchestratesStoreAndScheduler(t *testing.T) {
	store := &storeStub{}
	scheduler := &schedulerStub{}
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	service := NewServiceWithDependencies(store, scheduler, func() time.Time { return now }, func() (string, error) { return "dependency", nil })
	value, err := service.Create(context.Background(), "a", "b")
	if err != nil {
		t.Fatalf("Create() error=%v", err)
	}
	if value == nil || store.created.ID != "dependency" || scheduler.calls != 1 {
		t.Fatalf("result=%#v store=%#v calls=%d", value, store.created, scheduler.calls)
	}
}

func TestCreateReturnsNilOnDependencyFailure(t *testing.T) {
	store := &storeStub{createErr: errors.New("database unavailable")}
	service := NewServiceWithDependencies(store, NoopScheduler{}, time.Now, func() (string, error) { return "dependency", nil })
	value, err := service.Create(context.Background(), "a", "b")
	if value != nil || err == nil {
		t.Fatalf("Create()=(%#v,%v)", value, err)
	}
}

func TestDeletePropagatesSchedulerFailure(t *testing.T) {
	scheduler := &schedulerStub{err: errors.New("scheduler unavailable")}
	service := NewServiceWithDependencies(&storeStub{}, scheduler, time.Now, func() (string, error) { return "id", nil })
	if err := service.Delete(context.Background(), "dependency"); err == nil {
		t.Fatal("Delete() error=nil")
	}
}

func TestCreateInvalidatesAllAffectedProjects(t *testing.T) {
	store := &storeStub{
		affectedProjectIDs: []string{"alpha", "beta"},
	}
	scheduler := &schedulerStub{}

	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)

	service := NewServiceWithDependencies(
		store,
		scheduler,
		func() time.Time { return now },
		func() (string, error) { return "dependency", nil },
	)

	value, err := service.Create(
		context.Background(),
		"task-a",
		"task-b",
	)
	if err != nil {
		t.Fatalf("Create() error=%v", err)
	}

	if value == nil {
		t.Fatal("Create() result=nil")
	}

	if scheduler.calls != 1 {
		t.Fatalf(
			"scheduler.calls=%d, want 1",
			scheduler.calls,
		)
	}

	if len(scheduler.projectIDs) != 2 {
		t.Fatalf(
			"scheduler.projectIDs=%v, want [alpha beta]",
			scheduler.projectIDs,
		)
	}

	seen := map[string]bool{}
	for _, projectID := range scheduler.projectIDs {
		seen[projectID] = true
	}

	if !seen["alpha"] || !seen["beta"] {
		t.Fatalf(
			"scheduler.projectIDs=%v, want both alpha and beta",
			scheduler.projectIDs,
		)
	}
}

func TestKeepAsManualOrchestratesStoreAndScheduler_AC24(t *testing.T) {
	store := &storeStub{}
	scheduler := &schedulerStub{}
	service := NewServiceWithDependencies(store, scheduler, time.Now, func() (string, error) { return "unused", nil })

	value, err := service.KeepAsManual(context.Background(), "dependency")
	if err != nil {
		t.Fatalf("KeepAsManual() error = %v", err)
	}
	if value == nil || store.kept != "dependency" || scheduler.calls != 1 {
		t.Fatalf("value=%#v kept=%q scheduler.calls=%d", value, store.kept, scheduler.calls)
	}
	if !value.ManualOwned || !value.AutomaticOwned {
		t.Fatalf("ownership = manual:%v automatic:%v, want shared", value.ManualOwned, value.AutomaticOwned)
	}
}
