package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/shared/identity"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
	"github.com/banggok/sched_mind/backend/internal/teammembers/domain"
)

type WriteInput struct {
	Name             string
	RoleID           string
	DailyCapacity    *float64
	BufferPercentage *float64
}

type Scheduler interface {
	RecalculateMemberSchedule(context.Context, string) error
}

type scheduleAwareRepository interface {
	UpdateWithSchedule(context.Context, domain.TeamMember, bool, func(context.Context) error) error
}

type Service struct {
	repository Repository
	scheduler  Scheduler
	now        func() time.Time
	newID      func() (string, error)
}

func NewService(repository Repository) *Service {
	return NewServiceWithDependencies(repository, time.Now, identity.NewUUID)
}

func NewServiceWithScheduler(repository Repository, scheduler Scheduler) *Service {
	service := NewService(repository)
	service.scheduler = scheduler
	return service
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

func (service *Service) Get(ctx context.Context, id string) (*TeamMemberRecord, error) {
	record, err := service.repository.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get team member: %w", err)
	}
	if record == nil {
		return nil, errors.New("get team member: repository returned nil without error")
	}
	return record, nil
}

func (service *Service) Create(ctx context.Context, input WriteInput) (*TeamMemberRecord, error) {
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
	member, err := domain.NewTeamMember(id, input.Name, input.RoleID, dailyCapacity, buffer, service.now())
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

func (service *Service) Update(ctx context.Context, id string, input WriteInput) (*TeamMemberRecord, error) {
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
	capacityChanged := record.Member.DailyCapacity.Decimal() != dailyCapacity.Decimal() ||
		record.Member.BufferPercentage.Decimal() != buffer.Decimal()
	if err := record.Member.Update(input.Name, input.RoleID, dailyCapacity, buffer, service.now()); err != nil {
		return nil, err
	}

	ctx = schedulingimpact.WithOperation(ctx, "", schedulingimpact.ModeOrdinary)
	if repository, ok := service.repository.(scheduleAwareRepository); ok && service.scheduler != nil {
		err = repository.UpdateWithSchedule(ctx, record.Member, capacityChanged, func(txContext context.Context) error {
			return service.scheduler.RecalculateMemberSchedule(txContext, id)
		})
	} else {
		err = service.repository.Update(ctx, record.Member)
	}
	if err != nil {
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

func capacities(input WriteInput) (domain.DailyCapacity, domain.BufferPercentage, error) {
	if input.DailyCapacity == nil {
		return domain.DailyCapacity{}, domain.BufferPercentage{}, domain.ErrDailyCapacityRequired
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
