package http

import (
	"context"
	"encoding/json"
	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/application"
	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type stub struct {
	created   application.WriteInput
	listCalls int
}

func (s *stub) List(context.Context, string, application.ListQuery) (listing.Page[domain.CapacityOverride], error) {
	s.listCalls++
	return listing.Page[domain.CapacityOverride]{Items: []domain.CapacityOverride{}, Page: 1, PageSize: 5}, nil
}
func (s *stub) Get(context.Context, string, string) (*domain.CapacityOverride, error) {
	return nil, domain.ErrNotFound
}
func (s *stub) Create(_ context.Context, member string, input application.WriteInput) (*domain.CapacityOverride, error) {
	s.created = input
	capacity, _ := domain.NewCapacity(*input.Capacity)
	return domain.New("id", member, input.Description, input.StartDate, input.EndDate, capacity, time.Now())
}
func (s *stub) Update(context.Context, string, string, application.WriteInput) (*domain.CapacityOverride, error) {
	return nil, domain.ErrNotFound
}
func (s *stub) Delete(context.Context, string, string) error { return domain.ErrNotFound }
func TestHandlerCreateAndValidation(t *testing.T) {
	service := &stub{}
	mux := http.NewServeMux()
	New(service).Register(mux)
	request := httptest.NewRequest("POST", "/api/team-members/member/capacity-overrides", strings.NewReader(`{"description":"Training","startDate":"2026-07-03","endDate":"2026-07-03","capacity":0}`))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != 201 {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if service.created.Description != "Training" || !strings.Contains(response.Body.String(), `"description":"Training"`) {
		t.Fatalf("created=%+v body=%s", service.created, response.Body.String())
	}
	request = httptest.NewRequest("POST", "/api/team-members/member/capacity-overrides", strings.NewReader(`{"description":" ","startDate":"2026-07-03","endDate":"2026-07-03","capacity":4}`))
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != 400 || !strings.Contains(response.Body.String(), "CAPACITY_OVERRIDE_DESCRIPTION_REQUIRED") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest("POST", "/api/team-members/member/capacity-overrides", strings.NewReader(`{"description":"Training","startDate":"2026-02-30","endDate":"2026-07-03","capacity":4}`))
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	var payload errorResponse
	_ = json.NewDecoder(response.Body).Decode(&payload)
	if response.Code != 400 || payload.Code != "CAPACITY_OVERRIDE_INVALID_DATE" || payload.Field != "startDate" {
		t.Fatalf("response=%+v status=%d", payload, response.Code)
	}
}
func TestHandlerRejectsUnknownFieldAndInvalidPagination(t *testing.T) {
	mux := http.NewServeMux()
	New(&stub{}).Register(mux)
	request := httptest.NewRequest("PUT", "/api/team-members/member/capacity-overrides/id", strings.NewReader(`{"startDate":"2026-07-03","endDate":"2026-07-03","capacity":4,"teamMemberId":"other"}`))
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != 400 {
		t.Fatalf("status=%d", response.Code)
	}
	request = httptest.NewRequest("GET", "/api/team-members/member/capacity-overrides?pageSize=101", nil)
	response = httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), "INVALID_PAGE_SIZE") {
		t.Fatal(response.Body.String())
	}
}

func TestHandlerParsesEffectiveDateAndRejectsInvalidValue(t *testing.T) {
	query, _, _, err := parseQuery(httptest.NewRequest(
		"GET",
		"/api/team-members/member/capacity-overrides?effectiveDate=2026-07-27&page=2&pageSize=5",
		nil,
	))
	if err != nil || query.EffectiveDate == nil ||
		query.EffectiveDate.Format("2006-01-02") != "2026-07-27" || query.Page != 2 {
		t.Fatalf("query=%+v err=%v", query, err)
	}
	_, code, field, err := parseQuery(httptest.NewRequest(
		"GET",
		"/api/team-members/member/capacity-overrides?effectiveDate=2026-02-30",
		nil,
	))
	if err == nil || code != "INVALID_EFFECTIVE_DATE" || field != "effectiveDate" {
		t.Fatalf("code=%s field=%s err=%v", code, field, err)
	}
	service := &stub{}
	mux := http.NewServeMux()
	New(service).Register(mux)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(
		"GET",
		"/api/team-members/member/capacity-overrides?effectiveDate=2026-02-30",
		nil,
	))
	if response.Code != http.StatusBadRequest || service.listCalls != 0 {
		t.Fatalf("status=%d listCalls=%d", response.Code, service.listCalls)
	}
}
