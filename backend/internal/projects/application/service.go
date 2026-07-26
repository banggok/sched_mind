package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/banggok/sched_mind/backend/internal/projects/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type Service struct {
	store     Store
	scheduler Scheduler
	now       func() time.Time
	newID     func() (string, error)
}

func NewService(store Store, scheduler Scheduler) *Service {
	if scheduler == nil {
		scheduler = NoopScheduler{}
	}
	return &Service{store: store, scheduler: scheduler, now: func() time.Time { return time.Now().UTC() }, newID: newUUID}
}

func NewServiceWithDependencies(store Store, scheduler Scheduler, now func() time.Time, newID func() (string, error)) *Service {
	return &Service{store: store, scheduler: scheduler, now: now, newID: newID}
}

func (s *Service) List(ctx context.Context, query listing.Query) (listing.Page[domain.Project], error) {
	result, err := s.store.List(ctx, query)
	if err != nil {
		return listing.Page[domain.Project]{}, fmt.Errorf("list projects: %w", err)
	}
	return result, nil
}

func (s *Service) Get(ctx context.Context, id string) (*domain.Project, error) {
	value, err := s.store.Find(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	if value == nil {
		return nil, errors.New("get project: store returned nil without error")
	}
	return value, nil
}

func (s *Service) Create(ctx context.Context, name string) (*domain.Project, error) {
	if _, err := domain.NormalizeName(name); err != nil {
		return nil, err
	}
	id, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("generate project ID: %w", err)
	}
	value, err := s.store.CreateNext(ctx, id, name, s.now())
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	if value == nil {
		return nil, errors.New("create project: store returned nil without error")
	}
	return value, nil
}

func (s *Service) Update(ctx context.Context, id, name string) (*domain.Project, error) {
	value, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, errors.New("update project: get returned nil")
	}
	if err := value.Rename(name, s.now()); err != nil {
		return nil, err
	}
	if err := s.store.Update(ctx, *value); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}
	return value, nil
}

func (s *Service) ChangeStatus(ctx context.Context, id string, target domain.Status) (*domain.Project, error) {
	value, err := s.store.ChangeStatus(ctx, id, target, s.now())
	if err != nil {
		return nil, fmt.Errorf("change project status: %w", err)
	}
	if value == nil {
		return nil, errors.New("change project status: store returned nil without error")
	}
	return value, nil
}

func (s *Service) MovePriority(ctx context.Context, id string, direction domain.PriorityDirection) (*domain.Project, error) {
	if direction != domain.PriorityUp && direction != domain.PriorityDown {
		return nil, domain.ErrPriorityDirectionInvalid
	}
	value, err := s.store.MovePriority(ctx, id, direction, s.now(), s.scheduler.RecalculateActiveProjects)
	if err != nil {
		return nil, fmt.Errorf("move project priority: %w", err)
	}
	if value == nil {
		return nil, errors.New("move project priority: store returned nil without error")
	}
	return value, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	if err := s.store.DeleteChildless(ctx, id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

func newUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}
