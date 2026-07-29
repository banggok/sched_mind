package gormrepo

import (
	"context"
	"testing"
	"time"

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
	return New(db), db
}
func TestMoveConversionRetargetsDependencies(t *testing.T) {
	repo, db := dependencyTestDB(t)
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	if err := db.Create(&projectModel{ID: "project", Status: "open"}).Error; err != nil {
		t.Fatal(err)
	}
	effort := 60
	nodes := []nodeModel{{ID: "blocker", ProjectID: "project", ParentKey: "", Name: "Blocker", NameKey: "blocker", Position: 1}, {ID: "destination", ProjectID: "project", ParentKey: "", Name: "Destination", NameKey: "destination", Position: 2, EffortMinutes: &effort}, {ID: "moving", ProjectID: "project", ParentKey: "", Name: "Moving", NameKey: "moving", Position: 3}}
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
}
