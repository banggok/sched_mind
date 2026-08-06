package wbshttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/wbs/application"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
)

type reopenServiceStub struct {
	Service
	reopen     func(context.Context, string, string) (*domain.Node, error)
	executable func(context.Context, string, string, application.WriteExecutableInput) (*domain.Node, error)
	preview    func(context.Context, string, string, application.PreviewExecutableInput) (*application.SchedulePreview, error)
	recommend  func(context.Context, string, string, schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error)
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

func (s reopenServiceStub) RecommendAssignees(ctx context.Context, projectID, id string, input schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error) {
	if s.recommend == nil {
		return nil, errors.New("unexpected recommendation call")
	}
	return s.recommend(ctx, projectID, id, input)
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
	if !strings.Contains(response.Body.String(), `"executionTimeline"`) || !strings.Contains(response.Body.String(), `"lag":2`) {
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
				Task: value,
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
	if !strings.Contains(response.Body.String(), reason) {
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

func TestAssigneeRecommendationEndpointMapsOneBatchAndExactDraft_D01_D03_D06_D15_D17_AC3_AC11_AC12_AC25_AC33(t *testing.T) {
	calls := 0
	finish := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	reason := "NO_POSITIVE_CAPACITY"
	service := reopenServiceStub{recommend: func(_ context.Context, projectID, taskID string, input schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error) {
		calls++
		if projectID != "project" || taskID != "task" {
			t.Fatalf("scope=%s/%s", projectID, taskID)
		}
		if input.RoleID != "role" || input.EffortMinutes != 480 || input.LagDays != 2 || input.CapacityAllocationPercentage != 40 {
			t.Fatalf("input=%#v", input)
		}
		if input.ExecutionStart == nil || input.ExecutionStart.Format("2006-01-02") != "2026-08-10" {
			t.Fatalf("execution start=%v", input.ExecutionStart)
		}
		return &schedulingdomain.AssigneeRecommendationResult{
			CalculatedOnDate:        time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC),
			ProjectScheduleVersions: map[string]int64{"project": 12},
			Mode:                    schedulingdomain.RecommendationManualAdvisory,
			Items: []schedulingdomain.AssigneeRecommendationItem{
				{
					MemberID: "member", MemberName: "Dewi", RoleID: "role",
					RankGroup: schedulingdomain.RecommendationFeasible, ExecutionEnd: &finish,
					RemainingExecutionCapacityMinutes: 240,
				},
				{
					MemberID: "none", MemberName: "Gabby", RoleID: "role",
					RankGroup: schedulingdomain.RecommendationNoCompletion, ReasonCode: &reason,
				},
			},
		}, nil
	}}
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/project/wbs/task/assignee-recommendations",
		strings.NewReader(`{"roleId":"role","effortHours":8,"capacityAllocationPercentage":40,"lag":2,"executionStart":"2026-08-10","executionEnd":"2026-12-31"}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	reopenMux(service).ServeHTTP(response, request)
	if response.Code != http.StatusOK || calls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, calls, response.Body.String())
	}
	var payload struct {
		Data assigneeRecommendationResultItem `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Mode != "manual-advisory" || payload.Data.CalculatedOnDate != "2026-08-05" || payload.Data.Snapshot.ProjectScheduleVersions["project"] != 12 {
		t.Fatalf("metadata=%#v", payload.Data)
	}
	if len(payload.Data.Items) != 2 || payload.Data.Items[0].RemainingExecutionCapacityHours != 4 || payload.Data.Items[1].ReasonCode == nil || *payload.Data.Items[1].ReasonCode != reason {
		t.Fatalf("items=%#v", payload.Data.Items)
	}
}

func TestAssigneeRecommendationEndpointRejectsIncompleteOrBrowserToday_D03_D17_AC3_AC34(t *testing.T) {
	for name, body := range map[string]string{
		"missing percentage": `{"roleId":"role","effortHours":8,"lag":0}`,
		"invalid effort":     `{"roleId":"role","effortHours":0.25,"capacityAllocationPercentage":40,"lag":0}`,
		"invalid end":        `{"roleId":"role","effortHours":8,"capacityAllocationPercentage":40,"lag":0,"executionEnd":"not-a-date"}`,
		"browser today":      `{"roleId":"role","effortHours":8,"capacityAllocationPercentage":40,"lag":0,"today":"2026-08-05"}`,
	} {
		t.Run(name, func(t *testing.T) {
			calls := 0
			service := reopenServiceStub{recommend: func(context.Context, string, string, schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error) {
				calls++
				return nil, nil
			}}
			request := httptest.NewRequest(http.MethodPost, "/api/projects/project/wbs/task/assignee-recommendations", strings.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			reopenMux(service).ServeHTTP(response, request)
			if response.Code != http.StatusBadRequest || calls != 0 {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, calls, response.Body.String())
			}
			if name != "browser today" && !strings.Contains(response.Body.String(), "ASSIGNEE_RECOMMENDATION_INPUT_INVALID") {
				t.Fatalf("body=%s", response.Body.String())
			}
			if name == "browser today" && !strings.Contains(response.Body.String(), "INVALID_REQUEST") {
				t.Fatalf("body=%s", response.Body.String())
			}
		})
	}
}

func TestAssigneeRecommendationEndpointMapsStableErrors_D17_AC31(t *testing.T) {
	cases := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{name: "not recommendable", err: schedulingdomain.ErrTaskNotRecommendable, status: http.StatusConflict, code: "TASK_NOT_RECOMMENDABLE"},
		{name: "stale", err: schedulingdomain.ErrAssigneeRecommendationStale, status: http.StatusConflict, code: "ASSIGNEE_RECOMMENDATION_STALE"},
		{name: "unavailable", err: fmt.Errorf("%w: database secret", schedulingdomain.ErrAssigneeRecommendationUnavailable), status: http.StatusServiceUnavailable, code: "ASSIGNEE_RECOMMENDATION_UNAVAILABLE"},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			service := reopenServiceStub{recommend: func(context.Context, string, string, schedulingdomain.AssigneeRecommendationInput) (*schedulingdomain.AssigneeRecommendationResult, error) {
				return nil, testCase.err
			}}
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/projects/project/wbs/task/assignee-recommendations",
				strings.NewReader(`{"roleId":"role","effortHours":8,"capacityAllocationPercentage":40,"lag":0}`),
			)
			response := httptest.NewRecorder()
			reopenMux(service).ServeHTTP(response, request)
			if response.Code != testCase.status || !strings.Contains(response.Body.String(), testCase.code) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
			if testCase.code == "ASSIGNEE_RECOMMENDATION_UNAVAILABLE" && strings.Contains(response.Body.String(), "database secret") {
				t.Fatalf("unsafe infrastructure detail body=%s", response.Body.String())
			}
		})
	}
}

type structuralServiceStub struct {
	Service
	create        func(context.Context, string, *string, string, bool) (*domain.Node, error)
	createSibling func(context.Context, string, string, string) (*domain.Node, error)
	reorder       func(context.Context, string, string, domain.Direction) error
	place         func(context.Context, string, string, string, domain.Placement) error
}

func (s structuralServiceStub) Create(ctx context.Context, projectID string, parentID *string, name string, confirm bool) (*domain.Node, error) {
	if s.create == nil {
		return nil, errors.New("unexpected create call")
	}
	return s.create(ctx, projectID, parentID, name, confirm)
}

func (s structuralServiceStub) CreateSibling(ctx context.Context, projectID, insertAfterID, name string) (*domain.Node, error) {
	if s.createSibling == nil {
		return nil, errors.New("unexpected create sibling call")
	}
	return s.createSibling(ctx, projectID, insertAfterID, name)
}

func (s structuralServiceStub) Reorder(ctx context.Context, projectID, id string, direction domain.Direction) error {
	if s.reorder == nil {
		return errors.New("unexpected adjacent reorder call")
	}
	return s.reorder(ctx, projectID, id, direction)
}

func (s structuralServiceStub) Place(ctx context.Context, projectID, id, targetID string, placement domain.Placement) error {
	if s.place == nil {
		return errors.New("unexpected target placement call")
	}
	return s.place(ctx, projectID, id, targetID, placement)
}

func TestCreateSiblingEndpointUsesInsertAfterIntentWithoutClientParentOrPosition_DeltaD03(t *testing.T) {
	calls := 0
	service := structuralServiceStub{createSibling: func(_ context.Context, projectID, insertAfterID, name string) (*domain.Node, error) {
		calls++
		if projectID != "project" || insertAfterID != "anchor" || name != "New sibling" {
			t.Fatalf("scope=%q anchor=%q name=%q", projectID, insertAfterID, name)
		}
		parentID := "group"
		return &domain.Node{ID: "new", ProjectID: projectID, ParentID: &parentID, Name: name, Position: 4, Children: []domain.Node{}}, nil
	}}
	request := httptest.NewRequest(http.MethodPost, "/api/projects/project/wbs", strings.NewReader(`{"name":"New sibling","insertAfterWbsId":"anchor"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	reopenMux(service).ServeHTTP(response, request)

	if response.Code != http.StatusCreated || calls != 1 {
		t.Fatalf("status=%d calls=%d body=%s", response.Code, calls, response.Body.String())
	}
	var payload struct {
		Data struct {
			ID       string  `json:"id"`
			ParentID *string `json:"parentId"`
			Position int     `json:"position"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.ID != "new" || payload.Data.ParentID == nil || *payload.Data.ParentID != "group" || payload.Data.Position != 4 {
		t.Fatalf("confirmed response=%#v", payload.Data)
	}
}

func TestCreateSiblingEndpointRejectsConflictingStructuralIntent_DeltaD03(t *testing.T) {
	calls := 0
	service := structuralServiceStub{
		create: func(context.Context, string, *string, string, bool) (*domain.Node, error) {
			calls++
			return nil, errors.New("unexpected create")
		},
		createSibling: func(context.Context, string, string, string) (*domain.Node, error) {
			calls++
			return nil, errors.New("unexpected create sibling")
		},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/projects/project/wbs", strings.NewReader(`{"name":"Invalid","parentId":"group","insertAfterWbsId":"anchor"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	reopenMux(service).ServeHTTP(response, request)

	assertErrorCode(t, response, http.StatusBadRequest, "WBS_CREATE_POSITION_INVALID")
	if calls != 0 {
		t.Fatalf("mutation calls=%d", calls)
	}
}

func TestReorderEndpointDispatchesTargetBeforeAfterAndKeepsAdjacentFallback_DeltaD05(t *testing.T) {
	placeCalls := 0
	reorderCalls := 0
	service := structuralServiceStub{
		place: func(_ context.Context, projectID, id, targetID string, placement domain.Placement) error {
			placeCalls++
			if projectID != "project" || id != "source" || targetID != "target" || placement != domain.PlaceBefore {
				t.Fatalf("place scope=%q source=%q target=%q placement=%q", projectID, id, targetID, placement)
			}
			return nil
		},
		reorder: func(_ context.Context, projectID, id string, direction domain.Direction) error {
			reorderCalls++
			if projectID != "project" || id != "source" || direction != domain.MoveDown {
				t.Fatalf("reorder scope=%q source=%q direction=%q", projectID, id, direction)
			}
			return nil
		},
	}
	for _, test := range []struct {
		name string
		body string
	}{
		{name: "target placement", body: `{"targetSiblingId":"target","placement":"before"}`},
		{name: "adjacent fallback", body: `{"direction":"down"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/projects/project/wbs/source/reorder", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			reopenMux(service).ServeHTTP(response, request)
			if response.Code != http.StatusNoContent {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
	if placeCalls != 1 || reorderCalls != 1 {
		t.Fatalf("place calls=%d reorder calls=%d", placeCalls, reorderCalls)
	}
}

func TestReorderEndpointRejectsMixedOrIncompletePlacementIntent_DeltaD05_D09(t *testing.T) {
	calls := 0
	service := structuralServiceStub{
		place:   func(context.Context, string, string, string, domain.Placement) error { calls++; return nil },
		reorder: func(context.Context, string, string, domain.Direction) error { calls++; return nil },
	}
	for _, body := range []string{
		`{"direction":"up","targetSiblingId":"target","placement":"before"}`,
		`{"placement":"after"}`,
		`{"targetSiblingId":"target"}`,
	} {
		request := httptest.NewRequest(http.MethodPost, "/api/projects/project/wbs/source/reorder", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		reopenMux(service).ServeHTTP(response, request)
		assertErrorCode(t, response, http.StatusConflict, "WBS_REORDER_TARGET_INVALID")
	}
	if calls != 0 {
		t.Fatalf("mutation calls=%d", calls)
	}
}

func TestCreateSiblingEndpointMapsStaleAnchorAsRecoverableConflict_DeltaD03_D09(t *testing.T) {
	service := structuralServiceStub{createSibling: func(context.Context, string, string, string) (*domain.Node, error) {
		return nil, domain.ErrCreateAnchorConflict
	}}
	request := httptest.NewRequest(http.MethodPost, "/api/projects/project/wbs", strings.NewReader(`{"name":"New sibling","insertAfterWbsId":"deleted-anchor"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	reopenMux(service).ServeHTTP(response, request)

	assertErrorCode(t, response, http.StatusConflict, "WBS_CREATE_ANCHOR_CONFLICT")
}
