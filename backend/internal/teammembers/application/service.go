package application

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/teammembers/domain"
)

type WriteInput struct {
	Name             string
	RoleID           string
	DailyCapacity    *float64
	BufferPercentage *float64
}

type Service struct {
	repository Repository
	now        func() time.Time
	newID      func() (string, error)
}

func NewService(repository Repository) *Service {
	return NewServiceWithDependencies(repository, time.Now, newUUID)
}

func NewServiceWithDependencies(
	repository Repository,
	now func() time.Time,
	newID func() (string, error),
) *Service {
	return &Service{repository: repository, now: now, newID: newID}
}

func (service *Service) List(ctx context.Context, query listing.Query) (listing.Page[TeamMemberRecord], error) {
	records, err := service.repository.List(ctx, query)
	if err != nil {
		return listing.Page[TeamMemberRecord]{}, fmt.Errorf("list team members: %w", err)
	}
	return records, nil
}

func (service *Service) Get(
	ctx context.Context,
	id string,
) (*TeamMemberRecord, error) {
	record, err := service.repository.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get team member: %w", err)
	}
	if record == nil {
		return nil, errors.New("get team member: repository returned nil without error")
	}
	return record, nil
}

func (service *Service) Create(
	ctx context.Context,
	input WriteInput,
) (*TeamMemberRecord, error) {
	dailyCapacity, buffer, err := capacities(input)
	if err != nil {
		return nil, err
	}
	if err := service.ensureRole(ctx, input.RoleID); err != nil {
		return nil, err
	}
	id, err := service.newID()
	if err != nil {
		return nil, fmt.Errorf("create team member ID: %w", err)
	}
	member, err := domain.NewTeamMember(
		id,
		input.Name,
		input.RoleID,
		dailyCapacity,
		buffer,
		service.now(),
	)
	if err != nil {
		return nil, err
	}
	if member == nil {
		return nil, errors.New("create team member: domain returned nil without error")
	}
	if err := service.repository.Create(ctx, *member); err != nil {
		return nil, fmt.Errorf("create team member: %w", err)
	}
	return service.Get(ctx, member.ID)
}

func (service *Service) Update(
	ctx context.Context,
	id string,
	input WriteInput,
) (*TeamMemberRecord, error) {
	dailyCapacity, buffer, err := capacities(input)
	if err != nil {
		return nil, err
	}
	if err := service.ensureRole(ctx, input.RoleID); err != nil {
		return nil, err
	}
	record, err := service.repository.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find team member for update: %w", err)
	}
	if record == nil {
		return nil, errors.New("update team member: repository returned nil without error")
	}
	if err := record.Member.Update(
		input.Name,
		input.RoleID,
		dailyCapacity,
		buffer,
		service.now(),
	); err != nil {
		return nil, err
	}
	if err := service.repository.Update(ctx, record.Member); err != nil {
		return nil, fmt.Errorf("update team member: %w", err)
	}
	return service.Get(ctx, id)
}

func (service *Service) Delete(ctx context.Context, id string) error {
	record, err := service.repository.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("find team member for delete: %w", err)
	}
	if record == nil {
		return errors.New("delete team member: repository returned nil without error")
	}
	if err := service.repository.DeleteIfNoActiveTask(ctx, id); err != nil {
		return fmt.Errorf("delete team member: %w", err)
	}
	return nil
}

func (service *Service) ensureRole(ctx context.Context, roleID string) error {
	roleID = strings.TrimSpace(roleID)
	if roleID == "" {
		return domain.ErrRoleRequired
	}
	exists, err := service.repository.RoleExists(ctx, roleID)
	if err != nil {
		return fmt.Errorf("check team member role: %w", err)
	}
	if !exists {
		return domain.ErrRoleNotFound
	}
	return nil
}

func capacities(
	input WriteInput,
) (domain.DailyCapacity, domain.BufferPercentage, error) {
	if input.DailyCapacity == nil {
		return domain.DailyCapacity{}, domain.BufferPercentage{},
			domain.ErrDailyCapacityRequired
	}
	dailyCapacity, err := domain.NewDailyCapacity(*input.DailyCapacity)
	if err != nil {
		return domain.DailyCapacity{}, domain.BufferPercentage{}, err
	}
	buffer := domain.DefaultBufferPercentage()
	if input.BufferPercentage != nil {
		buffer, err = domain.NewBufferPercentage(*input.BufferPercentage)
		if err != nil {
			return domain.DailyCapacity{}, domain.BufferPercentage{}, err
		}
	}
	return dailyCapacity, buffer, nil
}

func newUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", fmt.Errorf("generate UUID: %w", err)
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
