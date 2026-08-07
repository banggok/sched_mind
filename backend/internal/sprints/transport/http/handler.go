package sprinthttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/banggok/sched_mind/backend/internal/shared/httpjson"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/sprints/application"
	"github.com/banggok/sched_mind/backend/internal/sprints/domain"
)

type Service interface {
	List(context.Context, listing.Query) (listing.Page[domain.Sprint], error)
	Get(context.Context, string) (*domain.Sprint, error)
	Detail(context.Context, string) (*application.Detail, error)
	Create(context.Context, application.WriteInput) (*domain.Sprint, error)
	Update(context.Context, string, application.WriteInput) (*domain.Sprint, error)
	Start(context.Context, string, int64) (*domain.Sprint, error)
	Delete(context.Context, string, int64) error
	Suggest(context.Context, application.SuggestionInput) (*application.Suggestion, error)
	Candidates(context.Context, string, listing.Query) (listing.Page[application.TaskProjection], error)
	DraftCandidates(context.Context, application.CandidateInput, listing.Query) (listing.Page[application.TaskProjection], error)
}

type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }

func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sprints", handler.list)
	mux.HandleFunc("POST /api/sprints", handler.create)
	mux.HandleFunc("GET /api/sprints/{sprintId}", handler.detail)
	mux.HandleFunc("PUT /api/sprints/{sprintId}", handler.update)
	mux.HandleFunc("POST /api/sprints/{sprintId}/start", handler.start)
	mux.HandleFunc("DELETE /api/sprints/{sprintId}", handler.delete)
	mux.HandleFunc("POST /api/sprints/suggestion", handler.suggest)
	mux.HandleFunc("POST /api/sprints/task-candidates", handler.draftCandidates)
	mux.HandleFunc("GET /api/sprints/{sprintId}/task-candidates", handler.candidates)
}

type writeRequest struct {
	Name      string   `json:"name"`
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	MemberIDs []string `json:"memberIds"`
	TaskIDs   []string `json:"taskIds"`
	Version   int64    `json:"version"`
}

type suggestionRequest struct {
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	MemberIDs []string `json:"memberIds"`
}

type candidateRequest struct {
	StartDate       string   `json:"startDate"`
	EndDate         string   `json:"endDate"`
	MemberIDs       []string `json:"memberIds"`
	ExcludedTaskIDs []string `json:"excludedTaskIds"`
}

type versionRequest struct {
	Version int64 `json:"version"`
}
type dataResponse struct {
	Data any `json:"data"`
}
type listResponse struct {
	Data     []sprintItem `json:"data"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
	Total    int64        `json:"total"`
}
type errorResponse struct {
	Code    string                  `json:"code"`
	Message string                  `json:"message"`
	Field   string                  `json:"field,omitempty"`
	Details *overlapDetailsResponse `json:"details,omitempty"`
}

type overlapDetailsResponse struct {
	SprintID   string                  `json:"sprintId"`
	SprintName string                  `json:"sprintName"`
	StartDate  string                  `json:"startDate"`
	EndDate    string                  `json:"endDate"`
	Members    []overlapMemberResponse `json:"members"`
}

type overlapMemberResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type sprintItem struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	StartDate string   `json:"startDate"`
	EndDate   string   `json:"endDate"`
	Status    string   `json:"status"`
	Version   int64    `json:"version"`
	MemberIDs []string `json:"memberIds,omitempty"`
	TaskIDs   []string `json:"taskIds,omitempty"`
	StartedAt *string  `json:"startedAt,omitempty"`
	CreatedAt string   `json:"createdAt"`
	UpdatedAt string   `json:"updatedAt"`
}

type dailyItem struct {
	Date    string `json:"date"`
	Minutes int64  `json:"minutes"`
}
type dailySummaryItem struct {
	Date                      string `json:"date"`
	CapacityMinutes           int64  `json:"capacityMinutes"`
	SelectedAllocationMinutes int64  `json:"selectedAllocationMinutes"`
	RemainingMinutes          int64  `json:"remainingMinutes"`
	OvercapacityMinutes       int64  `json:"overcapacityMinutes"`
}
type memberItem struct {
	ID                        string             `json:"id"`
	Name                      string             `json:"name"`
	RoleName                  string             `json:"roleName"`
	DailyCapacity             []dailyItem        `json:"dailyCapacity"`
	DailySummaries            []dailySummaryItem `json:"dailySummaries"`
	CapacityMinutes           int64              `json:"capacityMinutes"`
	InSprintAllocationMinutes int64              `json:"inSprintAllocationMinutes"`
	RemainingMinutes          int64              `json:"remainingMinutes"`
	OvercapacityMinutes       int64              `json:"overcapacityMinutes"`
	TotalAllocationMinutes    int64              `json:"totalAllocationMinutes"`
}
type taskItem struct {
	ID                        string      `json:"id"`
	ProjectID                 string      `json:"projectId"`
	ProjectName               string      `json:"projectName"`
	ParentName                string      `json:"parentName"`
	ProjectStatus             string      `json:"projectStatus"`
	ProjectPriority           int         `json:"projectPriority"`
	Name                      string      `json:"name"`
	WBSOrder                  string      `json:"wbsOrder"`
	WBSPath                   string      `json:"wbsPath"`
	WBSRank                   int         `json:"wbsRank"`
	AssigneeID                *string     `json:"assigneeId"`
	AssigneeName              *string     `json:"assigneeName"`
	EffortMinutes             *int        `json:"effortMinutes"`
	ExecutionStart            *string     `json:"executionStart"`
	ExecutionEnd              *string     `json:"executionEnd"`
	CommitmentStart           *string     `json:"commitmentStart"`
	CommitmentEnd             *string     `json:"commitmentEnd"`
	DailyPlanOrderDate        *string     `json:"dailyPlanOrderDate"`
	Completed                 bool        `json:"completed"`
	Allocations               []dailyItem `json:"allocations"`
	InSprintAllocationMinutes int64       `json:"inSprintAllocationMinutes"`
	OutsideAllocationMinutes  int64       `json:"outsideAllocationMinutes"`
	TotalAllocationMinutes    int64       `json:"totalAllocationMinutes"`
	Warnings                  []string    `json:"warnings"`
}
type totalsItem struct {
	CapacityMinutes                 int64              `json:"capacityMinutes"`
	SelectedMemberAllocationMinutes int64              `json:"selectedMemberAllocationMinutes"`
	RemainingMinutes                int64              `json:"remainingMinutes"`
	OvercapacityMinutes             int64              `json:"overcapacityMinutes"`
	NeedsReviewAllocationMinutes    int64              `json:"needsReviewAllocationMinutes"`
	NeedsReviewDailyAllocation      []dailyItem        `json:"needsReviewDailyAllocation"`
	AllTaskInSprintMinutes          int64              `json:"allTaskInSprintMinutes"`
	AllTaskTotalMinutes             int64              `json:"allTaskTotalMinutes"`
	DailySummaries                  []dailySummaryItem `json:"dailySummaries"`
}
type sprintSummaryItem struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	StartDate string  `json:"startDate"`
	EndDate   string  `json:"endDate"`
	Status    string  `json:"status"`
	Version   int64   `json:"version"`
	StartedAt *string `json:"startedAt,omitempty"`
}
type detailItem struct {
	Sprint          sprintSummaryItem `json:"sprint"`
	Members         []memberItem      `json:"members"`
	Tasks           []taskItem        `json:"tasks"`
	Totals          totalsItem        `json:"totals"`
	ProjectionToken string            `json:"projectionToken"`
}
type suggestedTaskItem struct {
	Task   taskItem `json:"task"`
	Reason string   `json:"reason"`
}
type suggestionItem struct {
	Members         []memberItem        `json:"members"`
	Tasks           []suggestedTaskItem `json:"tasks"`
	Totals          totalsItem          `json:"totals"`
	ProjectionToken string              `json:"projectionToken"`
}
type taskListResponse struct {
	Data     []taskItem `json:"data"`
	Page     int        `json:"page"`
	PageSize int        `json:"pageSize"`
	Total    int64      `json:"total"`
}

func (handler *Handler) list(w http.ResponseWriter, r *http.Request) {
	query, err := listing.ParseHTTPQuery(r)
	if err != nil {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}
	page, err := handler.service.List(r.Context(), query)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]sprintItem, 0, len(page.Items))
	for _, sprint := range page.Items {
		items = append(items, mapSprint(sprint))
	}
	httpjson.Write(w, http.StatusOK, listResponse{Data: items, Page: page.Page, PageSize: page.PageSize, Total: page.Total})
}

func (handler *Handler) create(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeWrite(w, r)
	if !ok {
		return
	}
	sprint, err := handler.service.Create(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusCreated, dataResponse{Data: mapSprint(*sprint)})
}

func (handler *Handler) detail(w http.ResponseWriter, r *http.Request) {
	detail, err := handler.service.Detail(r.Context(), r.PathValue("sprintId"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, dataResponse{Data: mapDetail(*detail)})
}

func (handler *Handler) update(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeWrite(w, r)
	if !ok {
		return
	}
	sprint, err := handler.service.Update(r.Context(), r.PathValue("sprintId"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, dataResponse{Data: mapSprint(*sprint)})
}

func (handler *Handler) start(w http.ResponseWriter, r *http.Request) {
	var payload versionRequest
	if !decode(w, r, &payload) {
		return
	}
	sprint, err := handler.service.Start(r.Context(), r.PathValue("sprintId"), payload.Version)
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, dataResponse{Data: mapSprint(*sprint)})
}

func (handler *Handler) delete(w http.ResponseWriter, r *http.Request) {
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{Code: "SPRINT_VERSION_INVALID", Message: "Sprint Version is invalid", Field: "version"})
		return
	}
	if err := handler.service.Delete(r.Context(), r.PathValue("sprintId"), version); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (handler *Handler) suggest(w http.ResponseWriter, r *http.Request) {
	var payload suggestionRequest
	if !decode(w, r, &payload) {
		return
	}
	start, end, ok := parseDates(w, payload.StartDate, payload.EndDate)
	if !ok {
		return
	}
	value, err := handler.service.Suggest(r.Context(), application.SuggestionInput{StartDate: start, EndDate: end, MemberIDs: payload.MemberIDs})
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, dataResponse{Data: mapSuggestion(*value)})
}

func (handler *Handler) candidates(w http.ResponseWriter, r *http.Request) {
	query, err := listing.ParseHTTPQuery(r)
	if err != nil {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}
	page, err := handler.service.Candidates(r.Context(), r.PathValue("sprintId"), query)
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, taskListResponse{Data: mapTasks(page.Items), Page: page.Page, PageSize: page.PageSize, Total: page.Total})
}

func (handler *Handler) draftCandidates(w http.ResponseWriter, r *http.Request) {
	query, err := listing.ParseHTTPQuery(r)
	if err != nil {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}
	var payload candidateRequest
	if !decode(w, r, &payload) {
		return
	}
	start, end, ok := parseDates(w, payload.StartDate, payload.EndDate)
	if !ok {
		return
	}
	page, err := handler.service.DraftCandidates(r.Context(), application.CandidateInput{
		StartDate: start, EndDate: end, MemberIDs: payload.MemberIDs, ExcludedTaskIDs: payload.ExcludedTaskIDs,
	}, query)
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, taskListResponse{Data: mapTasks(page.Items), Page: page.Page, PageSize: page.PageSize, Total: page.Total})
}

func decodeWrite(w http.ResponseWriter, r *http.Request) (application.WriteInput, bool) {
	var payload writeRequest
	if !decode(w, r, &payload) {
		return application.WriteInput{}, false
	}
	start, end, ok := parseDates(w, payload.StartDate, payload.EndDate)
	if !ok {
		return application.WriteInput{}, false
	}
	return application.WriteInput{Name: payload.Name, StartDate: start, EndDate: end, MemberIDs: payload.MemberIDs, TaskIDs: payload.TaskIDs, Version: payload.Version}, true
}

func parseDates(w http.ResponseWriter, startValue, endValue string) (time.Time, time.Time, bool) {
	start, startErr := time.Parse("2006-01-02", startValue)
	end, endErr := time.Parse("2006-01-02", endValue)
	if startErr != nil || endErr != nil {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{Code: "SPRINT_DATE_RANGE_INVALID", Message: domain.ErrDateRangeInvalid.Error(), Field: "dateRange"})
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "Request body is invalid"})
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: "Request body is invalid"})
		return false
	}
	return true
}

func mapSprint(value domain.Sprint) sprintItem {
	return sprintItem{ID: value.ID, Name: value.Name, StartDate: date(value.StartDate), EndDate: date(value.EndDate), Status: string(value.Status), Version: value.Version,
		MemberIDs: append([]string(nil), value.MemberIDs...), TaskIDs: append([]string(nil), value.TaskIDs...), StartedAt: timestamp(value.StartedAt), CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}

func mapDetail(value application.Detail) detailItem {
	return detailItem{Sprint: sprintSummaryItem{ID: value.Sprint.ID, Name: value.Sprint.Name, StartDate: date(value.Sprint.StartDate), EndDate: date(value.Sprint.EndDate), Status: value.Sprint.Status, Version: value.Sprint.Version, StartedAt: timestamp(value.Sprint.StartedAt)}, Members: mapMembers(value.Members), Tasks: mapTasks(value.Tasks), Totals: mapTotals(value.Totals), ProjectionToken: value.ProjectionToken}
}
func mapSuggestion(value application.Suggestion) suggestionItem {
	tasks := make([]suggestedTaskItem, 0, len(value.Tasks))
	for _, task := range value.Tasks {
		tasks = append(tasks, suggestedTaskItem{Task: mapTask(task.Task), Reason: task.Reason})
	}
	return suggestionItem{Members: mapMembers(value.Members), Tasks: tasks, Totals: mapTotals(value.Totals), ProjectionToken: value.ProjectionToken}
}
func mapMembers(values []application.MemberProjection) []memberItem {
	items := make([]memberItem, 0, len(values))
	for _, value := range values {
		items = append(items, memberItem{ID: value.ID, Name: value.Name, RoleName: value.RoleName, DailyCapacity: mapDaily(value.DailyCapacity), DailySummaries: mapDailySummaries(value.DailySummaries), CapacityMinutes: value.CapacityMinutes, InSprintAllocationMinutes: value.InSprintAllocationMinutes, RemainingMinutes: value.RemainingMinutes, OvercapacityMinutes: value.OvercapacityMinutes, TotalAllocationMinutes: value.TotalAllocationMinutes})
	}
	return items
}
func mapTasks(values []application.TaskProjection) []taskItem {
	items := make([]taskItem, 0, len(values))
	for _, value := range values {
		items = append(items, mapTask(value))
	}
	return items
}
func mapTask(value application.TaskProjection) taskItem {
	return taskItem{ID: value.ID, ProjectID: value.ProjectID, ProjectName: value.ProjectName, ParentName: value.ParentName, ProjectStatus: value.ProjectStatus, ProjectPriority: value.ProjectPriority, Name: value.Name, WBSOrder: value.WBSOrder, WBSPath: value.WBSPath, WBSRank: value.WBSRank, AssigneeID: value.AssigneeID, AssigneeName: value.AssigneeName, EffortMinutes: value.EffortMinutes, ExecutionStart: datePointer(value.ExecutionStart), ExecutionEnd: datePointer(value.ExecutionEnd), CommitmentStart: datePointer(value.CommitmentStart), CommitmentEnd: datePointer(value.CommitmentEnd), DailyPlanOrderDate: datePointer(value.DailyPlanOrderDate), Completed: value.Completed, Allocations: mapDaily(value.Allocations), InSprintAllocationMinutes: value.InSprintAllocationMinutes, OutsideAllocationMinutes: value.OutsideAllocationMinutes, TotalAllocationMinutes: value.TotalAllocationMinutes, Warnings: append([]string{}, value.Warnings...)}
}
func mapDaily(values []application.DailyValue) []dailyItem {
	items := make([]dailyItem, 0, len(values))
	for _, value := range values {
		items = append(items, dailyItem{Date: date(value.Date), Minutes: value.Minutes})
	}
	return items
}
func mapDailySummaries(values []application.DailySummary) []dailySummaryItem {
	items := make([]dailySummaryItem, 0, len(values))
	for _, value := range values {
		items = append(items, dailySummaryItem{Date: date(value.Date), CapacityMinutes: value.CapacityMinutes, SelectedAllocationMinutes: value.SelectedAllocationMinutes, RemainingMinutes: value.RemainingMinutes, OvercapacityMinutes: value.OvercapacityMinutes})
	}
	return items
}
func mapTotals(value application.Totals) totalsItem {
	return totalsItem{CapacityMinutes: value.CapacityMinutes, SelectedMemberAllocationMinutes: value.SelectedMemberAllocationMinutes, RemainingMinutes: value.RemainingMinutes, OvercapacityMinutes: value.OvercapacityMinutes, NeedsReviewAllocationMinutes: value.NeedsReviewAllocationMinutes, NeedsReviewDailyAllocation: mapDaily(value.NeedsReviewDailyAllocation), AllTaskInSprintMinutes: value.AllTaskInSprintMinutes, AllTaskTotalMinutes: value.AllTaskTotalMinutes, DailySummaries: mapDailySummaries(value.DailySummaries)}
}
func date(value time.Time) string { return value.Format("2006-01-02") }
func datePointer(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := date(*value)
	return &formatted
}
func timestamp(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339Nano)
	return &formatted
}

func writeError(w http.ResponseWriter, err error) {
	status, code, message, field := http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred", ""
	var details *overlapDetailsResponse
	switch {
	case errors.Is(err, domain.ErrNameRequired):
		status, code, message, field = http.StatusBadRequest, "SPRINT_NAME_REQUIRED", domain.ErrNameRequired.Error(), "name"
	case errors.Is(err, domain.ErrNameTooLong):
		status, code, message, field = http.StatusBadRequest, "SPRINT_NAME_TOO_LONG", domain.ErrNameTooLong.Error(), "name"
	case errors.Is(err, domain.ErrDateRangeInvalid):
		status, code, message, field = http.StatusBadRequest, "SPRINT_DATE_RANGE_INVALID", domain.ErrDateRangeInvalid.Error(), "dateRange"
	case errors.Is(err, domain.ErrMemberRequired):
		status, code, message, field = http.StatusBadRequest, "SPRINT_MEMBER_REQUIRED", domain.ErrMemberRequired.Error(), "memberIds"
	case errors.Is(err, application.ErrNameConflict):
		status, code, message, field = http.StatusConflict, "SPRINT_NAME_CONFLICT", application.ErrNameConflict.Error(), "name"
	case errors.Is(err, application.ErrMemberNotFound):
		status, code, message, field = http.StatusConflict, "SPRINT_MEMBER_INACTIVE", application.ErrMemberNotFound.Error(), "memberIds"
	case errors.Is(err, application.ErrMemberOverlap):
		status, code, message = http.StatusConflict, "SPRINT_MEMBER_OVERLAP", application.ErrMemberOverlap.Error()
		var overlap *application.MemberOverlapError
		if errors.As(err, &overlap) {
			members := make([]overlapMemberResponse, 0, len(overlap.Members))
			for _, member := range overlap.Members {
				members = append(members, overlapMemberResponse{ID: member.ID, Name: member.Name})
			}
			details = &overlapDetailsResponse{SprintID: overlap.SprintID, SprintName: overlap.SprintName, StartDate: date(overlap.StartDate), EndDate: date(overlap.EndDate), Members: members}
		}
	case errors.Is(err, application.ErrTaskNotEligible):
		status, code, message, field = http.StatusConflict, "SPRINT_TASK_NOT_ELIGIBLE", application.ErrTaskNotEligible.Error(), "taskIds"
	case errors.Is(err, application.ErrTaskNotFound):
		status, code, message, field = http.StatusNotFound, "SPRINT_TASK_NOT_FOUND", application.ErrTaskNotFound.Error(), "taskIds"
	case errors.Is(err, application.ErrTaskUnscheduled):
		status, code, message, field = http.StatusConflict, "SPRINT_TASK_UNSCHEDULED", application.ErrTaskUnscheduled.Error(), "taskIds"
	case errors.Is(err, application.ErrSuggestionUnavailable):
		status, code, message = http.StatusServiceUnavailable, "SPRINT_SUGGESTION_UNAVAILABLE", "Sprint suggestion could not be composed from the current capacity and allocation data."
	case errors.Is(err, domain.ErrAlreadyStarted):
		status, code, message = http.StatusConflict, "SPRINT_ALREADY_STARTED", domain.ErrAlreadyStarted.Error()
	case errors.Is(err, application.ErrVersionConflict):
		status, code, message = http.StatusConflict, "SPRINT_VERSION_CONFLICT", application.ErrVersionConflict.Error()
	case errors.Is(err, application.ErrNotFound):
		status, code, message = http.StatusNotFound, "SPRINT_NOT_FOUND", application.ErrNotFound.Error()
	}
	httpjson.Write(w, status, errorResponse{Code: code, Message: message, Field: field, Details: details})
}
