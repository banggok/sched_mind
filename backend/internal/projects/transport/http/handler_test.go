package projecthttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/projects/application"
	"github.com/banggok/sched_mind/backend/internal/projects/domain"
	"github.com/banggok/sched_mind/backend/internal/projects/infrastructure/gormrepo"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testHandler(t *testing.T) http.Handler {
	handler, _ := testHandlerWithDatabase(t)
	return handler
}

func testHandlerWithDatabase(t *testing.T) (http.Handler, *gorm.DB) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	// Isolated test schema is intentionally ORM-owned; production uses migration DDL.
	if err := database.Exec("CREATE TABLE projects (id TEXT PRIMARY KEY, name TEXT NOT NULL, name_key TEXT NOT NULL UNIQUE, status TEXT NOT NULL, start_date DATETIME, end_date DATETIME, auto_calculate_date NUMERIC NOT NULL, automatic_scheduling NUMERIC NOT NULL DEFAULT 1, scheduling_start_date DATE, project_buffer INTEGER NOT NULL DEFAULT 20, schedule_version INTEGER NOT NULL DEFAULT 0, priority INTEGER NOT NULL UNIQUE, closed_at DATETIME, locked_execution_snapshot TEXT, locked_commitment_snapshot TEXT, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	service := application.NewService(gormrepo.New(database), application.NoopScheduler{})
	mux := http.NewServeMux()
	New(service).Register(mux)
	return mux, database
}
func TestProjectHTTPFlowAndValidation(t *testing.T) {
	handler := testHandler(t)
	response := request(handler, "POST", "/api/projects", `{"name":" Alpha "}`)
	if response.Code != 201 || !strings.Contains(response.Body.String(), `"status":"open"`) || !strings.Contains(response.Body.String(), `"projectPriority":1`) {
		t.Fatalf("create %d %s", response.Code, response.Body.String())
	}
	response = request(handler, "POST", "/api/projects", `{"name":"Gamma","schedulingStartDate":"2026-08-03"}`)
	if response.Code != 201 || !strings.Contains(response.Body.String(), `"schedulingStartDate":"2026-08-03"`) {
		t.Fatalf("scheduling anchor: %d %s", response.Code, response.Body.String())
	}
	if response := request(handler, "POST", "/api/projects", `{"name":"Invalid","schedulingStartDate":"03-08-2026"}`); response.Code != 400 || !strings.Contains(response.Body.String(), "PROJECT_SCHEDULING_START_DATE_INVALID") {
		t.Fatalf("invalid scheduling anchor: %d %s", response.Code, response.Body.String())
	}
	if response := request(handler, "POST", "/api/projects", `{"name":"Beta","status":"closed"}`); response.Code != 400 {
		t.Fatalf("unknown field: %d", response.Code)
	}
	if response := request(handler, "GET", "/api/projects?search=al&page=1&pageSize=5", ""); response.Code != 200 || !strings.Contains(response.Body.String(), `"total":1`) {
		t.Fatalf("list: %d %s", response.Code, response.Body.String())
	}
	if response := request(handler, "PATCH", "/api/projects/"+projectID(responseBody(request(handler, "GET", "/api/projects?search=al&page=1&pageSize=5", "")))+"/settings", `{"automaticScheduling":false,"projectBuffer":35}`); response.Code != 200 || !strings.Contains(response.Body.String(), `"projectBuffer":35`) {
		t.Fatalf("settings: %d %s", response.Code, response.Body.String())
	}
}

func TestProjectHTTPAllowsLockedNameOnlyRenameAndRejectsSettingsMutation_US31_AC11_US62_AC16A(t *testing.T) {
	handler, database := testHandlerWithDatabase(t)
	created := request(handler, "POST", "/api/projects", `{"name":"Alpha","automaticScheduling":false,"schedulingStartDate":"2026-08-03","projectBuffer":35}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", created.Code, created.Body.String())
	}
	id := projectID(request(handler, "GET", "/api/projects?search=Alpha&page=1&pageSize=5", "").Body.String())
	if id == "" {
		t.Fatal("created project ID is empty")
	}

	// The HTTP fixture has no WBS rows, so seed the Locked state directly. The
	// scenario under test is the update contract, not lifecycle eligibility.
	if err := database.Exec("UPDATE projects SET status = ?, schedule_version = ? WHERE id = ?", string(domain.StatusLocked), 7, id).Error; err != nil {
		t.Fatal(err)
	}

	renamed := request(handler, "PUT", "/api/projects/"+id, `{"name":" Renamed "}`)
	if renamed.Code != http.StatusOK || !strings.Contains(renamed.Body.String(), `"name":"Renamed"`) ||
		!strings.Contains(renamed.Body.String(), `"status":"locked"`) ||
		!strings.Contains(renamed.Body.String(), `"automaticScheduling":false`) ||
		!strings.Contains(renamed.Body.String(), `"schedulingStartDate":"2026-08-03"`) ||
		!strings.Contains(renamed.Body.String(), `"projectBuffer":35`) ||
		!strings.Contains(renamed.Body.String(), `"scheduleVersion":7`) {
		t.Fatalf("name-only rename: %d %s", renamed.Code, renamed.Body.String())
	}

	mixed := request(handler, "PUT", "/api/projects/"+id, `{"name":"Rejected","automaticScheduling":true,"schedulingStartDate":null,"projectBuffer":20}`)
	if mixed.Code != http.StatusConflict || !strings.Contains(mixed.Body.String(), "PROJECT_LOCKED_READ_ONLY") {
		t.Fatalf("mixed locked update: %d %s", mixed.Code, mixed.Body.String())
	}
	after := request(handler, "GET", "/api/projects/"+id, "")
	if after.Code != http.StatusOK || !strings.Contains(after.Body.String(), `"name":"Renamed"`) ||
		!strings.Contains(after.Body.String(), `"automaticScheduling":false`) ||
		!strings.Contains(after.Body.String(), `"projectBuffer":35`) ||
		!strings.Contains(after.Body.String(), `"scheduleVersion":7`) {
		t.Fatalf("mixed update changed state: %d %s", after.Code, after.Body.String())
	}

	partial := request(handler, "PUT", "/api/projects/"+id, `{"name":"Rejected","projectBuffer":20}`)
	if partial.Code != http.StatusBadRequest || !strings.Contains(partial.Body.String(), "INVALID_REQUEST") {
		t.Fatalf("partial settings update: %d %s", partial.Code, partial.Body.String())
	}
}

func responseBody(recorder *httptest.ResponseRecorder) string { return recorder.Body.String() }
func projectID(body string) string {
	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(body), &payload)
	if len(payload.Data) == 0 {
		return ""
	}
	return payload.Data[0].ID
}
func TestProjectHTTPErrorMapping(t *testing.T) {
	service := &errorService{}
	mux := http.NewServeMux()
	New(service).Register(mux)
	if response := request(mux, "POST", "/api/projects/id/status", `{"status":"bad"}`); response.Code != 400 || !strings.Contains(response.Body.String(), "PROJECT_STATUS_INVALID") {
		t.Fatalf("status: %d %s", response.Code, response.Body.String())
	}
	if response := request(mux, "POST", "/api/projects/id/priority", `{"direction":"left"}`); response.Code != 400 || !strings.Contains(response.Body.String(), "PROJECT_PRIORITY_DIRECTION_INVALID") {
		t.Fatalf("priority: %d %s", response.Code, response.Body.String())
	}
	if response := request(mux, "PATCH", "/api/projects/id/settings", `{"automaticScheduling":true,"projectBuffer":20}`); response.Code != 409 || !strings.Contains(response.Body.String(), "PROJECT_SETTINGS_READ_ONLY") {
		t.Fatalf("read-only settings: %d %s", response.Code, response.Body.String())
	}
	recorder := httptest.NewRecorder()
	writeError(recorder, domain.ErrCannotLockWithoutTasks)
	if recorder.Code != 409 || !strings.Contains(recorder.Body.String(), "PROJECT_CANNOT_LOCK_WITHOUT_TASKS") {
		t.Fatalf("zero-task lock: %d %s", recorder.Code, recorder.Body.String())
	}
}
func request(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

type errorService struct{}

func (*errorService) List(context.Context, listing.Query) (listing.Page[domain.Project], error) {
	return listing.Page[domain.Project]{}, errors.New("unused")
}
func (*errorService) Get(context.Context, string) (*domain.Project, error) {
	return nil, domain.ErrNotFound
}
func (*errorService) Create(context.Context, string, bool, *time.Time, int) (*domain.Project, error) {
	return nil, errors.New("unused")
}
func (*errorService) Update(context.Context, string, string, bool, *time.Time, int) (*domain.Project, error) {
	return nil, errors.New("unused")
}
func (*errorService) Rename(context.Context, string, string) (*domain.Project, error) {
	return nil, errors.New("unused")
}
func (*errorService) ChangeStatus(context.Context, string, domain.Status) (*domain.Project, error) {
	return nil, errors.New("unused")
}
func (*errorService) BulkReopen(context.Context, string, string) ([]domain.Project, error) {
	return nil, errors.New("unused")
}
func (*errorService) MovePriority(context.Context, string, domain.PriorityDirection) (*domain.Project, error) {
	return nil, errors.New("unused")
}
func (*errorService) Delete(context.Context, string) error { return errors.New("unused") }
func (*errorService) UpdateSettings(context.Context, string, bool, *time.Time, int) (*domain.Project, error) {
	return nil, domain.ErrSettingsReadOnly
}
