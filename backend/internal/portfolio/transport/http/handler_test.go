package portfoliohttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/portfolio/application"
	"github.com/banggok/sched_mind/backend/internal/portfolio/domain"
)

type serviceStub struct {
	query application.PortfolioQuery
}

func (service *serviceStub) ActiveProjects(context.Context) ([]domain.ProjectOption, error) {
	return []domain.ProjectOption{{ID: "project", Name: "Project", Status: "open", ScheduleVersion: 4}}, nil
}
func (service *serviceStub) Portfolio(_ context.Context, query application.PortfolioQuery) (*domain.Portfolio, error) {
	service.query = query
	start := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	roleID := "testing"
	roleName := "Testing"
	return &domain.Portfolio{
		Projection:       query.Projection,
		Projects:         []domain.ProjectOption{{ID: "project", Name: "Project", Status: "open", ScheduleVersion: 4}},
		Rows:             []domain.Row{{ID: "task", ProjectID: "project", Kind: "task", Name: "Task", WBSNumber: "1", RoleID: &roleID, RoleName: &roleName, Start: &start, End: &end, Completed: true}},
		WorkingDayAnchor: &start,
	}, nil
}
func (service *serviceStub) ListSavedFilters(context.Context) ([]domain.SavedFilter, error) {
	return nil, nil
}
func (service *serviceStub) CreateSavedFilter(context.Context, string, []string) (*domain.SavedFilter, error) {
	return nil, domain.ErrFilterNameExists
}
func (service *serviceStub) UpdateSavedFilter(context.Context, string, int64, []string) (*domain.SavedFilter, error) {
	return nil, domain.ErrFilterConflict
}
func (service *serviceStub) DeleteSavedFilter(context.Context, string, int64) error {
	return domain.ErrFilterConflict
}

func TestPortfolioEndpointReturnsOneBoundedProjectionPayload(t *testing.T) {
	service := &serviceStub{}
	mux := http.NewServeMux()
	New(service).Register(mux)
	request := httptest.NewRequest(http.MethodGet, "/api/portfolio?projection=execution&from=2026-08-01&to=2026-08-31&projectId=beta&projectId=alpha", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", response.Code, response.Body.String())
	}
	if len(service.query.ProjectIDs) != 2 || service.query.ProjectIDs[0] != "beta" || service.query.ProjectIDs[1] != "alpha" {
		t.Fatalf("project IDs = %#v", service.query.ProjectIDs)
	}
	var payload struct {
		Data struct {
			Projection string `json:"projection"`
			Rows       []struct {
				ID        string `json:"id"`
				RoleID    string `json:"roleId"`
				RoleName  string `json:"roleName"`
				Completed bool   `json:"completed"`
				Start     string `json:"start"`
			} `json:"rows"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Projection != "execution" || len(payload.Data.Rows) != 1 || payload.Data.Rows[0].RoleID != "testing" || payload.Data.Rows[0].RoleName != "Testing" || !payload.Data.Rows[0].Completed || payload.Data.Rows[0].Start != "2026-08-03" {
		t.Fatalf("payload = %#v", payload)
	}
}

func TestSavedFilterConflictUsesObservableOptimisticConcurrencyError(t *testing.T) {
	mux := http.NewServeMux()
	New(&serviceStub{}).Register(mux)
	request := httptest.NewRequest(http.MethodPut, "/api/portfolio/saved-filters/filter", strings.NewReader(`{"version":1,"projectIds":["project"]}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "SAVED_FILTER_CONFLICT") {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}
