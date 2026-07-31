package application

import "context"

type Store interface {
	RecalculatePortfolio(context.Context, []string) error
	MarkProjectUnscheduled(context.Context, string, string) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service { return &Service{store: store} }

func (service *Service) RecalculateActiveProjects(ctx context.Context) error {
	return service.store.RecalculatePortfolio(ctx, nil)
}

func (service *Service) RecalculateProjectSchedule(ctx context.Context, projectID string) error {
	return service.store.RecalculatePortfolio(ctx, []string{projectID})
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
