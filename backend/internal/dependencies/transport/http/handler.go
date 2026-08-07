package dependencyhttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/banggok/sched_mind/backend/internal/dependencies/domain"
	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/httpjson"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
)

type Service interface {
	List(context.Context, string) (*domain.Detail, error)
	Candidates(context.Context, string, domain.Direction, string, int, int) (*domain.CandidatePage, error)
	Create(context.Context, string, string) (*domain.Dependency, error)
	Delete(context.Context, string) error
}
type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }
func (h *Handler) Register(m *http.ServeMux) {
	m.HandleFunc("GET /api/tasks/{taskId}/dependencies", h.list)
	m.HandleFunc("GET /api/dependency-candidates", h.candidates)
	m.HandleFunc("POST /api/dependencies", h.create)
	m.HandleFunc("DELETE /api/dependencies/{dependencyId}", h.delete)
}

type response struct {
	Data any `json:"data"`
}
type createRequest struct {
	BlockingTaskID string `json:"blockingTaskId"`
	BlockedTaskID  string `json:"blockedTaskId"`
}
type dependencyResponse struct {
	ID             string `json:"id"`
	BlockingTaskID string `json:"blockingTaskId"`
	BlockedTaskID  string `json:"blockedTaskId"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
}
type taskItem struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ProjectID     string  `json:"projectId"`
	ProjectName   string  `json:"projectName"`
	HierarchyPath string  `json:"hierarchyPath"`
	Completed     bool    `json:"completed"`
	ExpectedStart *string `json:"expectedStart"`
}
type relationItem struct {
	ID   string   `json:"id"`
	Task taskItem `json:"task"`
}
type detailResponse struct {
	BlockedBy []relationItem `json:"blockedBy"`
	Blocks    []relationItem `json:"blocks"`
}
type pageResponse struct {
	Items      []taskItem `json:"items"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	TotalItems int64      `json:"totalItems"`
}
type errorResponse struct {
	Code    string        `json:"code"`
	Message string        `json:"message"`
	Details *errorDetails `json:"details,omitempty"`
}
type errorDetails struct {
	Path []cycleItem `json:"path"`
}
type cycleItem struct {
	TaskID      string `json:"taskId"`
	TaskName    string `json:"taskName"`
	ProjectName string `json:"projectName"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.List(r.Context(), r.PathValue("taskId"))
	if e != nil {
		writeError(w, e)
		return
	}
	httpjson.Write(w, 200, response{detail(v)})
}

func (h *Handler) candidates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("taskId") == "" {
		writeInvalid(w, "INVALID_REQUEST", "taskId is required")
		return
	}
	page, e := positive(q.Get("page"), 1)
	if e != nil {
		writeInvalid(w, "INVALID_PAGE", "page must be positive")
		return
	}
	size, e := positive(q.Get("pageSize"), 5)
	if e != nil || size > 100 {
		writeInvalid(w, "INVALID_PAGE_SIZE", "pageSize must be between 1 and 100")
		return
	}
	v, e := h.service.Candidates(r.Context(), q.Get("taskId"), domain.Direction(q.Get("direction")), q.Get("search"), page, size)
	if e != nil {
		writeError(w, e)
		return
	}
	items := make([]taskItem, 0, len(v.Items))
	for _, t := range v.Items {
		items = append(items, mapTask(t))
	}
	httpjson.Write(w, 200, response{pageResponse{Items: items, Page: v.Page, PageSize: v.PageSize, TotalItems: v.TotalItems}})
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var p createRequest
	if !decode(w, r, &p) {
		return
	}
	if p.BlockingTaskID == "" || p.BlockedTaskID == "" {
		writeInvalid(w, "INVALID_REQUEST", "blockingTaskId and blockedTaskId are required")
		return
	}
	v, e := h.service.Create(r.Context(), p.BlockingTaskID, p.BlockedTaskID)
	if e != nil {
		writeError(w, e)
		return
	}
	httpjson.Write(w, 201, response{mapDependency(*v)})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if e := h.service.Delete(r.Context(), r.PathValue("dependencyId")); e != nil {
		writeError(w, e)
		return
	}
	w.WriteHeader(204)
}

func mapDependency(value domain.Dependency) dependencyResponse {
	return dependencyResponse{
		ID:             value.ID,
		BlockingTaskID: value.BlockingTaskID,
		BlockedTaskID:  value.BlockedTaskID,
		CreatedAt:      value.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:      value.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func detail(v *domain.Detail) detailResponse {
	return detailResponse{BlockedBy: relations(v.BlockedBy), Blocks: relations(v.Blocks)}
}

func relations(values []domain.Item) []relationItem {
	out := make([]relationItem, 0, len(values))
	for _, v := range values {
		out = append(out, relationItem{ID: v.Dependency.ID, Task: mapTask(v.Task)})
	}
	return out
}

func mapTask(v domain.Task) taskItem {
	var expected *string
	if v.ExpectedStart != nil {
		s := v.ExpectedStart.Format("2006-01-02")
		expected = &s
	}
	return taskItem{
		ID:            v.ID,
		Name:          v.Name,
		ProjectID:     v.ProjectID,
		ProjectName:   v.ProjectName,
		HierarchyPath: v.HierarchyPath,
		Completed:     v.ActualStart != nil && v.ActualEnd != nil,
		ExpectedStart: expected,
	}
}

func positive(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	v, e := strconv.Atoi(raw)
	if e != nil || v < 1 {
		return 0, errors.New("invalid positive integer")
	}
	return v, nil
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(target); e != nil {
		writeInvalid(w, "INVALID_REQUEST", "Request body is invalid")
		return false
	}
	var extra any
	if e := d.Decode(&extra); !errors.Is(e, io.EOF) {
		writeInvalid(w, "INVALID_REQUEST", "Request body is invalid")
		return false
	}
	return true
}

func writeInvalid(w http.ResponseWriter, code, message string) {
	httpjson.Write(w, 400, errorResponse{Code: code, Message: message})
}

func writeError(w http.ResponseWriter, err error) {
	if schedulingimpact.WriteHTTPError(w, err) {
		return
	}
	status, code, message := 500, "INTERNAL_ERROR", "The request could not be completed."
	switch {
	case errors.Is(err, domain.ErrTaskNotFound):
		status, code, message = 404, "TASK_NOT_FOUND", "Task was not found."
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = 404, "DEPENDENCY_NOT_FOUND", "Dependency was not found."
	case errors.Is(err, domain.ErrSelfReference):
		status, code, message = 409, "DEPENDENCY_SELF_REFERENCE", "A task cannot block itself."
	case errors.Is(err, domain.ErrAlreadyExists):
		status, code, message = 409, "DEPENDENCY_ALREADY_EXISTS", "This dependency already exists."
	case errors.Is(err, domain.ErrExecutableNeeded):
		status, code, message = 409, "DEPENDENCY_EXECUTABLE_TASK_REQUIRED", "Dependencies require tasks, not groups."
	case errors.Is(err, domain.ErrCycle):
		status, code, message = 409, "DEPENDENCY_CYCLE_DETECTED", "Dependency cannot be added because it creates a cycle."
	case errors.Is(err, domain.ErrClosedProject):
		status, code, message = 409, "DEPENDENCY_CLOSED_PROJECT_TASK_NOT_ALLOWED", "Closed project tasks cannot be used for a new dependency."
	case errors.Is(err, domain.ErrLockedProject):
		status, code, message = 409, "PROJECT_LOCKED_READ_ONLY", "Locked project dependencies are read-only."
	case errors.Is(err, domain.ErrLockedGroup):
		status, code, message = 409, "GROUP_LOCKED_READ_ONLY", "Locked Group dependencies are read-only."
	case errors.Is(err, domain.ErrCompletedBlocked):
		status, code, message = 409, "DEPENDENCY_COMPLETED_TASK_CANNOT_BE_BLOCKED", "A completed task cannot be blocked by a new dependency."
	case errors.Is(err, domain.ErrCompletedHistory):
		status, code, message = 409, "DEPENDENCY_COMPLETED_HISTORY_READ_ONLY", "This historical dependency is read-only."
	case errors.Is(err, domain.ErrInvalidDirection):
		status, code, message = 400, "INVALID_DEPENDENCY_DIRECTION", "Dependency direction is invalid."
	case errors.Is(err, schedulingdomain.ErrConcurrentConflict):
		status, code, message = 409, "SCHEDULING_CONFLICT", "The schedule changed concurrently. Refresh and try again."
	case errors.Is(err, schedulingdomain.ErrDataIntegrity):
		status, code, message = 409, "SCHEDULING_DATA_INTEGRITY_CONFLICT", "The portfolio schedule is inconsistent and was not changed."
	}
	res := errorResponse{Code: code, Message: message}
	var cycle *domain.CycleError
	if errors.As(err, &cycle) {
		path := make([]cycleItem, 0, len(cycle.Path))
		for _, v := range cycle.Path {
			path = append(path, cycleItem{TaskID: v.TaskID, TaskName: v.TaskName, ProjectName: v.ProjectName})
		}
		res.Details = &errorDetails{Path: path}
	}
	if status == http.StatusInternalServerError {
		log.Printf("dependency request failed: %v", err)
	}
	httpjson.Write(w, status, res)
}
