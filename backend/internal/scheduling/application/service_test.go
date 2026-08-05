package application

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"

	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
)

type serviceStoreSpy struct {
	requested            []string
	recommendationInput  *schedulingdomain.AssigneeRecommendationInput
	recommendationResult *schedulingdomain.AssigneeRecommendationResult
	recommendationError  error
}

func (spy *serviceStoreSpy) RecommendAssignees(_ context.Context, input schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error) {
	copy := input
	spy.recommendationInput = &copy
	return spy.recommendationResult, spy.recommendationError
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

func TestRecommendAssigneesValidatesAndDelegatesOneBatch_D01_D03_D17_AC3_AC12(t *testing.T) {
	result := &schedulingdomain.AssigneeRecommendationResult{
		Mode: schedulingdomain.RecommendationAutomatic,
	}
	store := &serviceStoreSpy{recommendationResult: result}
	service := NewService(store)
	input := schedulingdomain.AssigneeRecommendationInput{
		ProjectID:                    "project",
		TaskID:                       "task",
		RoleID:                       "role",
		EffortMinutes:                480,
		LagDays:                      0,
		CapacityAllocationPercentage: 40,
		CalculatedOn:                 time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC),
	}

	value, err := service.RecommendAssignees(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if value != result || store.recommendationInput == nil || !reflect.DeepEqual(*store.recommendationInput, input) {
		t.Fatalf("value=%#v input=%#v", value, store.recommendationInput)
	}
}

func TestRecommendAssigneesRejectsInvalidInputBeforeStore_D03_AC3(t *testing.T) {
	store := &serviceStoreSpy{}
	service := NewService(store)

	_, err := service.RecommendAssignees(context.Background(), schedulingdomain.AssigneeRecommendationInput{})
	if !errors.Is(err, schedulingdomain.ErrAssigneeRecommendationInputInvalid) {
		t.Fatalf("error=%v", err)
	}
	if store.recommendationInput != nil {
		t.Fatalf("store received invalid input=%#v", store.recommendationInput)
	}
}

func TestRecommendAssigneesRejectsNilStoreResult_D17_AC31(t *testing.T) {
	store := &serviceStoreSpy{}
	service := NewService(store)
	input := schedulingdomain.AssigneeRecommendationInput{
		ProjectID:                    "project",
		TaskID:                       "task",
		RoleID:                       "role",
		EffortMinutes:                480,
		CapacityAllocationPercentage: 100,
		CalculatedOn:                 time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC),
	}

	_, err := service.RecommendAssignees(context.Background(), input)
	if !errors.Is(err, schedulingdomain.ErrAssigneeRecommendationUnavailable) {
		t.Fatalf("error=%v", err)
	}
}
