package application

import (
	"context"
	"errors"
	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"testing"
	"time"
)

type fakeRepository struct {
	values    map[string]domain.CapacityOverride
	err       error
	listQuery ListQuery
}

func (f *fakeRepository) List(_ context.Context, _ string, query ListQuery) (listing.Page[domain.CapacityOverride], error) {
	f.listQuery = query
	return listing.Page[domain.CapacityOverride]{}, f.err
}
func (f *fakeRepository) Find(_ context.Context, member, id string) (*domain.CapacityOverride, error) {
	if f.err != nil {
		return nil, f.err
	}
	value, ok := f.values[id]
	if !ok || value.TeamMemberID != member {
		return nil, domain.ErrNotFound
	}
	return &value, nil
}
func (f *fakeRepository) Create(_ context.Context, v domain.CapacityOverride) error {
	if f.err != nil {
		return f.err
	}
	f.values[v.ID] = v
	return nil
}
func (f *fakeRepository) Update(_ context.Context, v domain.CapacityOverride) error {
	if f.err != nil {
		return f.err
	}
	f.values[v.ID] = v
	return nil
}
func (f *fakeRepository) Delete(_ context.Context, member, id string) error {
	v, ok := f.values[id]
	if !ok || v.TeamMemberID != member {
		return domain.ErrNotFound
	}
	delete(f.values, id)
	return nil
}
func TestServiceCreateUpdateDelete(t *testing.T) {
	repo := &fakeRepository{values: map[string]domain.CapacityOverride{}}
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	service := NewServiceWithDependencies(repo, func() time.Time { return now }, func() (string, error) { return "override", nil })
	capacity := 4.0
	value, err := service.Create(context.Background(), "member", WriteInput{StartDate: now, EndDate: now, Capacity: &capacity})
	if err != nil || value == nil {
		t.Fatalf("create: %v", err)
	}
	zero := 0.0
	value, err = service.Update(context.Background(), "member", "override", WriteInput{StartDate: now, EndDate: now, Capacity: &zero})
	if err != nil || value.Capacity.Hours() != 0 {
		t.Fatalf("update: %v", err)
	}
	if err := service.Delete(context.Background(), "member", "override"); err != nil {
		t.Fatal(err)
	}
}
func TestServicePreservesDependencyErrors(t *testing.T) {
	sentinel := errors.New("database unavailable")
	service := NewService(&fakeRepository{err: sentinel})
	_, err := service.List(context.Background(), "member", ListQuery{Query: listing.Query{Page: 1, PageSize: 5}})
	if !errors.Is(err, sentinel) {
		t.Fatalf("cause not preserved: %v", err)
	}
}

func TestServicePassesOptionalEffectiveDateToRepository(t *testing.T) {
	repository := &fakeRepository{}
	service := NewService(repository)
	effectiveDate := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	_, err := service.List(context.Background(), "member", ListQuery{
		Query: listing.Query{Page: 2, PageSize: 5}, EffectiveDate: &effectiveDate,
	})
	if err != nil || repository.listQuery.EffectiveDate == nil ||
		!repository.listQuery.EffectiveDate.Equal(effectiveDate) ||
		repository.listQuery.Page != 2 {
		t.Fatalf("query=%+v err=%v", repository.listQuery, err)
	}
}
