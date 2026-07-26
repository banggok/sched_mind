package http

import (
	"context"
	"github.com/banggok/sched_mind/backend/internal/publicholidays/application"
	"github.com/banggok/sched_mind/backend/internal/publicholidays/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type stub struct {
	input     application.WriteInput
	listCalls int
	err       error
}

func (s *stub) List(_ context.Context, q application.ListQuery) (listing.Page[domain.PublicHoliday], error) {
	s.listCalls++
	return listing.Page[domain.PublicHoliday]{Page: q.Page, PageSize: q.PageSize}, s.err
}
func (s *stub) Get(context.Context, string) (*domain.PublicHoliday, error) {
	return nil, domain.ErrNotFound
}
func (s *stub) Create(_ context.Context, input application.WriteInput) (*domain.PublicHoliday, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.input = input
	return domain.New("id", input.StartDate, input.EndDate, input.Description, time.Now())
}
func (s *stub) Update(context.Context, string, application.WriteInput) (*domain.PublicHoliday, error) {
	return nil, domain.ErrNotFound
}
func (s *stub) Delete(context.Context, string) error { return domain.ErrNotFound }
func (s *stub) CalendarDates(context.Context, time.Time, time.Time) ([]time.Time, error) {
	return []time.Time{}, s.err
}
func TestHandlerCreateValidationFilterAndErrors(t *testing.T) {
	service := &stub{}
	mux := http.NewServeMux()
	New(service).Register(mux)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("POST", "/api/public-holidays", strings.NewReader(`{"startDate":"2026-08-17","endDate":"2026-08-18","description":"Independence Day"}`)))
	if response.Code != 201 || !strings.Contains(response.Body.String(), `"startDate":"2026-08-17"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("GET", "/api/public-holidays?holidayDate=2026-02-30", nil))
	if response.Code != 400 || service.listCalls != 0 || !strings.Contains(response.Body.String(), "INVALID_HOLIDAY_DATE") {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, service.listCalls, response.Body.String())
	}
	service.err = domain.ErrDateAlreadyExists
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest("POST", "/api/public-holidays", strings.NewReader(`{"startDate":"2026-08-17","endDate":"2026-08-18","description":"Duplicate"}`)))
	if response.Code != 409 || !strings.Contains(response.Body.String(), "PUBLIC_HOLIDAY_DATE_ALREADY_EXISTS") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
