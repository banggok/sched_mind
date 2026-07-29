package dependencyhttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/banggok/sched_mind/backend/internal/dependencies/domain"
)

type serviceStub struct {
	createErr              error
	createdFrom, createdTo string
}

func (s *serviceStub) List(context.Context, string) (*domain.Detail, error) {
	return &domain.Detail{BlockedBy: []domain.Item{}, Blocks: []domain.Item{}}, nil
}
func (s *serviceStub) Candidates(_ context.Context, _ string, direction domain.Direction, _ string, _ int, _ int) (*domain.CandidatePage, error) {
	if !direction.Valid() {
		return nil, domain.ErrInvalidDirection
	}
	return &domain.CandidatePage{Items: []domain.Task{}, Page: 1, PageSize: 5}, nil
}
func (s *serviceStub) Create(_ context.Context, from, to string) (*domain.Dependency, error) {
	s.createdFrom, s.createdTo = from, to
	if s.createErr != nil {
		return nil, s.createErr
	}
	return &domain.Dependency{ID: "dependency", BlockingTaskID: from, BlockedTaskID: to}, nil
}
func (s *serviceStub) Delete(context.Context, string) error { return nil }
func testHandler(service Service) *http.ServeMux {
	mux := http.NewServeMux()
	New(service).Register(mux)
	return mux
}

func TestCreateDependency(t *testing.T) {
	service := &serviceStub{}
	request := httptest.NewRequest(http.MethodPost, "/api/dependencies", strings.NewReader(`{"blockingTaskId":"a","blockedTaskId":"b"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	testHandler(service).ServeHTTP(response, request)
	if response.Code != http.StatusCreated || service.createdFrom != "a" || service.createdTo != "b" {
		t.Fatalf("status=%d body=%s service=%#v", response.Code, response.Body.String(), service)
	}
}
func TestCreateMapsCycleWithSafePath(t *testing.T) {
	service := &serviceStub{createErr: &domain.CycleError{Path: []domain.CycleStep{{TaskID: "a", TaskName: "A", ProjectName: "Alpha"}, {TaskID: "b", TaskName: "B", ProjectName: "Beta"}, {TaskID: "a", TaskName: "A", ProjectName: "Alpha"}}}}
	request := httptest.NewRequest(http.MethodPost, "/api/dependencies", strings.NewReader(`{"blockingTaskId":"a","blockedTaskId":"b"}`))
	response := httptest.NewRecorder()
	testHandler(service).ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "DEPENDENCY_CYCLE_DETECTED") || !strings.Contains(response.Body.String(), `"taskName":"A"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
func TestCandidatesRejectInvalidDirection(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/dependency-candidates?taskId=a&direction=other", nil)
	testHandler(&serviceStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
func TestCreateRejectsUnknownField(t *testing.T) {
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/dependencies", strings.NewReader(`{"blockingTaskId":"a","blockedTaskId":"b","extra":true}`))
	testHandler(&serviceStub{}).ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
