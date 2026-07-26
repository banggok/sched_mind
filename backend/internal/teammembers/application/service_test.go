package application

import (
	"context"
	"errors"
	"testing"

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
