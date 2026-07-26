package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type WriteInput struct {
	StartDate, EndDate time.Time
	Capacity           *float64
}
type Service struct {
	repository Repository
	now        func() time.Time
	newID      func() (string, error)
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository, now: time.Now, newID: newUUID}
}
func NewServiceWithDependencies(repository Repository, now func() time.Time, newID func() (string, error)) *Service {
	return &Service{repository: repository, now: now, newID: newID}
}

func (s *Service) List(ctx context.Context, memberID string, query ListQuery) (listing.Page[domain.CapacityOverride], error) {
	result, err := s.repository.List(ctx, memberID, query)
	if err != nil {
		return listing.Page[domain.CapacityOverride]{}, fmt.Errorf("list capacity overrides: %w", err)
	}
	return result, nil
}
func (s *Service) Get(ctx context.Context, memberID, id string) (*domain.CapacityOverride, error) {
	result, err := s.repository.Find(ctx, memberID, id)
	if err != nil {
		return nil, fmt.Errorf("get capacity override: %w", err)
	}
	if result == nil {
		return nil, errors.New("get capacity override: repository returned nil without error")
	}
	return result, nil
}
func (s *Service) Create(ctx context.Context, memberID string, input WriteInput) (*domain.CapacityOverride, error) {
	capacity, err := capacityFrom(input.Capacity)
	if err != nil {
		return nil, err
	}
	id, err := s.newID()
	if err != nil {
		return nil, fmt.Errorf("create capacity override ID: %w", err)
	}
	override, err := domain.New(id, memberID, input.StartDate, input.EndDate, capacity, s.now())
	if err != nil {
		return nil, err
	}
	if override == nil {
		return nil, errors.New("create capacity override: domain returned nil without error")
	}
	if err := s.repository.Create(ctx, *override); err != nil {
		return nil, fmt.Errorf("create capacity override: %w", err)
	}
	return s.Get(ctx, memberID, id)
}
func (s *Service) Update(ctx context.Context, memberID, id string, input WriteInput) (*domain.CapacityOverride, error) {
	capacity, err := capacityFrom(input.Capacity)
	if err != nil {
		return nil, err
	}
	override, err := s.repository.Find(ctx, memberID, id)
	if err != nil {
		return nil, fmt.Errorf("find capacity override for update: %w", err)
	}
	if override == nil {
		return nil, errors.New("update capacity override: repository returned nil without error")
	}
	if err := override.Update(input.StartDate, input.EndDate, capacity, s.now()); err != nil {
		return nil, err
	}
	if err := s.repository.Update(ctx, *override); err != nil {
		return nil, fmt.Errorf("update capacity override: %w", err)
	}
	return s.Get(ctx, memberID, id)
}
func (s *Service) Delete(ctx context.Context, memberID, id string) error {
	if err := s.repository.Delete(ctx, memberID, id); err != nil {
		return fmt.Errorf("delete capacity override: %w", err)
	}
	return nil
}
func capacityFrom(value *float64) (domain.Capacity, error) {
	if value == nil {
		return domain.Capacity{}, domain.ErrCapacityRequired
	}
	return domain.NewCapacity(*value)
}
func newUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate UUID: %w", err)
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", value[0:4], value[4:6], value[6:8], value[8:10], value[10:16]), nil
}
