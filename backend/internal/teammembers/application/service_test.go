package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/teammembers/domain"
)

type deleteRepository struct {
	activeAssignment bool
	deleted          bool
}

func (repository *deleteRepository) List(context.Context, listing.Query) (listing.Page[TeamMemberRecord], error) {
	return listing.Page[TeamMemberRecord]{}, nil
}
func (repository *deleteRepository) FindByID(context.Context, string) (*TeamMemberRecord, error) {
	return &TeamMemberRecord{Member: domain.TeamMember{ID: "member"}}, nil
}
func (repository *deleteRepository) RoleExists(context.Context, string) (bool, error) {
	return true, nil
}
func (repository *deleteRepository) Create(context.Context, domain.TeamMember) error { return nil }
func (repository *deleteRepository) Update(context.Context, domain.TeamMember) error { return nil }
func (repository *deleteRepository) DeleteIfNoActiveTask(context.Context, string) error {
	if repository.activeAssignment {
		return domain.ErrAssignedToTask
	}
	repository.deleted = true
	return nil
}

func TestDeleteRejectsActiveTaskAssignment(t *testing.T) {
	repository := &deleteRepository{activeAssignment: true}
	err := NewService(repository).Delete(context.Background(), "member")
	if !errors.Is(err, domain.ErrAssignedToTask) {
		t.Fatalf("delete error = %v", err)
	}
	if repository.deleted {
		t.Fatal("repository delete called for member with active assignment")
	}
}

func TestDeleteAllowsMemberWithoutActiveTaskAssignment(t *testing.T) {
	repository := &deleteRepository{}
	if err := NewService(repository).Delete(context.Background(), "member"); err != nil {
		t.Fatal(err)
	}
	if !repository.deleted {
		t.Fatal("repository delete was not called")
	}
}

type schedulingRepository struct {
	deleteRepository
	record          TeamMemberRecord
	capacityChanged bool
	scheduleCalls   int
}

func (repository *schedulingRepository) FindByID(context.Context, string) (*TeamMemberRecord, error) {
	copy := repository.record
	return &copy, nil
}
func (repository *schedulingRepository) Update(_ context.Context, member domain.TeamMember) error {
	repository.record.Member = member
	return nil
}
func (repository *schedulingRepository) UpdateWithSchedule(
	ctx context.Context,
	member domain.TeamMember,
	capacityChanged bool,
	schedule func(context.Context) error,
) error {
	repository.capacityChanged = capacityChanged
	repository.record.Member = member
	if capacityChanged {
		repository.scheduleCalls++
		return schedule(ctx)
	}
	return nil
}

type memberScheduler struct{ calls int }

func (scheduler *memberScheduler) RecalculateMemberSchedule(context.Context, string) error {
	scheduler.calls++
	return nil
}

func TestUpdateRecalculatesOnlyWhenCapacityInputsChange(t *testing.T) {
	daily, err := domain.NewDailyCapacity(8)
	if err != nil {
		t.Fatal(err)
	}
	buffer, err := domain.NewBufferPercentage(20)
	if err != nil {
		t.Fatal(err)
	}
	member, err := domain.NewTeamMember("member", "Alice", "role", daily, buffer, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	repository := &schedulingRepository{record: TeamMemberRecord{Member: *member, RoleName: "Engineer"}}
	scheduler := &memberScheduler{}
	service := NewServiceWithScheduler(repository, scheduler)

	dailyValue, bufferValue := 8.0, 20.0
	if _, err := service.Update(context.Background(), "member", WriteInput{
		Name: "Alice Updated", RoleID: "role", DailyCapacity: &dailyValue, BufferPercentage: &bufferValue,
	}); err != nil {
		t.Fatal(err)
	}
	if repository.capacityChanged || scheduler.calls != 0 || repository.scheduleCalls != 0 {
		t.Fatalf("name-only update scheduled: changed=%v scheduler=%d repository=%d", repository.capacityChanged, scheduler.calls, repository.scheduleCalls)
	}

	dailyValue = 6
	if _, err := service.Update(context.Background(), "member", WriteInput{
		Name: "Alice Updated", RoleID: "role", DailyCapacity: &dailyValue, BufferPercentage: &bufferValue,
	}); err != nil {
		t.Fatal(err)
	}
	if !repository.capacityChanged || scheduler.calls != 1 || repository.scheduleCalls != 1 {
		t.Fatalf("capacity update did not schedule exactly once: changed=%v scheduler=%d repository=%d", repository.capacityChanged, scheduler.calls, repository.scheduleCalls)
	}
}
