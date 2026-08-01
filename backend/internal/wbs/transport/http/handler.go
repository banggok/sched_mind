package wbshttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	dependencydomain "github.com/banggok/sched_mind/backend/internal/dependencies/domain"
	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/httpjson"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
	"github.com/banggok/sched_mind/backend/internal/wbs/application"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
)

type Service interface {
	Tree(context.Context, string) ([]domain.Node, error)
	Get(context.Context, string, string) (*domain.Node, error)
	Allocations(context.Context, string, string) (*application.AllocationGroups, error)
	Create(context.Context, string, *string, string, bool) (*domain.Node, error)
	Rename(context.Context, string, string, string) (*domain.Node, error)
	UpdateExecutable(context.Context, string, string, application.WriteExecutableInput) (*domain.Node, error)
	PreviewExecutableSchedule(context.Context, string, string, application.PreviewExecutableInput) (*application.SchedulePreview, error)
	Complete(context.Context, string, string, time.Time, time.Time) (*domain.Node, error)
	Reopen(context.Context, string, string) (*domain.Node, error)
	Reorder(context.Context, string, string, domain.Direction) error
	Move(context.Context, string, string, *string, bool) error
	Delete(context.Context, string, string) error
}
type Handler struct{ service Service }

func New(s Service) *Handler { return &Handler{service: s} }

type writeRequest struct {
	Name              string  `json:"name"`
	ParentID          *string `json:"parentId"`
	ConfirmConversion bool    `json:"confirmConversion"`
}
type executableRequest struct {
	Name            *string         `json:"name"`
	RoleID          *string         `json:"roleId"`
	AssigneeID      *string         `json:"assigneeId"`
	EffortHours     *float64        `json:"effortHours"`
	LagDays         json.RawMessage `json:"lag"`
	ExecutionStart  *string         `json:"executionStart"`
	ExecutionEnd    *string         `json:"executionEnd"`
	CommitmentStart *string         `json:"commitmentStart"`
	CommitmentEnd   *string         `json:"commitmentEnd"`
}
type previewExecutableRequest struct {
	RoleID      *string         `json:"roleId"`
	AssigneeID  *string         `json:"assigneeId"`
	EffortHours *float64        `json:"effortHours"`
	LagDays     json.RawMessage `json:"lag"`
}

type commandRequest struct {
	Direction         domain.Direction `json:"direction"`
	ParentID          *string          `json:"parentId"`
	ConfirmConversion bool             `json:"confirmConversion"`
	ActualStart       *string          `json:"actualStart"`
	ActualEnd         *string          `json:"actualEnd"`
}
type response struct {
	Data any `json:"data"`
}
type timelineItem struct {
	Start *string `json:"start,omitempty"`
	End   *string `json:"end,omitempty"`
}
type executableItem struct {
	RoleID                      *string      `json:"roleId,omitempty"`
	AssigneeID                  *string      `json:"assigneeId,omitempty"`
	EffortMinutes               *int         `json:"effortMinutes,omitempty"`
	LagDays                     int          `json:"lag"`
	ExecutionTimeline           timelineItem `json:"executionTimeline"`
	CommitmentTimeline          timelineItem `json:"commitmentTimeline"`
	ExecutionUnscheduledReason  *string      `json:"executionUnscheduledReason,omitempty"`
	CommitmentUnscheduledReason *string      `json:"commitmentUnscheduledReason,omitempty"`
	ActualStart                 *string      `json:"actualStart,omitempty"`
	ActualEnd                   *string      `json:"actualEnd,omitempty"`
}
type item struct {
	ID          string         `json:"id"`
	ProjectID   string         `json:"projectId"`
	ParentID    *string        `json:"parentId,omitempty"`
	Name        string         `json:"name"`
	Position    int            `json:"position"`
	HasChildren bool           `json:"hasChildren"`
	Executable  executableItem `json:"executable"`
	Children    []item         `json:"children"`
}
type schedulePreviewTaskItem struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	ProjectID     string  `json:"projectId"`
	ProjectName   string  `json:"projectName"`
	HierarchyPath string  `json:"hierarchyPath"`
	Completed     bool    `json:"completed"`
	ExpectedStart *string `json:"expectedStart"`
}
type schedulePreviewRelationItem struct {
	ID              string                  `json:"id"`
	Source          string                  `json:"source"`
	ManualRemovable bool                    `json:"manualRemovable"`
	Task            schedulePreviewTaskItem `json:"task"`
}
type schedulePreviewDependencyDetail struct {
	BlockedBy []schedulePreviewRelationItem `json:"blockedBy"`
	Blocks    []schedulePreviewRelationItem `json:"blocks"`
}
type schedulePreviewItem struct {
	Task         item                            `json:"task"`
	Dependencies schedulePreviewDependencyDetail `json:"dependencies"`
}
type allocationRowItem struct {
	Date                string `json:"date"`
	AllocatedMinutes    int    `json:"allocatedMinutes"`
	CapacityMinutes     int    `json:"capacityMinutes"`
	RemainingMinutes    int    `json:"remainingMinutes"`
	OvercapacityMinutes int    `json:"overcapacityMinutes"`
}
type allocationGroupsItem struct {
	Execution  []allocationRowItem `json:"execution"`
	Commitment []allocationRowItem `json:"commitment"`
	Actual     []allocationRowItem `json:"actual"`
}
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (h *Handler) Register(m *http.ServeMux) {
	m.HandleFunc("GET /api/projects/{projectId}/wbs", h.tree)
	m.HandleFunc("GET /api/projects/{projectId}/wbs/{wbsId}", h.get)
	m.HandleFunc("GET /api/projects/{projectId}/wbs/{wbsId}/allocations", h.allocations)
	m.HandleFunc("POST /api/projects/{projectId}/wbs", h.create)
	m.HandleFunc("POST /api/projects/{projectId}/wbs/{wbsId}/children", h.child)
	m.HandleFunc("PUT /api/projects/{projectId}/wbs/{wbsId}", h.rename)
	m.HandleFunc("PUT /api/projects/{projectId}/wbs/{wbsId}/executable", h.executable)
	m.HandleFunc("POST /api/projects/{projectId}/wbs/{wbsId}/executable/preview", h.previewExecutable)
	m.HandleFunc("POST /api/projects/{projectId}/wbs/{wbsId}/actual-date", h.complete)
	m.HandleFunc("POST /api/projects/{projectId}/wbs/{wbsId}/reopen", h.reopen)
	m.HandleFunc("POST /api/projects/{projectId}/wbs/{wbsId}/reorder", h.reorder)
	m.HandleFunc("POST /api/projects/{projectId}/wbs/{wbsId}/move", h.move)
	m.HandleFunc("DELETE /api/projects/{projectId}/wbs/{wbsId}", h.delete)
}
func (h *Handler) tree(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.Tree(r.Context(), r.PathValue("projectId"))
	if e != nil {
		writeError(w, e)
		return
	}
	items := make([]item, 0, len(v))
	for _, node := range v {
		items = append(items, mapNode(node))
	}
	httpjson.Write(w, 200, response{items})
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	v, e := h.service.Get(r.Context(), r.PathValue("projectId"), r.PathValue("wbsId"))
	if e != nil {
		writeError(w, e)
		return
	}
	httpjson.Write(w, 200, response{mapNode(*v)})
}
func (h *Handler) allocations(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.Allocations(r.Context(), r.PathValue("projectId"), r.PathValue("wbsId"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, 200, response{mapAllocationGroups(*value)})
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var p writeRequest
	if !decode(w, r, &p) {
		return
	}
	v, e := h.service.Create(r.Context(), r.PathValue("projectId"), p.ParentID, p.Name, p.ConfirmConversion)
	if e != nil {
		writeError(w, e)
		return
	}
	httpjson.Write(w, 201, response{mapNode(*v)})
}
func (h *Handler) child(w http.ResponseWriter, r *http.Request) {
	var p writeRequest
	if !decode(w, r, &p) {
		return
	}
	id := r.PathValue("wbsId")
	v, e := h.service.Create(r.Context(), r.PathValue("projectId"), &id, p.Name, p.ConfirmConversion)
	if e != nil {
		writeError(w, e)
		return
	}
	httpjson.Write(w, 201, response{mapNode(*v)})
}
func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	var p writeRequest
	if !decode(w, r, &p) {
		return
	}
	v, e := h.service.Rename(r.Context(), r.PathValue("projectId"), r.PathValue("wbsId"), p.Name)
	if e != nil {
		writeError(w, e)
		return
	}
	httpjson.Write(w, 200, response{mapNode(*v)})
}
func (h *Handler) executable(w http.ResponseWriter, r *http.Request) {
	var p executableRequest
	if !decode(w, r, &p) {
		return
	}
	minutes, ok := effortMinutes(w, p.EffortHours)
	if !ok {
		return
	}
	execution, ok := timeline(w, p.ExecutionStart, p.ExecutionEnd)
	if !ok {
		return
	}
	commitment, ok := timeline(w, p.CommitmentStart, p.CommitmentEnd)
	if !ok {
		return
	}
	lagDays, ok := parseLag(w, p.LagDays)
	if !ok {
		return
	}
	v, e := h.service.UpdateExecutable(r.Context(), r.PathValue("projectId"), r.PathValue("wbsId"), application.WriteExecutableInput{Name: p.Name, RoleID: p.RoleID, AssigneeID: p.AssigneeID, EffortMinutes: minutes, LagDays: lagDays, Execution: execution, Commitment: commitment})
	if e != nil {
		writeError(w, e)
		return
	}
	httpjson.Write(w, 200, response{mapNode(*v)})
}
func (h *Handler) previewExecutable(w http.ResponseWriter, r *http.Request) {
	var p previewExecutableRequest
	if !decode(w, r, &p) {
		return
	}
	minutes, ok := effortMinutes(w, p.EffortHours)
	if !ok {
		return
	}
	lagDays, ok := parseLag(w, p.LagDays)
	if !ok {
		return
	}
	value, err := h.service.PreviewExecutableSchedule(
		r.Context(),
		r.PathValue("projectId"),
		r.PathValue("wbsId"),
		application.PreviewExecutableInput{
			RoleID:        p.RoleID,
			AssigneeID:    p.AssigneeID,
			EffortMinutes: minutes,
			LagDays:       lagDays,
		},
	)
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, response{schedulePreviewItem{
		Task:         mapNode(*value.Task),
		Dependencies: mapSchedulePreviewDependencies(value.Dependencies),
	}})
}

func mapSchedulePreviewDependencies(value dependencydomain.Detail) schedulePreviewDependencyDetail {
	return schedulePreviewDependencyDetail{
		BlockedBy: mapSchedulePreviewRelations(value.BlockedBy),
		Blocks:    mapSchedulePreviewRelations(value.Blocks),
	}
}

func mapSchedulePreviewRelations(values []dependencydomain.Item) []schedulePreviewRelationItem {
	items := make([]schedulePreviewRelationItem, 0, len(values))
	for _, value := range values {
		var expectedStart *string
		if value.Task.ExpectedStart != nil {
			formatted := value.Task.ExpectedStart.Format("2006-01-02")
			expectedStart = &formatted
		}
		items = append(items, schedulePreviewRelationItem{
			ID:              value.Dependency.ID,
			Source:          string(value.Dependency.Source()),
			ManualRemovable: value.Dependency.ManualRemovable(),
			Task: schedulePreviewTaskItem{
				ID:            value.Task.ID,
				Name:          value.Task.Name,
				ProjectID:     value.Task.ProjectID,
				ProjectName:   value.Task.ProjectName,
				HierarchyPath: value.Task.HierarchyPath,
				Completed:     value.Task.ActualStart != nil && value.Task.ActualEnd != nil,
				ExpectedStart: expectedStart,
			},
		})
	}
	return items
}

func (h *Handler) complete(w http.ResponseWriter, r *http.Request) {
	var p commandRequest
	if !decode(w, r, &p) {
		return
	}
	if (p.ActualStart == nil) != (p.ActualEnd == nil) || p.ActualStart == nil {
		httpjson.Write(w, 400, errorResponse{"ACTUAL_DATE_INCOMPLETE", domain.ErrActualDatePair.Error(), "actualDate"})
		return
	}
	actualStart, startError := time.Parse("2006-01-02", *p.ActualStart)
	actualEnd, endError := time.Parse("2006-01-02", *p.ActualEnd)
	if startError != nil || endError != nil {
		httpjson.Write(w, 400, errorResponse{"INVALID_REQUEST", "actualStart and actualEnd must use YYYY-MM-DD", "actualDate"})
		return
	}
	if actualEnd.Before(actualStart) {
		httpjson.Write(w, 400, errorResponse{"ACTUAL_DATE_INVALID_RANGE", domain.ErrActualDateOrder.Error(), "actualDate"})
		return
	}
	v, e := h.service.Complete(r.Context(), r.PathValue("projectId"), r.PathValue("wbsId"), actualStart, actualEnd)
	if e != nil {
		writeError(w, e)
		return
	}
	httpjson.Write(w, 200, response{mapNode(*v)})
}
func (h *Handler) reopen(w http.ResponseWriter, r *http.Request) {
	if !decodeOptionalEmptyObject(w, r) {
		return
	}
	value, err := h.service.Reopen(r.Context(), r.PathValue("projectId"), r.PathValue("wbsId"))
	if err != nil {
		writeReopenError(w, err)
		return
	}
	httpjson.Write(w, 200, response{mapReopenNode(*value)})
}
func (h *Handler) reorder(w http.ResponseWriter, r *http.Request) {
	var p commandRequest
	if !decode(w, r, &p) {
		return
	}
	if e := h.service.Reorder(r.Context(), r.PathValue("projectId"), r.PathValue("wbsId"), p.Direction); e != nil {
		writeError(w, e)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) move(w http.ResponseWriter, r *http.Request) {
	var p commandRequest
	if !decode(w, r, &p) {
		return
	}
	if e := h.service.Move(r.Context(), r.PathValue("projectId"), r.PathValue("wbsId"), p.ParentID, p.ConfirmConversion); e != nil {
		writeError(w, e)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if e := h.service.Delete(r.Context(), r.PathValue("projectId"), r.PathValue("wbsId")); e != nil {
		writeError(w, e)
		return
	}
	w.WriteHeader(204)
}
func decodeOptionalEmptyObject(w http.ResponseWriter, r *http.Request) bool {
	decoder := json.NewDecoder(r.Body)
	var payload map[string]json.RawMessage
	if err := decoder.Decode(&payload); errors.Is(err, io.EOF) {
		return true
	} else if err != nil || payload == nil || len(payload) != 0 {
		httpjson.Write(w, 400, errorResponse{"INVALID_REQUEST", "Request body must be empty or an empty JSON object", ""})
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		httpjson.Write(w, 400, errorResponse{"INVALID_REQUEST", "Request body is invalid", ""})
		return false
	}
	return true
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(target); e != nil {
		httpjson.Write(w, 400, errorResponse{"INVALID_REQUEST", "Request body is invalid", ""})
		return false
	}
	var extra any
	if e := d.Decode(&extra); !errors.Is(e, io.EOF) {
		httpjson.Write(w, 400, errorResponse{"INVALID_REQUEST", "Request body is invalid", ""})
		return false
	}
	return true
}
func effortMinutes(w http.ResponseWriter, hours *float64) (*int, bool) {
	if hours == nil {
		return nil, true
	}
	value := int(*hours * 60)
	if float64(value) != *hours*60 {
		writeError(w, domain.ErrEffortInvalid)
		return nil, false
	}
	return &value, true
}

func parseLag(w http.ResponseWriter, raw json.RawMessage) (int, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, true
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil || value < 0 {
		httpjson.Write(w, 400, errorResponse{"INVALID_LAG", domain.ErrLagInvalid.Error(), "lag"})
		return 0, false
	}
	return value, true
}

func timeline(w http.ResponseWriter, start, end *string) (domain.Timeline, bool) {
	parse := func(v *string) (*time.Time, error) {
		if v == nil {
			return nil, nil
		}
		d, e := time.Parse("2006-01-02", *v)
		return &d, e
	}
	s, e := parse(start)
	if e != nil {
		httpjson.Write(w, 400, errorResponse{"INVALID_REQUEST", "date must use YYYY-MM-DD", ""})
		return domain.Timeline{}, false
	}
	finish, e := parse(end)
	if e != nil {
		httpjson.Write(w, 400, errorResponse{"INVALID_REQUEST", "date must use YYYY-MM-DD", ""})
		return domain.Timeline{}, false
	}
	return domain.Timeline{Start: s, End: finish}, true
}
func writeReopenError(w http.ResponseWriter, err error) {
	if schedulingimpact.WriteHTTPError(w, err) {
		return
	}
	status, code, message := 500, "TASK_REOPEN_FAILED", "Task could not be reopened. Try again."
	switch {
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, domain.ErrProjectNotFound):
		status, code, message = 404, "TASK_NOT_FOUND", domain.ErrNotFound.Error()
	case errors.Is(err, domain.ErrExecutableOnly):
		status, code, message = 409, "EXECUTABLE_TASK_REQUIRED", "Reopen is available only for a Task."
	case errors.Is(err, domain.ErrTaskNotCompleted):
		status, code, message = 409, "TASK_NOT_COMPLETED", domain.ErrTaskNotCompleted.Error()
	case errors.Is(err, domain.ErrProjectClosedReadOnly):
		status, code, message = 409, "PROJECT_CLOSED_READ_ONLY", domain.ErrProjectClosedReadOnly.Error()
	case errors.Is(err, domain.ErrProjectLockedReadOnly):
		status, code, message = 409, "PROJECT_LOCKED_READ_ONLY", domain.ErrProjectLockedReadOnly.Error()
	case errors.Is(err, domain.ErrTaskReopenConflict):
		status, code, message = 409, "TASK_REOPEN_CONFLICT", "The Task was changed by another request. Refresh and try again."
	}
	httpjson.Write(w, status, errorResponse{code, message, ""})
}

func writeError(w http.ResponseWriter, err error) {
	if schedulingimpact.WriteHTTPError(w, err) {
		return
	}
	status, code := 500, "INTERNAL_ERROR"
	switch {
	case errors.Is(err, domain.ErrProjectNotFound):
		status, code = 404, "PROJECT_NOT_FOUND"
	case errors.Is(err, domain.ErrNotFound):
		status, code = 404, "WBS_NOT_FOUND"
	case errors.Is(err, domain.ErrNameRequired):
		status, code = 400, "WBS_NAME_REQUIRED"
	case errors.Is(err, domain.ErrNameTooLong):
		status, code = 400, "WBS_NAME_TOO_LONG"
	case errors.Is(err, domain.ErrNameExists):
		status, code = 409, "WBS_NAME_EXISTS"
	case errors.Is(err, domain.ErrConversionRequired):
		status, code = 409, "WBS_CONVERSION_REQUIRED"
	case errors.Is(err, domain.ErrHasChildren):
		status, code = 409, "WBS_HAS_CHILDREN"
	case errors.Is(err, domain.ErrCycle):
		status, code = 409, "WBS_MOVE_CYCLE"
	case errors.Is(err, domain.ErrCompletedReadOnly):
		status, code = 409, "COMPLETED_TASK_READ_ONLY"
	case errors.Is(err, domain.ErrProjectClosedReadOnly):
		status, code = 409, "PROJECT_CLOSED_READ_ONLY"
	case errors.Is(err, domain.ErrProjectLockedReadOnly):
		status, code = 409, "PROJECT_LOCKED_READ_ONLY"
	case errors.Is(err, domain.ErrIncompletePredecessor):
		status, code = 409, "ACTUAL_DATE_PREDECESSOR_UNFINISHED"
	case errors.Is(err, domain.ErrActualDatePair):
		status, code = 400, "ACTUAL_DATE_INCOMPLETE"
	case errors.Is(err, domain.ErrActualDateOrder):
		status, code = 400, "ACTUAL_DATE_INVALID_RANGE"
	case errors.Is(err, domain.ErrSchedulePreviewIncomplete):
		status, code = 400, "SCHEDULING_INPUT_INCOMPLETE"
	case errors.Is(err, domain.ErrSchedulePreviewUnavailable):
		status, code = 409, "SCHEDULE_PREVIEW_UNAVAILABLE"
	case errors.Is(err, domain.ErrManualTimeline):
		status, code = 409, "WBS_MANUAL_TIMELINE_NOT_ALLOWED"
	case errors.Is(err, domain.ErrRoleAssigneeMismatch):
		status, code = 400, "WBS_ROLE_ASSIGNEE_MISMATCH"
	case errors.Is(err, domain.ErrExecutableOnly):
		status, code = 409, "WBS_GROUPING_EXECUTABLE_FIELDS_NOT_ALLOWED"
	case errors.Is(err, domain.ErrMoveNotAllowed):
		status, code = 409, "WBS_MOVE_NOT_ALLOWED"
	case errors.Is(err, domain.ErrLagInvalid):
		status, code = 400, "INVALID_LAG"
	case errors.Is(err, domain.ErrEffortInvalid) || errors.Is(err, domain.ErrTimelinePair) || errors.Is(err, domain.ErrTimelineOrder):
		status, code = 400, "WBS_EXECUTABLE_INVALID"
	case errors.Is(err, schedulingdomain.ErrConcurrentConflict):
		status, code = 409, "SCHEDULING_CONFLICT"
	case errors.Is(err, schedulingdomain.ErrDataIntegrity), errors.Is(err, schedulingdomain.ErrNoConvergence):
		status, code = 409, "SCHEDULING_DATA_INTEGRITY_CONFLICT"
	}
	message := err.Error()
	if status == 500 {
		message = "An internal error occurred"
	}
	httpjson.Write(w, status, errorResponse{code, message, ""})
}

func mapAllocationGroups(value application.AllocationGroups) allocationGroupsItem {
	return allocationGroupsItem{
		Execution:  mapAllocationRows(value.Execution),
		Commitment: mapAllocationRows(value.Commitment),
		Actual:     mapAllocationRows(value.Actual),
	}
}

func mapAllocationRows(values []application.AllocationRow) []allocationRowItem {
	items := make([]allocationRowItem, 0, len(values))
	for _, value := range values {
		items = append(items, allocationRowItem{
			Date:                value.Date.Format("2006-01-02"),
			AllocatedMinutes:    value.AllocatedMinutes,
			CapacityMinutes:     value.CapacityMinutes,
			RemainingMinutes:    value.RemainingMinutes,
			OvercapacityMinutes: value.OvercapacityMinutes,
		})
	}
	return items
}

func mapReopenNode(value domain.Node) map[string]any {
	children := make([]map[string]any, 0, len(value.Children))
	for _, child := range value.Children {
		children = append(children, mapReopenNode(child))
	}
	return map[string]any{
		"id":          value.ID,
		"projectId":   value.ProjectID,
		"parentId":    value.ParentID,
		"name":        value.Name,
		"position":    value.Position,
		"hasChildren": value.HasChildren,
		"executable": map[string]any{
			"roleId":                      value.Executable.RoleID,
			"assigneeId":                  value.Executable.AssigneeID,
			"effortMinutes":               value.Executable.EffortMinutes,
			"lag":                         value.Executable.LagDays,
			"executionTimeline":           timelineItem{Start: date(value.Executable.ExecutionTimeline.Start), End: date(value.Executable.ExecutionTimeline.End)},
			"commitmentTimeline":          timelineItem{Start: date(value.Executable.CommitmentTimeline.Start), End: date(value.Executable.CommitmentTimeline.End)},
			"executionUnscheduledReason":  value.Executable.ExecutionUnscheduledReason,
			"commitmentUnscheduledReason": value.Executable.CommitmentUnscheduledReason,
			"actualStart":                 date(value.Executable.ActualStart),
			"actualEnd":                   date(value.Executable.ActualEnd),
		},
		"children": children,
	}
}

func mapNode(value domain.Node) item {
	children := make([]item, 0, len(value.Children))
	for _, child := range value.Children {
		children = append(children, mapNode(child))
	}
	return item{ID: value.ID, ProjectID: value.ProjectID, ParentID: value.ParentID, Name: value.Name, Position: value.Position, HasChildren: value.HasChildren, Executable: executableItem{RoleID: value.Executable.RoleID, AssigneeID: value.Executable.AssigneeID, EffortMinutes: value.Executable.EffortMinutes, LagDays: value.Executable.LagDays, ExecutionTimeline: timelineItem{Start: date(value.Executable.ExecutionTimeline.Start), End: date(value.Executable.ExecutionTimeline.End)}, CommitmentTimeline: timelineItem{Start: date(value.Executable.CommitmentTimeline.Start), End: date(value.Executable.CommitmentTimeline.End)}, ExecutionUnscheduledReason: value.Executable.ExecutionUnscheduledReason, CommitmentUnscheduledReason: value.Executable.CommitmentUnscheduledReason, ActualStart: date(value.Executable.ActualStart), ActualEnd: date(value.Executable.ActualEnd)}, Children: children}
}
func date(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02")
	return &formatted
}
