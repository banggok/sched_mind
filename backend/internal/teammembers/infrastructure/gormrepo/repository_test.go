package gormrepo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/teammembers/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestDeleteSoftDeletesMemberAndCapacityOverrides(t *testing.T) {
	database := openTestDatabase(t)
	now := time.Now().UTC()
	if err := database.Create(&roleModel{ID: "role", Name: "Backend"}).Error; err != nil {
		t.Fatal(err)
	}
	member := teamMemberModel{
		ID: "member", Name: "Harry", RoleID: "role", DailyCapacity: "8.0",
		BufferPercentage: "20.0", CreatedAt: now, UpdatedAt: now,
	}
	if err := database.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&capacityOverrideModel{ID: "override", TeamMemberID: member.ID}).Error; err != nil {
		t.Fatal(err)
	}

	if err := New(database).DeleteIfNoActiveTask(context.Background(), member.ID); err != nil {
		t.Fatal(err)
	}

	var activeMembers, activeOverrides int64
	if err := database.Model(&teamMemberModel{}).Count(&activeMembers).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&capacityOverrideModel{}).Count(&activeOverrides).Error; err != nil {
		t.Fatal(err)
	}
	if activeMembers != 0 || activeOverrides != 0 {
		t.Fatalf("active members=%d overrides=%d", activeMembers, activeOverrides)
	}
	var deletedMember teamMemberModel
	if err := database.Unscoped().First(&deletedMember, "id = ?", member.ID).Error; err != nil {
		t.Fatal(err)
	}
	var deletedOverride capacityOverrideModel
	if err := database.Unscoped().First(&deletedOverride, "id = ?", "override").Error; err != nil {
		t.Fatal(err)
	}
	if !deletedMember.DeletedAt.Valid || !deletedOverride.DeletedAt.Valid {
		t.Fatal("member and override must retain soft-delete timestamps")
	}
	if _, err := New(database).FindByID(context.Background(), member.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("find deleted member error = %v", err)
	}
}

func TestDeleteRejectsActiveAssignmentProjection(t *testing.T) {
	database := openTestDatabase(t)
	now := time.Now().UTC()
	if err := database.Create(&roleModel{ID: "role", Name: "Backend"}).Error; err != nil {
		t.Fatal(err)
	}
	member := teamMemberModel{
		ID: "member", Name: "Harry", RoleID: "role", DailyCapacity: "8.0",
		BufferPercentage: "20.0", CreatedAt: now, UpdatedAt: now,
	}
	if err := database.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&assignmentProjectModel{ID: "project", Status: "open"}).Error; err != nil {
		t.Fatal(err)
	}
	memberID := "member"
	if err := database.Create(&assignmentWBSNodeModel{ID: "task", ProjectID: "project", AssigneeID: &memberID}).Error; err != nil {
		t.Fatal(err)
	}

	err := New(database).DeleteIfNoActiveTask(context.Background(), "member")
	if !errors.Is(err, domain.ErrAssignedToTask) {
		t.Fatalf("delete error = %v", err)
	}
	var active int64
	if err := database.Model(&teamMemberModel{}).
		Where("id = ?", "member").Count(&active).Error; err != nil {
		t.Fatal(err)
	}
	if active != 1 {
		t.Fatal("active assignment rejection must retain member")
	}
}

func TestDeleteAllowsAssignmentOnlyInClosedProject(t *testing.T) {
	database := openTestDatabase(t)
	now := time.Now().UTC()
	if err := database.Create(&roleModel{ID: "role", Name: "Backend"}).Error; err != nil {
		t.Fatal(err)
	}
	member := teamMemberModel{ID: "member", Name: "Harry", RoleID: "role", DailyCapacity: "8.0", BufferPercentage: "20.0", CreatedAt: now, UpdatedAt: now}
	if err := database.Create(&member).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&assignmentProjectModel{ID: "closed-project", Status: "closed"}).Error; err != nil {
		t.Fatal(err)
	}
	memberID := member.ID
	if err := database.Create(&assignmentWBSNodeModel{ID: "historical-task", ProjectID: "closed-project", AssigneeID: &memberID}).Error; err != nil {
		t.Fatal(err)
	}
	if err := New(database).DeleteIfNoActiveTask(context.Background(), member.ID); err != nil {
		t.Fatal(err)
	}
}

func openTestDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := MigrateTestSchema(database); err != nil {
		t.Fatal(err)
	}
	return database
}
