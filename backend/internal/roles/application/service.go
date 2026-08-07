package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/banggok/sched_mind/backend/internal/roles/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type Service struct {
	repository Repository
	now        func() time.Time
	newID      func() (string, error)
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
		now:        func() time.Time { return time.Now().UTC() },
		newID:      identity.NewUUID,
	}
}

func NewServiceWithDependencies(
	repository Repository,
	now func() time.Time,
	newID func() (string, error),
) *Service {
	return &Service{repository: repository, now: now, newID: newID}
}

func (service *Service) List(ctx context.Context, query listing.Query) (listing.Page[domain.Role], error) {
	return service.repository.List(ctx, query)
}

func (service *Service) ListMembers(
	ctx context.Context,
	roleID string,
	query listing.Query,
) (listing.Page[MemberUsage], error) {
	role, err := service.repository.FindByID(ctx, roleID)
	if err != nil {
		return listing.Page[MemberUsage]{}, err
	}
	if role == nil {
		return listing.Page[MemberUsage]{}, errors.New("list role members: repository returned nil without error")
	}
	result, err := service.repository.ListMembers(ctx, roleID, query)
	if err != nil {
		return listing.Page[MemberUsage]{}, fmt.Errorf("list role members: %w", err)
	}
	return result, nil
}

func (service *Service) Create(ctx context.Context, name string) (*domain.Role, error) {
	id, err := service.newID()
	if err != nil {
		return nil, fmt.Errorf("generate role ID: %w", err)
	}

	role, err := domain.NewRole(id, name, service.now())
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("create role: domain returned nil without error")
	}

	exists, err := service.repository.NameExists(ctx, domain.NormalizedNameKey(role.Name), "")
	if err != nil {
		return nil, fmt.Errorf("check role name: %w", err)
	}
	if exists {
		return nil, domain.ErrNameExists
	}

	if err := service.repository.Create(ctx, *role); err != nil {
		return nil, fmt.Errorf("create role: %w", err)
	}

	return role, nil
}

func (service *Service) Update(
	ctx context.Context,
	id string,
	name string,
) (*domain.Role, error) {
	role, err := service.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if role == nil {
		return nil, errors.New("update role: repository returned nil without error")
	}

	if err := role.Rename(name, service.now()); err != nil {
		return nil, err
	}

	exists, err := service.repository.NameExists(
		ctx,
		domain.NormalizedNameKey(role.Name),
		role.ID,
	)
	if err != nil {
		return nil, fmt.Errorf("check role name: %w", err)
	}
	if exists {
		return nil, domain.ErrNameExists
	}

	if err := service.repository.Update(ctx, *role); err != nil {
		return nil, fmt.Errorf("update role: %w", err)
	}

	return role, nil
}

func (service *Service) Delete(ctx context.Context, id string) error {
	if err := service.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete role: %w", err)
	}
	return nil
}
