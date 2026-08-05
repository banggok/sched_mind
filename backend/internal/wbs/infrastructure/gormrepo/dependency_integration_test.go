package gormrepo

import (
	"context"
	"errors"
	"testing"
	"time"

	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func dependencyTestDB(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&projectModel{}, &nodeModel{}, &dependencyLinkModel{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE sprint_tasks (
		sprint_id TEXT NOT NULL,
		task_id TEXT NOT NULL REFERENCES wbs_nodes(id) ON DELETE CASCADE,
		PRIMARY KEY (sprint_id, task_id)
	)`).Error; err != nil {
		t.Fatal(err)
	}
	return New(db), db
}
func TestMoveConversionRetargetsDependenciesAndPreservesPercentage_US63_AC24(t *testing.T) {
	repo, db := dependencyTestDB(t)
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&projectModel{ID: "project", Status: "open"}).Error; err != nil {
		t.Fatal(err)
	}
	effort := 60
	nodes := []nodeModel{{ID: "blocker", ProjectID: "project", ParentKey: "", Name: "Blocker", NameKey: "blocker", Position: 1}, {ID: "destination", ProjectID: "project", ParentKey: "", Name: "Destination", NameKey: "destination", Position: 2, EffortMinutes: &effort, CapacityAllocationPercentage: 20}, {ID: "moving", ProjectID: "project", ParentKey: "", Name: "Moving", NameKey: "moving", Position: 3}}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&dependencyLinkModel{ID: "dependency", BlockingTaskID: "blocker", BlockedTaskID: "destination"}).Error; err != nil {
		t.Fatal(err)
	}
	parent := "destination"
	if err := repo.Move(context.Background(), "project", "moving", "converted", &parent, true, now, func(context.Context, string) error { return nil }, func(context.Context, []string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	var link dependencyLinkModel
	if err := db.First(&link, "id = ?", "dependency").Error; err != nil {
		t.Fatal(err)
	}
	if link.BlockedTaskID != "converted" {
		t.Fatalf("blocked task=%q", link.BlockedTaskID)
	}
	var group, converted nodeModel
	if err := db.First(&group, "id = ?", "destination").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.First(&converted, "id = ?", "converted").Error; err != nil {
		t.Fatal(err)
	}
	if group.CapacityAllocationPercentage != 100 || converted.CapacityAllocationPercentage != 20 {
		t.Fatalf("group=%d converted=%d", group.CapacityAllocationPercentage, converted.CapacityAllocationPercentage)
	}
}
func TestDeleteTaskCleansDependenciesAtomically(t *testing.T) {
	repo, db := dependencyTestDB(t)
	now := time.Now()
	if err := db.Create(&projectModel{ID: "project", Status: "open"}).Error; err != nil {
		t.Fatal(err)
	}
	nodes := []nodeModel{{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1}, {ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2}}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&dependencyLinkModel{ID: "ab", BlockingTaskID: "a", BlockedTaskID: "b"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO sprint_tasks (sprint_id, task_id) VALUES (?, ?), (?, ?)", "sprint-1", "a", "sprint-1", "b").Error; err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(context.Background(), "project", "b", now, func(context.Context, string) error { return nil }, func(context.Context, []string) error { return nil }); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&dependencyLinkModel{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("dependency count=%d", count)
	}
	if err := db.Table("sprint_tasks").Where("task_id = ?", "b").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("deleted Task Sprint relation count=%d", count)
	}
	if err := db.Table("sprint_tasks").Where("task_id = ?", "a").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("unrelated Task Sprint relation count=%d, want 1", count)
	}
}

func TestDeleteTaskRemovesNodeAndDependenciesBeforeAutomaticInvalidation(t *testing.T) {
	repo, db := dependencyTestDB(t)
	now := time.Date(2026, 7, 30, 18, 16, 0, 0, time.UTC)
	if err := db.Create(&projectModel{ID: "project", Status: "open", AutomaticScheduling: true}).Error; err != nil {
		t.Fatal(err)
	}
	nodes := []nodeModel{
		{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1},
		{ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2},
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&dependencyLinkModel{ID: "ab", BlockingTaskID: "a", BlockedTaskID: "b"}).Error; err != nil {
		t.Fatal(err)
	}

	invalidationCalls := 0
	scheduleCalls := 0
	invalidate := func(ctx context.Context, projectIDs []string) error {
		invalidationCalls++
		if len(projectIDs) != 1 || projectIDs[0] != "project" {
			t.Fatalf("invalidated projects=%v", projectIDs)
		}
		tx := sharedpersistence.Transaction(ctx, db)
		var nodeCount int64
		if err := tx.Model(&nodeModel{}).Where("id = ?", "b").Count(&nodeCount).Error; err != nil {
			return err
		}
		if nodeCount != 0 {
			return errors.New("deleted task remained visible during schedule invalidation")
		}
		var dependencyCount int64
		if err := tx.Model(&dependencyLinkModel{}).
			Where("blocking_task_id = ? OR blocked_task_id = ?", "b", "b").
			Count(&dependencyCount).Error; err != nil {
			return err
		}
		if dependencyCount != 0 {
			return errors.New("deleted task dependency remained visible during schedule invalidation")
		}
		return nil
	}
	schedule := func(context.Context, string) error {
		scheduleCalls++
		return nil
	}

	if err := repo.Delete(context.Background(), "project", "b", now, schedule, invalidate); err != nil {
		t.Fatal(err)
	}
	if invalidationCalls != 1 {
		t.Fatalf("invalidation calls=%d, want 1", invalidationCalls)
	}
	if scheduleCalls != 0 {
		t.Fatalf("project schedule calls=%d, want 0 because portfolio invalidation already includes the current project", scheduleCalls)
	}

	var nodeCount int64
	if err := db.Model(&nodeModel{}).Where("id = ?", "b").Count(&nodeCount).Error; err != nil {
		t.Fatal(err)
	}
	if nodeCount != 0 {
		t.Fatalf("deleted node count=%d", nodeCount)
	}
	var dependencyCount int64
	if err := db.Model(&dependencyLinkModel{}).Count(&dependencyCount).Error; err != nil {
		t.Fatal(err)
	}
	if dependencyCount != 0 {
		t.Fatalf("dependency count=%d", dependencyCount)
	}
}

func TestDeleteTaskRollsBackNodeAndDependenciesWhenInvalidationFails(t *testing.T) {
	repo, db := dependencyTestDB(t)
	now := time.Date(2026, 7, 30, 18, 16, 0, 0, time.UTC)
	if err := db.Create(&projectModel{ID: "project", Status: "open", AutomaticScheduling: true}).Error; err != nil {
		t.Fatal(err)
	}
	nodes := []nodeModel{
		{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1},
		{ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2},
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&dependencyLinkModel{ID: "ab", BlockingTaskID: "a", BlockedTaskID: "b"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO sprint_tasks (sprint_id, task_id) VALUES (?, ?)", "sprint-1", "b").Error; err != nil {
		t.Fatal(err)
	}

	invalidationFailure := errors.New("schedule invalidation failed")
	err := repo.Delete(
		context.Background(),
		"project",
		"b",
		now,
		func(context.Context, string) error { return nil },
		func(context.Context, []string) error { return invalidationFailure },
	)
	if !errors.Is(err, invalidationFailure) {
		t.Fatalf("delete error=%v, want invalidation failure", err)
	}

	var nodeCount int64
	if err := db.Model(&nodeModel{}).Where("id = ?", "b").Count(&nodeCount).Error; err != nil {
		t.Fatal(err)
	}
	if nodeCount != 1 {
		t.Fatalf("rolled-back node count=%d, want 1", nodeCount)
	}
	var dependencyCount int64
	if err := db.Model(&dependencyLinkModel{}).Where("id = ?", "ab").Count(&dependencyCount).Error; err != nil {
		t.Fatal(err)
	}
	if dependencyCount != 1 {
		t.Fatalf("rolled-back dependency count=%d, want 1", dependencyCount)
	}
	var sprintTaskCount int64
	if err := db.Table("sprint_tasks").Where("task_id = ?", "b").Count(&sprintTaskCount).Error; err != nil {
		t.Fatal(err)
	}
	if sprintTaskCount != 1 {
		t.Fatalf("rolled-back Sprint Task relation count=%d, want 1", sprintTaskCount)
	}
}
