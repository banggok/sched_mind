package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/publicholidays/application"
	"github.com/banggok/sched_mind/backend/internal/publicholidays/infrastructure/gormrepo"
	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestAPIIntegrationCRUDFilterPaginationAndPersistence(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`CREATE TABLE public_holidays (id TEXT PRIMARY KEY, start_date DATE NOT NULL, end_date DATE NOT NULL, description VARCHAR(100) NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL); CREATE TABLE public_holiday_dates(public_holiday_id TEXT NOT NULL, date DATE NOT NULL UNIQUE, PRIMARY KEY(public_holiday_id,date))`).Error; err != nil {
		t.Fatal(err)
	}
	service := application.NewService(gormrepo.New(database))
	mux := http.NewServeMux()
	New(service).Register(mux)
	request := func(method, path, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		return response
	}
	created := request("POST", "/api/public-holidays", `{"startDate":"2026-08-17","endDate":"2026-08-18","description":" Independence Day "}`)
	if created.Code != 201 {
		t.Fatalf("create=%d %s", created.Code, created.Body.String())
	}
	var payload itemResponse
	if err := json.NewDecoder(created.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	id := payload.Data.ID
	filtered := request("GET", "/api/public-holidays?holidayDate=2026-08-17&page=1&pageSize=5", "")
	if filtered.Code != 200 || !strings.Contains(filtered.Body.String(), `"total":1`) {
		t.Fatalf("filtered=%d %s", filtered.Code, filtered.Body.String())
	}
	duplicate := request("POST", "/api/public-holidays", `{"startDate":"2026-08-17","endDate":"2026-08-17","description":"Duplicate"}`)
	if duplicate.Code != 409 {
		t.Fatalf("duplicate=%d %s", duplicate.Code, duplicate.Body.String())
	}
	updated := request("PUT", "/api/public-holidays/"+id, `{"startDate":"2026-08-19","endDate":"2026-08-21","description":"Updated"}`)
	if updated.Code != 200 || !strings.Contains(updated.Body.String(), `"startDate":"2026-08-19"`) {
		t.Fatalf("update=%d %s", updated.Code, updated.Body.String())
	}
	deleted := request("DELETE", "/api/public-holidays/"+id, "")
	if deleted.Code != 204 {
		t.Fatalf("delete=%d %s", deleted.Code, deleted.Body.String())
	}
	notFound := request("GET", "/api/public-holidays/"+id, "")
	if notFound.Code != 404 {
		t.Fatalf("notfound=%d %s", notFound.Code, notFound.Body.String())
	}
}

type reentrantScheduleAssertion struct {
	database *gorm.DB
	calls    int
}

func (scheduler *reentrantScheduleAssertion) RecalculateActiveProjects(ctx context.Context) error {
	if !sharedpersistence.ScheduleMutationSerialized(ctx) {
		return errors.New("scheduler did not inherit the schedule-mutation serialization marker")
	}

	nestedContext, release := sharedpersistence.SerializeScheduleMutation(ctx)
	defer release()
	transaction := sharedpersistence.Transaction(nestedContext, scheduler.database)

	var stored struct {
		Date time.Time `gorm:"column:date"`
	}
	if err := transaction.Table("public_holiday_dates").Select("date").Take(&stored).Error; err != nil {
		return err
	}
	if got := stored.Date.UTC().Format("2006-01-02"); got != "2026-08-03" {
		return errors.New("scheduler cannot observe the uncommitted public holiday date")
	}
	scheduler.calls++
	return nil
}

func TestAPIIntegrationCreateCompletesWhenSchedulerReentersSerializedMutation_DefectPublicHolidayPending(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`CREATE TABLE public_holidays (id TEXT PRIMARY KEY, start_date DATE NOT NULL, end_date DATE NOT NULL, description VARCHAR(100) NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL); CREATE TABLE public_holiday_dates(public_holiday_id TEXT NOT NULL, date DATE NOT NULL UNIQUE, PRIMARY KEY(public_holiday_id,date))`).Error; err != nil {
		t.Fatal(err)
	}

	repository := gormrepo.New(database)
	scheduler := &reentrantScheduleAssertion{database: database}
	service := application.NewServiceWithScheduler(repository, scheduler, func() time.Time {
		return time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	})
	mux := http.NewServeMux()
	New(service).Register(mux)

	response := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/api/public-holidays", strings.NewReader(`{"startDate":"2026-08-03","endDate":"2026-08-03","description":"test"}`))
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if scheduler.calls != 1 {
		t.Fatalf("scheduler calls=%d, want 1", scheduler.calls)
	}
}
