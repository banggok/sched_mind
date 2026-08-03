package wbshttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	dependencydomain "github.com/banggok/sched_mind/backend/internal/dependencies/domain"
	"github.com/banggok/sched_mind/backend/internal/wbs/application"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
)

type reopenServiceStub struct {
	Service
	reopen     func(context.Context, string, string) (*domain.Node, error)
	executable func(context.Context, string, string, application.WriteExecutableInput) (*domain.Node, error)
	preview    func(context.Context, string, string, application.PreviewExecutableInput) (*application.SchedulePreview, error)
}

func (s reopenServiceStub) Reopen(ctx context.Context, projectID, id string) (*domain.Node, error) {
	return s.reopen(ctx, projectID, id)
}

func (s reopenServiceStub) UpdateExecutable(ctx context.Context, projectID, id string, input application.WriteExecutableInput) (*domain.Node, error) {
	if s.executable == nil {
		return nil, errors.New("unexpected executable call")
	}
	return s.executable(ctx, projectID, id, input)
}

func (s reopenServiceStub) PreviewExecutableSchedule(ctx context.Context, projectID, id string, input application.PreviewExecutableInput) (*application.SchedulePreview, error) {
	if s.preview == nil {
		return nil, errors.New("unexpected preview call")
	}
	return s.preview(ctx, projectID, id, input)
}

func reopenMux(service Service) http.Handler {
	mux := http.NewServeMux()
	New(service).Register(mux)
	return mux
}

func completedHTTPNode() *domain.Node {
	actualStart := time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC)
	actualEnd := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	executionStart := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	executionEnd := time.Date(2026, 7, 25, 0, 0, 0, 0, time.UTC)
	return &domain.Node{
		ID: "task", ProjectID: "project", Name: "Build API", Position: 1,
		Executable: domain.ExecutableFields{
			ExecutionTimeline:  domain.Timeline{Start: &executionStart, End: &executionEnd},
			CommitmentTimeline: domain.Timeline{Start: &executionStart, End: &executionEnd},
			ActualStart:        &actualStart,
			ActualEnd:          &actualEnd,
		},
		Children: []domain.Node{},
	}
}

func TestReopenEndpointAcceptsNoBodyAndEmptyObjectAndReturnsConfirmedNull_AC4(t *testing.T) {
	for _, body := range []string{"", "{}"} {
		t.Run("body="+body, func(t *testing.T) {
			calls := 0
			service := reopenServiceStub{reopen: func(_ context.Context, projectID, id string) (*domain.Node, error) {
				calls++
				if projectID != "project" || id != "task" {
					t.Fatalf("scope=%q/%q", projectID, id)
				}
				value := completedHTTPNode()
				value.Executable.ActualStart = nil
				value.Executable.ActualEnd = nil
				return value, nil
			}}
			request := httptest.NewRequest(http.MethodPost, "/api/projects/project/wbs/task/reopen", strings.NewReader(body))
			if body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()
			reopenMux(service).ServeHTTP(response, request)
			if response.Code != http.StatusOK || calls != 1 {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, calls, response.Body.String())
			}
			var payload map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			data, ok := payload["data"].(map[string]any)
			if !ok {
				t.Fatalf("data=%#v", payload["data"])
			}
			executable, ok := data["executable"].(map[string]any)
			if !ok {
				t.Fatalf("executable=%#v", data["executable"])
			}
			actual, exists := executable["actualEnd"]
			if !exists || actual != nil {
				t.Fatalf("actualEnd missing or non-null: exists=%v value=%#v body=%s", exists, actual, response.Body.String())
			}
			if data["id"] != "task" || data["name"] != "Build API" {
				t.Fatalf("confirmed identity changed: %#v", data)
			}
		})
	}
}

func TestReopenEndpointRejectsMalformedUnknownAndArbitraryPayload(t *testing.T) {
	for _, body := range []string{"{", `{"actualEnd":"2026-07-29"}`, `{"name":"Changed"}`, "null", "{} {}"} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			service := reopenServiceStub{reopen: func(context.Context, string, string) (*domain.Node, error) { calls++; return nil, nil }}
			request := httptest.NewRequest(http.MethodPost, "/api/projects/project/wbs/task/reopen", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			reopenMux(service).ServeHTTP(response, request)
			assertErrorCode(t, response, http.StatusBadRequest, "INVALID_REQUEST")
			if calls != 0 {
				t.Fatalf("reopen called %d times", calls)
			}
		})
	}
}

func TestReopenEndpointMapsStableSafeBusinessErrors_AC2_AC9_AC11_AC12_AC13(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{"not found", domain.ErrNotFound, 404, "TASK_NOT_FOUND"},
		{"group", domain.ErrExecutableOnly, 409, "EXECUTABLE_TASK_REQUIRED"},
		{"unfinished", domain.ErrTaskNotCompleted, 409, "TASK_NOT_COMPLETED"},
		{"closed", domain.ErrProjectClosedReadOnly, 409, "PROJECT_CLOSED_READ_ONLY"},
		{"conflict", domain.ErrTaskReopenConflict, 409, "TASK_REOPEN_CONFLICT"},
		{"forecast", errors.New("scheduler internal SQL secret"), 500, "TASK_REOPEN_FAILED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			service := reopenServiceStub{reopen: func(context.Context, string, string) (*domain.Node, error) { return nil, tc.err }}
			request := httptest.NewRequest(http.MethodPost, "/api/projects/project/wbs/task/reopen", nil)
			response := httptest.NewRecorder()
			reopenMux(service).ServeHTTP(response, request)
			assertErrorCode(t, response, tc.status, tc.code)
			if strings.Contains(response.Body.String(), "SQL") || strings.Contains(response.Body.String(), "scheduler") {
				t.Fatalf("infrastructure details leaked: %s", response.Body.String())
			}
		})
	}
}

func TestGenericExecutableUpdateCannotClearCompletedActualEnd_AC6(t *testing.T) {
	completed := completedHTTPNode()
	service := reopenServiceStub{
		reopen: func(context.Context, string, string) (*domain.Node, error) {
			return nil, errors.New("unexpected reopen")
		},
		executable: func(context.Context, string, string, application.WriteExecutableInput) (*domain.Node, error) {
			return nil, domain.ErrCompletedReadOnly
		},
	}
	body := bytes.NewBufferString(`{"name":"Changed","executionStart":"2026-07-20","executionEnd":"2026-07-25","commitmentStart":"2026-07-20","commitmentEnd":"2026-07-25"}`)
	request := httptest.NewRequest(http.MethodPut, "/api/projects/project/wbs/task/executable", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	reopenMux(service).ServeHTTP(response, request)
	assertErrorCode(t, response, http.StatusConflict, "COMPLETED_TASK_READ_ONLY")
	if completed.Executable.ActualEnd == nil {
		t.Fatal("test fixture lost Actual End")
	}
}

func TestGenericExecutableUpdateRejectsActualEndPayloadWithoutCallingMutation_AC6(t *testing.T) {
	calls := 0
	service := reopenServiceStub{
		reopen: func(context.Context, string, string) (*domain.Node, error) {
			return nil, errors.New("unexpected reopen")
		},
		executable: func(context.Context, string, string, application.WriteExecutableInput) (*domain.Node, error) {
			calls++
			return nil, errors.New("unexpected executable mutation")
		},
	}
	body := bytes.NewBufferString(`{"name":"Build API","actualEnd":null}`)
	request := httptest.NewRequest(http.MethodPut, "/api/projects/project/wbs/task/executable", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	reopenMux(service).ServeHTTP(response, request)
	assertErrorCode(t, response, http.StatusBadRequest, "INVALID_REQUEST")
	if calls != 0 {
		t.Fatalf("generic mutation called %d times", calls)
	}
}

func assertErrorCode(t *testing.T, response *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status=%d want=%d body=%s", response.Code, status, response.Body.String())
	}
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Code != code {
		t.Fatalf("code=%q want=%q body=%s", payload.Code, code, response.Body.String())
	}
}

func TestExecutableUpdateAcceptsLagContractAndReturnsLag_US6_AC2(t *testing.T) {
	captured := -1
	service := reopenServiceStub{
		reopen: func(context.Context, string, string) (*domain.Node, error) {
			return nil, errors.New("unexpected reopen")
		},
		executable: func(_ context.Context, projectID, id string, input application.WriteExecutableInput) (*domain.Node, error) {
			if projectID != "project" || id != "task" {
				t.Fatalf("scope = %s/%s", projectID, id)
			}
			captured = input.LagDays
			value := completedHTTPNode()
			value.Executable.ActualStart = nil
			value.Executable.ActualEnd = nil
			value.Executable.LagDays = input.LagDays
			return value, nil
		},
	}
	request := httptest.NewRequest(http.MethodPut, "/api/projects/project/wbs/task/executable", strings.NewReader(`{"lag":2}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	reopenMux(service).ServeHTTP(response, request)
	if response.Code != http.StatusOK || captured != 2 || !strings.Contains(response.Body.String(), `"lag":2`) {
		t.Fatalf("status=%d captured=%d body=%s", response.Code, captured, response.Body.String())
	}
}

func TestExecutableUpdateRejectsMalformedLagWithoutCallingMutation_US6_AC3(t *testing.T) {
	for _, body := range []string{`{"lag":-1}`, `{"lag":1.5}`, `{"lag":"1"}`, `{"lag":"1,5"}`} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			service := reopenServiceStub{
				reopen: func(context.Context, string, string) (*domain.Node, error) {
					return nil, errors.New("unexpected reopen")
				},
				executable: func(context.Context, string, string, application.WriteExecutableInput) (*domain.Node, error) {
					calls++
					return nil, errors.New("unexpected executable mutation")
				},
			}
			request := httptest.NewRequest(http.MethodPut, "/api/projects/project/wbs/task/executable", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			reopenMux(service).ServeHTTP(response, request)
			assertErrorCode(t, response, http.StatusBadRequest, "INVALID_LAG")
			if calls != 0 {
				t.Fatalf("mutation calls = %d", calls)
			}
		})
	}
}

func TestExecutableUpdateDistinguishesOmittedPercentageFromExplicitZero_US63_AC2_AC4(t *testing.T) {
	t.Run("omission reaches application as preserve existing", func(t *testing.T) {
		calls := 0
		service := reopenServiceStub{reopen: func(context.Context, string, string) (*domain.Node, error) {
			return nil, errors.New("unexpected reopen")
		}, executable: func(_ context.Context, _, _ string, input application.WriteExecutableInput) (*domain.Node, error) {
			calls++
			if input.CapacityAllocationPercentage != nil {
				t.Fatalf("omitted percentage=%v", *input.CapacityAllocationPercentage)
			}
			value := completedHTTPNode()
			value.Executable.ActualStart = nil
			value.Executable.ActualEnd = nil
			value.Executable.CapacityAllocationPercentage = 20
			return value, nil
		}}
		request := httptest.NewRequest(http.MethodPut, "/api/projects/project/wbs/task/executable", strings.NewReader(`{"lag":0}`))
		response := httptest.NewRecorder()
		reopenMux(service).ServeHTTP(response, request)
		if response.Code != http.StatusOK || calls != 1 || !strings.Contains(response.Body.String(), `"capacityAllocationPercentage":20`) {
			t.Fatalf("status=%d calls=%d body=%s", response.Code, calls, response.Body.String())
		}
	})
	t.Run("explicit zero is rejected before mutation", func(t *testing.T) {
		calls := 0
		service := reopenServiceStub{reopen: func(context.Context, string, string) (*domain.Node, error) {
			return nil, errors.New("unexpected reopen")
		}, executable: func(context.Context, string, string, application.WriteExecutableInput) (*domain.Node, error) {
			calls++
			return nil, errors.New("unexpected mutation")
		}}
		request := httptest.NewRequest(http.MethodPut, "/api/projects/project/wbs/task/executable", strings.NewReader(`{"capacityAllocationPercentage":0,"lag":0}`))
		response := httptest.NewRecorder()
		reopenMux(service).ServeHTTP(response, request)
		assertErrorCode(t, response, http.StatusUnprocessableEntity, "INVALID_TASK_CAPACITY_ALLOCATION")
		if calls != 0 {
			t.Fatalf("mutation calls=%d", calls)
		}
	})
}

func TestExecutablePreviewReturnsGeneratedDraftWithoutCallingConfirmedUpdate(t *testing.T) {
	previewCalls, updateCalls := 0, 0
	service := reopenServiceStub{
		reopen: func(context.Context, string, string) (*domain.Node, error) {
			return nil, errors.New("unexpected reopen")
		},
		executable: func(context.Context, string, string, application.WriteExecutableInput) (*domain.Node, error) {
			updateCalls++
			return nil, errors.New("unexpected confirmed update")
		},
		preview: func(_ context.Context, projectID, id string, input application.PreviewExecutableInput) (*application.SchedulePreview, error) {
			previewCalls++
			if projectID != "project" || id != "task" {
				t.Fatalf("scope=%q/%q", projectID, id)
			}
			if input.RoleID == nil || *input.RoleID != "role" || input.AssigneeID == nil || *input.AssigneeID != "member" || input.EffortMinutes == nil || *input.EffortMinutes != 480 || input.LagDays != 2 {
				t.Fatalf("input=%#v", input)
			}
			value := completedHTTPNode()
			value.Executable.ActualStart = nil
			value.Executable.ActualEnd = nil
			value.Executable.RoleID = input.RoleID
			value.Executable.AssigneeID = input.AssigneeID
			value.Executable.EffortMinutes = input.EffortMinutes
			value.Executable.LagDays = input.LagDays
			return &application.SchedulePreview{
				Task: value,
				Dependencies: dependencydomain.Detail{
					BlockedBy: []dependencydomain.Item{{
						Dependency: dependencydomain.Dependency{ID: "automatic-1", BlockingTaskID: "task-1", BlockedTaskID: "task", AutomaticOwned: true},
						Task:       dependencydomain.Task{ID: "task-1", Name: "Task 1", ProjectID: "project", ProjectName: "Alpha"},
					}},
					Blocks: []dependencydomain.Item{},
				},
			}, nil
		},
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/project/wbs/task/executable/preview",
		strings.NewReader(`{"roleId":"role","assigneeId":"member","effortHours":8,"lag":2}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	reopenMux(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK || previewCalls != 1 || updateCalls != 0 {
		t.Fatalf("status=%d preview=%d update=%d body=%s", response.Code, previewCalls, updateCalls, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"executionTimeline"`) || !strings.Contains(response.Body.String(), `"lag":2`) || !strings.Contains(response.Body.String(), `"source":"automatic"`) || !strings.Contains(response.Body.String(), `"name":"Task 1"`) {
		t.Fatalf("preview response=%s", response.Body.String())
	}
}

func TestExecutablePreviewAllowsClearedAssigneeForDependencyReconciliation(t *testing.T) {
	previewCalls := 0
	reason := "Task requires an Assignee before it can be scheduled."
	service := reopenServiceStub{
		reopen: func(context.Context, string, string) (*domain.Node, error) {
			return nil, errors.New("unexpected reopen")
		},
		preview: func(_ context.Context, projectID, id string, input application.PreviewExecutableInput) (*application.SchedulePreview, error) {
			previewCalls++
			if projectID != "project" || id != "task" || input.AssigneeID != nil || input.RoleID == nil || *input.RoleID != "role" || input.EffortMinutes == nil || *input.EffortMinutes != 480 || input.LagDays != 0 {
				t.Fatalf("input=%#v scope=%q/%q", input, projectID, id)
			}
			value := completedHTTPNode()
			value.Executable.ActualStart = nil
			value.Executable.ActualEnd = nil
			value.Executable.AssigneeID = nil
			value.Executable.ExecutionTimeline = domain.Timeline{}
			value.Executable.CommitmentTimeline = domain.Timeline{}
			value.Executable.ExecutionUnscheduledReason = &reason
			value.Executable.CommitmentUnscheduledReason = &reason
			return &application.SchedulePreview{
				Task:         value,
				Dependencies: dependencydomain.Detail{BlockedBy: []dependencydomain.Item{}, Blocks: []dependencydomain.Item{}},
			}, nil
		},
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/project/wbs/task/executable/preview",
		strings.NewReader(`{"roleId":"role","effortHours":8,"lag":0}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	reopenMux(service).ServeHTTP(response, request)

	if response.Code != http.StatusOK || previewCalls != 1 {
		t.Fatalf("status=%d preview=%d body=%s", response.Code, previewCalls, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), reason) || !strings.Contains(response.Body.String(), `"blockedBy":[]`) {
		t.Fatalf("cleared-assignee preview response=%s", response.Body.String())
	}
}

func TestExecutablePreviewRejectsUnknownOrMalformedDraftWithoutCallingService(t *testing.T) {
	for _, body := range []string{
		`{"name":"not-a-preview-field","roleId":"role","assigneeId":"member","effortHours":8,"lag":0}`,
		`{"roleId":"role","assigneeId":"member","effortHours":8,"lag":-1}`,
		`{"roleId":"role","assigneeId":"member","effortHours":8.333,"lag":0}`,
	} {
		t.Run(body, func(t *testing.T) {
			calls := 0
			service := reopenServiceStub{
				reopen: func(context.Context, string, string) (*domain.Node, error) {
					return nil, errors.New("unexpected reopen")
				},
				preview: func(context.Context, string, string, application.PreviewExecutableInput) (*application.SchedulePreview, error) {
					calls++
					return nil, errors.New("unexpected preview")
				},
			}
			request := httptest.NewRequest(http.MethodPost, "/api/projects/project/wbs/task/executable/preview", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			reopenMux(service).ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, calls, response.Body.String())
			}
		})
	}
}

func TestExecutablePreviewMapsIncompleteDraftToStableValidationError(t *testing.T) {
	service := reopenServiceStub{
		reopen: func(context.Context, string, string) (*domain.Node, error) {
			return nil, errors.New("unexpected reopen")
		},
		preview: func(context.Context, string, string, application.PreviewExecutableInput) (*application.SchedulePreview, error) {
			return nil, domain.ErrSchedulePreviewIncomplete
		},
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/project/wbs/task/executable/preview",
		strings.NewReader(`{"assigneeId":"member","effortHours":8,"lag":0}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	reopenMux(service).ServeHTTP(response, request)

	assertErrorCode(t, response, http.StatusBadRequest, "SCHEDULING_INPUT_INCOMPLETE")
}

func TestExecutablePreviewMapsUnavailableStateToStableConflict(t *testing.T) {
	service := reopenServiceStub{
		reopen: func(context.Context, string, string) (*domain.Node, error) {
			return nil, errors.New("unexpected reopen")
		},
		preview: func(context.Context, string, string, application.PreviewExecutableInput) (*application.SchedulePreview, error) {
			return nil, domain.ErrSchedulePreviewUnavailable
		},
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/project/wbs/task/executable/preview",
		strings.NewReader(`{"roleId":"role","assigneeId":"member","effortHours":8,"lag":0}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	reopenMux(service).ServeHTTP(response, request)

	assertErrorCode(t, response, http.StatusConflict, "SCHEDULE_PREVIEW_UNAVAILABLE")
}
