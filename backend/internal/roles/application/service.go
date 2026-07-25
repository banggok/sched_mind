package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/banggok/sched_mind/backend/internal/roles/domain"
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
		newID:      newUUID,
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
	role, err := service.repository.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New("delete role: repository returned nil without error")
	}

	inUse, err := service.repository.IsInUse(ctx, id)
	if err != nil {
		return fmt.Errorf("check role usage: %w", err)
	}
	if inUse {
		return domain.ErrInUse
	}

	if err := service.repository.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete role: %w", err)
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

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		value[0:4],
		value[4:6],
		value[6:8],
		value[8:10],
		value[10:16],
	), nil
}
