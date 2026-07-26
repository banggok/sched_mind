package gormrepo

import (
	"context"
	"errors"
	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/application"
	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"sync"
	"testing"
	"time"
)

func setup(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&teamMemberModel{}, &capacityOverrideModel{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&teamMemberModel{ID: "member-a"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&teamMemberModel{ID: "member-b"}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestConcurrentOverlappingCreateStoresAtMostOne(t *testing.T) {
	database := setup(t)
	database.Exec("PRAGMA busy_timeout = 5000")
	repository := New(database)
	values := []domain.CapacityOverride{
		value(t, "concurrent-one", "member-a", "2026-08-01", "2026-08-03", 4),
		value(t, "concurrent-two", "member-a", "2026-08-02", "2026-08-04", 6),
	}
	start := make(chan struct{})
	var wait sync.WaitGroup
	wait.Add(2)
	for index := range values {
		go func(item domain.CapacityOverride) {
			defer wait.Done()
			<-start
			_ = repository.Create(context.Background(), item)
		}(values[index])
	}
	close(start)
	wait.Wait()
	var count int64
	if err := database.Model(&capacityOverrideModel{}).
		Where("team_member_id = ?", "member-a").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count > 1 {
		t.Fatalf("stored %d overlapping rows", count)
	}
}
func value(t *testing.T, id, member, start, end string, hours float64) domain.CapacityOverride {
	t.Helper()
	capacity, _ := domain.NewCapacity(hours)
	parse := func(v string) time.Time { r, _ := time.Parse("2006-01-02", v); return r }
	result, err := domain.New(id, member, parse(start), parse(end), capacity, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	return *result
}
func TestRepositoryScopePaginationOverlapUpdateDelete(t *testing.T) {
	database := setup(t)
	repo := New(database)
	ctx := context.Background()
	first := value(t, "one", "member-a", "2026-07-03", "2026-07-04", 4)
	if err := repo.Create(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, value(t, "other", "member-b", "2026-07-03", "2026-07-04", 4)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, value(t, "overlap", "member-a", "2026-07-04", "2026-07-05", 4)); !errors.Is(err, domain.ErrOverlaps) {
		t.Fatalf("overlap=%v", err)
	}
	adjacent := value(t, "adjacent", "member-a", "2026-07-05", "2026-07-06", 0)
	if err := repo.Create(ctx, adjacent); err != nil {
		t.Fatal(err)
	}
	page, err := repo.List(ctx, "member-a", application.ListQuery{Query: listing.Query{Page: 1, PageSize: 1}})
	if err != nil || page.Total != 2 || page.Items[0].ID != "one" {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	first.Capacity, _ = domain.NewCapacity(6)
	first.UpdatedAt = time.Now().Add(time.Hour)
	if err := repo.Update(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, "member-b", "one"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("mismatch=%v", err)
	}
	if err := repo.Delete(ctx, "member-a", "one"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Find(ctx, "member-a", "one"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("find=%v", err)
	}
	var deletedCount int64
	if err := database.Unscoped().Model(&capacityOverrideModel{}).
		Where("id = ?", "one").Count(&deletedCount).Error; err != nil {
		t.Fatal(err)
	}
	if deletedCount != 0 {
		t.Fatalf("direct delete retained %d capacity overrides", deletedCount)
	}
}

func TestRepositoryFiltersByInclusiveEffectiveDate(t *testing.T) {
	repository := New(setup(t))
	ctx := context.Background()
	for _, item := range []domain.CapacityOverride{
		value(t, "period", "member-a", "2026-07-26", "2026-07-28", 3),
		value(t, "other-member", "member-b", "2026-07-26", "2026-07-28", 5),
	} {
		if err := repository.Create(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	for _, selected := range []string{"2026-07-26", "2026-07-27", "2026-07-28"} {
		effectiveDate, _ := time.Parse("2006-01-02", selected)
		page, err := repository.List(ctx, "member-a", application.ListQuery{
			Query: listing.Query{Page: 1, PageSize: 5}, EffectiveDate: &effectiveDate,
		})
		if err != nil || page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != "period" {
			t.Fatalf("date=%s page=%+v err=%v", selected, page, err)
		}
	}
	for _, selected := range []string{"2026-07-25", "2026-07-29"} {
		effectiveDate, _ := time.Parse("2006-01-02", selected)
		page, err := repository.List(ctx, "member-a", application.ListQuery{
			Query: listing.Query{Page: 1, PageSize: 5}, EffectiveDate: &effectiveDate,
		})
		if err != nil || page.Total != 0 || len(page.Items) != 0 {
			t.Fatalf("date=%s page=%+v err=%v", selected, page, err)
		}
	}
}
