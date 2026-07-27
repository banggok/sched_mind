package gormrepo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/projects/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testRepository(t *testing.T) *Repository {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&projectModel{}); err != nil {
		t.Fatal(err)
	}
	return New(database)
}

func TestRepositoryCreateListSearchAndPriority(t *testing.T) {
	repository := testRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	alpha, err := repository.CreateNext(ctx, "a", "Alpha", true, nil, 20, now)
	if err != nil || alpha == nil {
		t.Fatalf("alpha: %#v %v", alpha, err)
	}
	beta, err := repository.CreateNext(ctx, "b", "Beta", true, nil, 20, now)
	if err != nil || beta == nil || beta.Priority != 2 {
		t.Fatalf("beta: %#v %v", beta, err)
	}
	if _, err := repository.CreateNext(ctx, "duplicate", "ALPHA", true, nil, 20, now); !errors.Is(err, domain.ErrNameExists) {
		t.Fatalf("duplicate: %v", err)
	}
	result, err := repository.List(ctx, listing.Query{Search: "al", Page: 1, PageSize: 5})
	if err != nil || result.Total != 1 || result.Items[0].ID != "a" {
		t.Fatalf("list: %#v %v", result, err)
	}
	called := false
	moved, err := repository.MovePriority(ctx, "b", domain.PriorityUp, now.Add(time.Hour), func(context.Context) error { called = true; return nil })
	if err != nil || moved == nil || moved.Priority != 1 || !called {
		t.Fatalf("move: %#v %v called=%v", moved, err, called)
	}
	if _, err := repository.MovePriority(ctx, "b", domain.PriorityUp, now, func(context.Context) error { return nil }); !errors.Is(err, domain.ErrPriorityMoveNotAllowed) {
		t.Fatalf("boundary: %v", err)
	}
}

func TestRepositoryLifecycleDeleteAndRollback(t *testing.T) {
	repository := testRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	value, err := repository.CreateNext(ctx, "a", "Alpha", true, nil, 20, now)
	if err != nil || value == nil {
		t.Fatal(err)
	}
	if _, err := repository.ChangeStatus(ctx, "a", domain.StatusLocked, now.Add(time.Hour)); !errors.Is(err, domain.ErrCannotLockWithoutTasks) {
		t.Fatalf("lock zero tasks: %v", err)
	}
	if _, err := repository.ChangeStatus(ctx, "a", domain.StatusClosed, now); !errors.Is(err, domain.ErrCannotCloseWithoutTasks) {
		t.Fatalf("close zero: %v", err)
	}
	_, _ = repository.CreateNext(ctx, "b", "Beta", true, nil, 20, now)
	before, _ := repository.Find(ctx, "b")
	_, err = repository.MovePriority(ctx, "b", domain.PriorityUp, now, func(context.Context) error { return errors.New("scheduler unavailable") })
	if err == nil {
		t.Fatal("expected scheduler failure")
	}
	after, _ := repository.Find(ctx, "b")
	if before == nil || after == nil || before.Priority != after.Priority {
		t.Fatalf("rollback: %#v %#v", before, after)
	}
	if err := repository.DeleteChildless(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Find(ctx, "a"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted find: %v", err)
	}
}

func TestRepositorySettingsPersistenceAndSchedulerRollback(t *testing.T) {
	repository := testRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	created, err := repository.CreateNext(ctx, "a", "Alpha", true, nil, 20, now)
	if err != nil || created == nil || !created.AutomaticScheduling || created.ProjectBuffer != 20 {
		t.Fatalf("defaults: %#v %v", created, err)
	}
	anchor := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	updated, err := repository.UpdateSettings(ctx, "a", false, &anchor, 35, now.Add(time.Hour), func(context.Context, string) error { t.Fatal("scheduler called while disabling"); return nil })
	if err != nil || updated == nil || updated.AutomaticScheduling || updated.ProjectBuffer != 35 {
		t.Fatalf("disable: %#v %v", updated, err)
	}
	if updated.SchedulingStartDate == nil || !updated.SchedulingStartDate.Equal(anchor) {
		t.Fatalf("persist anchor: %#v", updated)
	}
	_, err = repository.UpdateSettings(ctx, "a", true, &anchor, 40, now.Add(2*time.Hour), func(context.Context, string) error { return errors.New("scheduler unavailable") })
	if err == nil {
		t.Fatal("expected scheduler failure")
	}
	stored, _ := repository.Find(ctx, "a")
	if stored == nil || stored.AutomaticScheduling || stored.ProjectBuffer != 35 || stored.SchedulingStartDate == nil || !stored.SchedulingStartDate.Equal(anchor) {
		t.Fatalf("rollback: %#v", stored)
	}
	_, err = repository.UpdateDetails(ctx, "a", "Renamed", true, &anchor, 40, now.Add(3*time.Hour), func(context.Context, string) error { return errors.New("scheduler unavailable") })
	if err == nil {
		t.Fatal("expected combined update failure")
	}
	stored, _ = repository.Find(ctx, "a")
	if stored == nil || stored.Name != "Alpha" || stored.AutomaticScheduling || stored.ProjectBuffer != 35 {
		t.Fatalf("combined rollback: %#v", stored)
	}
}

func TestRepositoryDoesNotScheduleWithoutProjectAnchor(t *testing.T) {
	repository := testRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	created, err := repository.CreateNext(ctx, "a", "Alpha", false, nil, 20, now)
	if err != nil || created == nil {
		t.Fatalf("create: %#v %v", created, err)
	}
	updated, err := repository.UpdateSettings(ctx, "a", true, nil, 20, now.Add(time.Hour), func(context.Context, string) error {
		t.Fatal("scheduler must not run without Scheduling Start Date")
		return nil
	})
	if err != nil || updated == nil || !updated.AutomaticScheduling || updated.SchedulingStartDate != nil {
		t.Fatalf("enable without anchor: %#v %v", updated, err)
	}
}
