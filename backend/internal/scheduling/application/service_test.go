package application

import (
	"context"
	"reflect"
	"testing"

	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
)

type serviceStoreSpy struct {
	requested []string
}

func (spy *serviceStoreSpy) RecalculatePortfolio(_ context.Context, projectIDs []string) error {
	spy.requested = append([]string{}, projectIDs...)
	return nil
}
func (*serviceStoreSpy) RecalculateMemberSchedule(context.Context, string) error { return nil }
func (*serviceStoreSpy) MarkProjectUnscheduled(context.Context, string, string) error {
	return nil
}

func TestRecalculateActiveProjectsUsesMutationOwnerAsTransitiveClosureRoot_US62_AC22(t *testing.T) {
	store := &serviceStoreSpy{}
	service := NewService(store)
	ctx := schedulingimpact.WithOperation(context.Background(), "project-a", schedulingimpact.ModeOrdinary)

	if err := service.RecalculateActiveProjects(ctx); err != nil {
		t.Fatalf("recalculate active projects: %v", err)
	}
	if want := []string{"project-a"}; !reflect.DeepEqual(store.requested, want) {
		t.Fatalf("requested projects = %#v, want %#v", store.requested, want)
	}
}

func TestRecalculateActiveProjectsRetainsExplicitGlobalFallback(t *testing.T) {
	store := &serviceStoreSpy{requested: []string{"stale"}}
	service := NewService(store)

	if err := service.RecalculateActiveProjects(context.Background()); err != nil {
		t.Fatalf("recalculate active projects: %v", err)
	}
	if store.requested == nil || len(store.requested) != 0 {
		t.Fatalf("requested projects = %#v, want explicit empty global scope", store.requested)
	}
}
