package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/projects/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type storeStub struct {
	items map[string]domain.Project
	move  func(context.Context, string, domain.PriorityDirection, time.Time, func(context.Context) error) (*domain.Project, error)
}

func newStoreStub() *storeStub { return &storeStub{items: map[string]domain.Project{}} }
func (s *storeStub) List(_ context.Context, query listing.Query) (listing.Page[domain.Project], error) {
	values := make([]domain.Project, 0, len(s.items))
	for _, value := range s.items {
		values = append(values, value)
	}
	return listing.Page[domain.Project]{Items: values, Page: query.Page, PageSize: query.PageSize, Total: int64(len(values))}, nil
}
func (s *storeStub) Find(_ context.Context, id string) (*domain.Project, error) {
	value, ok := s.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &value, nil
}
func (s *storeStub) CreateNext(_ context.Context, id, name string, now time.Time) (*domain.Project, error) {
	value, err := domain.NewProject(id, name, len(s.items)+1, now)
	if err != nil {
		return nil, err
	}
	s.items[id] = *value
	return value, nil
}
func (s *storeStub) Update(_ context.Context, value domain.Project) error {
	s.items[value.ID] = value
	return nil
}
func (s *storeStub) DeleteChildless(_ context.Context, id string) error {
	if _, ok := s.items[id]; !ok {
		return domain.ErrNotFound
	}
	delete(s.items, id)
	return nil
}
func (s *storeStub) ChangeStatus(_ context.Context, id string, target domain.Status, now time.Time) (*domain.Project, error) {
	value, ok := s.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	if err := value.ChangeStatus(target, false, false, nil, nil, now); err != nil {
		return nil, err
	}
	s.items[id] = value
	return &value, nil
}
func (s *storeStub) MovePriority(ctx context.Context, id string, direction domain.PriorityDirection, now time.Time, schedule func(context.Context) error) (*domain.Project, error) {
	return s.move(ctx, id, direction, now, schedule)
}

type schedulerStub struct {
	err   error
	calls int
}

func (s *schedulerStub) RecalculateActiveProjects(context.Context) error { s.calls++; return s.err }

func TestServiceCreateGetUpdateListDelete(t *testing.T) {
	store := newStoreStub()
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	service := NewServiceWithDependencies(store, &schedulerStub{}, func() time.Time { return now }, func() (string, error) { return "p1", nil })
	created, err := service.Create(context.Background(), " Alpha ")
	if err != nil || created == nil || created.Name != "Alpha" {
		t.Fatalf("create: %#v %v", created, err)
	}
	got, err := service.Get(context.Background(), "p1")
	if err != nil || got == nil {
		t.Fatalf("get: %#v %v", got, err)
	}
	updated, err := service.Update(context.Background(), "p1", "Beta")
	if err != nil || updated == nil || updated.Name != "Beta" {
		t.Fatalf("update: %#v %v", updated, err)
	}
	page, err := service.List(context.Background(), listing.Query{Page: 1, PageSize: 5})
	if err != nil || page.Total != 1 {
		t.Fatalf("list: %#v %v", page, err)
	}
	if err := service.Delete(context.Background(), "p1"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Get(context.Background(), "p1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("not found: %v", err)
	}
}

func TestServicePriorityCoordinatesSchedulerAndPropagatesFailure(t *testing.T) {
	store := newStoreStub()
	scheduler := &schedulerStub{}
	store.move = func(ctx context.Context, _ string, _ domain.PriorityDirection, _ time.Time, schedule func(context.Context) error) (*domain.Project, error) {
		if err := schedule(ctx); err != nil {
			return nil, err
		}
		value, _ := domain.NewProject("p1", "Alpha", 1, time.Now())
		return value, nil
	}
	service := NewServiceWithDependencies(store, scheduler, time.Now, func() (string, error) { return "p1", nil })
	value, err := service.MovePriority(context.Background(), "p1", domain.PriorityUp)
	if err != nil || value == nil || scheduler.calls != 1 {
		t.Fatalf("success: %#v %v calls=%d", value, err, scheduler.calls)
	}
	scheduler.err = errors.New("scheduler down")
	if _, err := service.MovePriority(context.Background(), "p1", domain.PriorityDown); err == nil {
		t.Fatal("expected dependency failure")
	}
}
