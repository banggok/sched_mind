package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/publicholidays/application"
	"github.com/banggok/sched_mind/backend/internal/publicholidays/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setup(t *testing.T) *Repository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&publicHolidayModel{}, &publicHolidayDateModel{}); err != nil {
		t.Fatal(err)
	}
	return New(db)
}
func holiday(t *testing.T, id, date, description string, updated time.Time) domain.PublicHoliday {
	t.Helper()
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil {
		t.Fatal(err)
	}
	value, err := domain.New(id, parsed, parsed, description, updated)
	if err != nil {
		t.Fatal(err)
	}
	return *value
}

func TestRepositoryCRUDFilterPaginationAndOrdering(t *testing.T) {
	repo := setup(t)
	ctx := context.Background()
	now := time.Now().UTC()
	for _, value := range []domain.PublicHoliday{holiday(t, "b", "2026-08-18", "Second", now), holiday(t, "a", "2026-08-17", "First", now)} {
		if err := repo.Create(ctx, value); err != nil {
			t.Fatal(err)
		}
	}
	page, err := repo.List(ctx, application.ListQuery{Query: listing.Query{Page: 1, PageSize: 1}})
	if err != nil || page.Total != 2 || len(page.Items) != 1 || page.Items[0].ID != "a" {
		t.Fatalf("page=%+v err=%v", page, err)
	}
	date, _ := time.Parse("2006-01-02", "2026-08-18")
	filtered, err := repo.List(ctx, application.ListQuery{Query: listing.Query{Page: 1, PageSize: 5}, HolidayDate: &date})
	if err != nil || filtered.Total != 1 || filtered.Items[0].ID != "b" {
		t.Fatalf("filtered=%+v err=%v", filtered, err)
	}
	found, err := repo.Find(ctx, "a")
	if err != nil || found == nil || found.StartDate.Format("2006-01-02") != "2026-08-17" {
		t.Fatalf("found=%+v err=%v", found, err)
	}
	found.Description = "Updated"
	found.UpdatedAt = now.Add(time.Hour)
	if err := repo.Update(ctx, *found); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Find(ctx, "a"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("find after delete=%v", err)
	}
}
func TestRepositoryEnforcesUniqueDateIncludingConcurrentCreate(t *testing.T) {
	repo := setup(t)
	ctx := context.Background()
	now := time.Now().UTC()
	first := holiday(t, "one", "2026-08-17", "One", now)
	second := holiday(t, "two", "2026-08-17", "Two", now)
	if err := repo.Create(ctx, first); err != nil {
		t.Fatal(err)
	}
	if err := repo.Create(ctx, second); !errors.Is(err, domain.ErrDateAlreadyExists) {
		t.Fatalf("duplicate=%v", err)
	}
	repo = setup(t)
	start := make(chan struct{})
	var wait sync.WaitGroup
	var successes int
	var lock sync.Mutex
	for _, value := range []domain.PublicHoliday{first, second} {
		wait.Add(1)
		go func(item domain.PublicHoliday) {
			defer wait.Done()
			<-start
			if repo.Create(ctx, item) == nil {
				lock.Lock()
				successes++
				lock.Unlock()
			}
		}(value)
	}
	close(start)
	wait.Wait()
	if successes > 1 {
		t.Fatalf("successes=%d", successes)
	}
}

func TestRepositoryListsCurrentBeforeExpiredThenByRange(t *testing.T) {
	repo := setup(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	for _, input := range []struct{ id, start, end string }{
		{"expired-later", "2026-01-05", "2026-01-06"},
		{"current-later", "2026-08-03", "2026-08-04"},
		{"current-first", "2026-07-27", "2026-07-28"},
		{"expired-first", "2025-12-01", "2025-12-02"},
	} {
		start, _ := time.Parse("2006-01-02", input.start)
		end, _ := time.Parse("2006-01-02", input.end)
		value, err := domain.New(input.id, start, end, input.id, now)
		if err != nil || value == nil {
			t.Fatalf("create fixture %s: value=%v err=%v", input.id, value, err)
		}
		if err := repo.Create(ctx, *value); err != nil {
			t.Fatal(err)
		}
	}
	page, err := repo.List(ctx, application.ListQuery{
		Query: listing.Query{Page: 1, PageSize: 5}, Today: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(page.Items))
	for _, item := range page.Items {
		got = append(got, item.ID)
	}
	want := []string{"current-first", "current-later", "expired-first", "expired-later"}
	if len(got) != len(want) {
		t.Fatalf("IDs=%v want=%v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("IDs=%v want=%v", got, want)
		}
	}
}

func TestScheduleAwareWritesPropagateSerializedTransactionContext_DefectNestedScheduleLock(t *testing.T) {
	repository := setup(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	value := holiday(t, "holiday", "2026-08-03", "Holiday", now)

	assertSchedulerContext := func(wantHolidayCount, wantDateCount int64, wantDescription string) func(context.Context) error {
		return func(scheduleContext context.Context) error {
			if !sharedpersistence.ScheduleMutationSerialized(scheduleContext) {
				return errors.New("scheduler callback lost the schedule-mutation serialization marker")
			}

			nestedContext, release := sharedpersistence.SerializeScheduleMutation(scheduleContext)
			defer release()
			transaction := sharedpersistence.Transaction(nestedContext, repository.database)

			var holidayCount int64
			if err := transaction.Model(&publicHolidayModel{}).Where("id = ?", value.ID).Count(&holidayCount).Error; err != nil {
				return err
			}
			if holidayCount != wantHolidayCount {
				return fmt.Errorf("holiday count inside scheduler transaction = %d, want %d", holidayCount, wantHolidayCount)
			}

			var dateCount int64
			if err := transaction.Model(&publicHolidayDateModel{}).Where("public_holiday_id = ?", value.ID).Count(&dateCount).Error; err != nil {
				return err
			}
			if dateCount != wantDateCount {
				return fmt.Errorf("holiday date count inside scheduler transaction = %d, want %d", dateCount, wantDateCount)
			}

			if wantHolidayCount == 1 {
				var stored publicHolidayModel
				if err := transaction.First(&stored, "id = ?", value.ID).Error; err != nil {
					return err
				}
				if stored.Description != wantDescription {
					return fmt.Errorf("description inside scheduler transaction = %q, want %q", stored.Description, wantDescription)
				}
			}
			return nil
		}
	}

	if err := repository.CreateWithSchedule(ctx, value, assertSchedulerContext(1, 1, "Holiday")); err != nil {
		t.Fatalf("create with schedule: %v", err)
	}

	value.Description = "Updated"
	value.UpdatedAt = now.Add(time.Hour)
	if err := repository.UpdateWithSchedule(ctx, value, true, assertSchedulerContext(1, 1, "Updated")); err != nil {
		t.Fatalf("update with schedule: %v", err)
	}

	if err := repository.DeleteWithSchedule(ctx, value.ID, assertSchedulerContext(0, 0, "")); err != nil {
		t.Fatalf("delete with schedule: %v", err)
	}
}
