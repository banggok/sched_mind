package application

import (
	"context"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
)

type Store interface {
	RecalculatePortfolio(context.Context, []string) error
	RecommendAssignees(context.Context, schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error)
	RecalculateMemberSchedule(context.Context, string) error
	MarkProjectUnscheduled(context.Context, string, string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service { return &Service{store: store} }

func (service *Service) RecommendAssignees(ctx context.Context, input schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	value, err := service.store.RecommendAssignees(ctx, input)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, schedulingdomain.ErrAssigneeRecommendationUnavailable
	}
	return value, nil
}

func (service *Service) RecalculateActiveProjects(ctx context.Context) error {
	enabled, ownerProjectID, _, _ := schedulingimpact.Operation(ctx)
	if enabled && ownerProjectID != "" {
		return service.store.RecalculatePortfolio(ctx, []string{ownerProjectID})
	}
	return service.store.RecalculatePortfolio(ctx, nil)
}

func (service *Service) RecalculateProjectSchedule(ctx context.Context, projectID string) error {
	return service.store.RecalculatePortfolio(ctx, []string{projectID})
}

func (service *Service) RecalculateMemberSchedule(ctx context.Context, memberID string) error {
	return service.store.RecalculateMemberSchedule(ctx, memberID)
}

func (service *Service) MarkProjectUnscheduled(ctx context.Context, projectID, reason string) error {
	return service.store.MarkProjectUnscheduled(ctx, projectID, reason)
}

func (service *Service) InvalidatePortfolio(ctx context.Context, projectIDs []string) error {
	return service.store.RecalculatePortfolio(ctx, projectIDs)
}

// Forecast remains a separate Epic 6 contract. US-6.1 explicitly preserves
// Actual End and Reopen coordination instead of converting them into a full
// Execution/Commitment scheduling trigger.
func (service *Service) RecalculateProjectForecast(context.Context, string) error { return nil }
