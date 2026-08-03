package portfoliohttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/banggok/sched_mind/backend/internal/portfolio/application"
	"github.com/banggok/sched_mind/backend/internal/portfolio/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/httpjson"
)

type Service interface {
	ActiveProjects(context.Context) ([]domain.ProjectOption, error)
	Portfolio(context.Context, application.PortfolioQuery) (*domain.Portfolio, error)
	ListSavedFilters(context.Context) ([]domain.SavedFilter, error)
	CreateSavedFilter(context.Context, string, []string) (*domain.SavedFilter, error)
	UpdateSavedFilter(context.Context, string, int64, []string) (*domain.SavedFilter, error)
	DeleteSavedFilter(context.Context, string, int64) error
}

type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }

func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/portfolio/projects", handler.activeProjects)
	mux.HandleFunc("GET /api/portfolio", handler.portfolio)
	mux.HandleFunc("GET /api/portfolio/saved-filters", handler.savedFilters)
	mux.HandleFunc("POST /api/portfolio/saved-filters", handler.createSavedFilter)
	mux.HandleFunc("PUT /api/portfolio/saved-filters/{filterId}", handler.updateSavedFilter)
	mux.HandleFunc("DELETE /api/portfolio/saved-filters/{filterId}", handler.deleteSavedFilter)
}

type response struct {
	Data any `json:"data"`
}
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}
type projectItem struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	Priority        int    `json:"priority"`
	ScheduleVersion int64  `json:"scheduleVersion"`
}
type rowItem struct {
	ID                 string  `json:"id"`
	ProjectID          string  `json:"projectId"`
	ParentID           *string `json:"parentId,omitempty"`
	Kind               string  `json:"kind"`
	Name               string  `json:"name"`
	WBSNumber          string  `json:"wbsNumber"`
	Depth              int     `json:"depth"`
	Position           int     `json:"position"`
	Status             string  `json:"status,omitempty"`
	RoleID             *string `json:"roleId,omitempty"`
	RoleName           *string `json:"roleName,omitempty"`
	AssigneeID         *string `json:"assigneeId,omitempty"`
	AssigneeName       *string `json:"assigneeName,omitempty"`
	EffortMinutes      *int    `json:"effortMinutes,omitempty"`
	Start              *string `json:"start,omitempty"`
	End                *string `json:"end,omitempty"`
	UnscheduledReason  *string `json:"unscheduledReason,omitempty"`
	IncompleteEffort   bool    `json:"incompleteEffort"`
	IncompleteSchedule bool    `json:"incompleteSchedule"`
	HasChildren        bool    `json:"hasChildren"`
	Completed          bool    `json:"completed"`
}
type dependencyItem struct {
	ID             string `json:"id"`
	BlockingTaskID string `json:"blockingTaskId"`
	BlockedTaskID  string `json:"blockedTaskId"`
}
type holidayItem struct {
	Date        string `json:"date"`
	Description string `json:"description"`
}
type portfolioItem struct {
	Projection       string           `json:"projection"`
	Projects         []projectItem    `json:"projects"`
	Rows             []rowItem        `json:"rows"`
	Dependencies     []dependencyItem `json:"dependencies"`
	Holidays         []holidayItem    `json:"holidays"`
	WorkingDayAnchor *string          `json:"workingDayAnchor,omitempty"`
}
type savedFilterItem struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ProjectIDs []string `json:"projectIds"`
	Version    int64    `json:"version"`
	CreatedAt  string   `json:"createdAt"`
	UpdatedAt  string   `json:"updatedAt"`
}
type savedFilterRequest struct {
	Name       string   `json:"name"`
	ProjectIDs []string `json:"projectIds"`
	Version    int64    `json:"version"`
}

func (handler *Handler) activeProjects(w http.ResponseWriter, r *http.Request) {
	values, err := handler.service.ActiveProjects(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]projectItem, 0, len(values))
	for _, value := range values {
		items = append(items, mapProject(value))
	}
	httpjson.Write(w, http.StatusOK, response{items})
}

func (handler *Handler) portfolio(w http.ResponseWriter, r *http.Request) {
	projection, err := domain.ParseProjection(r.URL.Query().Get("projection"))
	if err != nil {
		writeError(w, err)
		return
	}
	from, err := time.Parse("2006-01-02", r.URL.Query().Get("from"))
	if err != nil {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{"PORTFOLIO_DATE_RANGE_INVALID", domain.ErrDateRangeInvalid.Error(), "from"})
		return
	}
	to, err := time.Parse("2006-01-02", r.URL.Query().Get("to"))
	if err != nil {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{"PORTFOLIO_DATE_RANGE_INVALID", domain.ErrDateRangeInvalid.Error(), "to"})
		return
	}
	projectIDs := splitIDs(r.URL.Query()["projectId"])
	value, err := handler.service.Portfolio(r.Context(), application.PortfolioQuery{ProjectIDs: projectIDs, Projection: projection, From: from, To: to})
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, response{mapPortfolio(*value)})
}

func (handler *Handler) savedFilters(w http.ResponseWriter, r *http.Request) {
	values, err := handler.service.ListSavedFilters(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]savedFilterItem, 0, len(values))
	for _, value := range values {
		items = append(items, mapSavedFilter(value))
	}
	httpjson.Write(w, http.StatusOK, response{items})
}

func (handler *Handler) createSavedFilter(w http.ResponseWriter, r *http.Request) {
	var payload savedFilterRequest
	if !decode(w, r, &payload) {
		return
	}
	value, err := handler.service.CreateSavedFilter(r.Context(), payload.Name, payload.ProjectIDs)
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusCreated, response{mapSavedFilter(*value)})
}

func (handler *Handler) updateSavedFilter(w http.ResponseWriter, r *http.Request) {
	var payload savedFilterRequest
	if !decode(w, r, &payload) {
		return
	}
	value, err := handler.service.UpdateSavedFilter(r.Context(), r.PathValue("filterId"), payload.Version, payload.ProjectIDs)
	if err != nil {
		writeError(w, err)
		return
	}
	httpjson.Write(w, http.StatusOK, response{mapSavedFilter(*value)})
}

func (handler *Handler) deleteSavedFilter(w http.ResponseWriter, r *http.Request) {
	version, err := strconv.ParseInt(r.URL.Query().Get("version"), 10, 64)
	if err != nil {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{"SAVED_FILTER_VERSION_INVALID", domain.ErrFilterConflict.Error(), "version"})
		return
	}
	if err := handler.service.DeleteSavedFilter(r.Context(), r.PathValue("filterId"), version); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mapProject(value domain.ProjectOption) projectItem {
	return projectItem{ID: value.ID, Name: value.Name, Status: value.Status, Priority: value.Priority, ScheduleVersion: value.ScheduleVersion}
}

func mapPortfolio(value domain.Portfolio) portfolioItem {
	projects := make([]projectItem, 0, len(value.Projects))
	for _, project := range value.Projects {
		projects = append(projects, mapProject(project))
	}
	rows := make([]rowItem, 0, len(value.Rows))
	for _, row := range value.Rows {
		rows = append(rows, rowItem{
			ID: row.ID, ProjectID: row.ProjectID, ParentID: row.ParentID, Kind: row.Kind, Name: row.Name,
			WBSNumber: row.WBSNumber, Depth: row.Depth, Position: row.Position, Status: row.Status,
			RoleID: row.RoleID, RoleName: row.RoleName,
			AssigneeID: row.AssigneeID, AssigneeName: row.AssigneeName, EffortMinutes: row.EffortMinutes,
			Start: dateString(row.Start), End: dateString(row.End), UnscheduledReason: row.UnscheduledReason,
			IncompleteEffort: row.IncompleteEffort, IncompleteSchedule: row.IncompleteSchedule, HasChildren: row.HasChildren, Completed: row.Completed,
		})
	}
	dependencies := make([]dependencyItem, 0, len(value.Dependencies))
	for _, dependency := range value.Dependencies {
		dependencies = append(dependencies, dependencyItem{ID: dependency.ID, BlockingTaskID: dependency.BlockingTaskID, BlockedTaskID: dependency.BlockedTaskID})
	}
	holidays := make([]holidayItem, 0, len(value.Holidays))
	for _, holiday := range value.Holidays {
		holidays = append(holidays, holidayItem{Date: holiday.Date.Format("2006-01-02"), Description: holiday.Description})
	}
	return portfolioItem{Projection: string(value.Projection), Projects: projects, Rows: rows, Dependencies: dependencies, Holidays: holidays, WorkingDayAnchor: dateString(value.WorkingDayAnchor)}
}

func mapSavedFilter(value domain.SavedFilter) savedFilterItem {
	return savedFilterItem{ID: value.ID, Name: value.Name, ProjectIDs: append([]string{}, value.ProjectIDs...), Version: value.Version, CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339Nano), UpdatedAt: value.UpdatedAt.UTC().Format(time.RFC3339Nano)}
}

func dateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02")
	return &formatted
}

func splitIDs(values []string) []string {
	out := make([]string, 0)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			if id := strings.TrimSpace(part); id != "" {
				out = append(out, id)
			}
		}
	}
	return out
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{"INVALID_REQUEST", "Request body is invalid", ""})
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		httpjson.Write(w, http.StatusBadRequest, errorResponse{"INVALID_REQUEST", "Request body is invalid", ""})
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, err error) {
	status, code, message, field := http.StatusInternalServerError, "INTERNAL_ERROR", "An internal error occurred", ""
	switch {
	case errors.Is(err, domain.ErrFilterNameRequired):
		status, code, message, field = http.StatusBadRequest, "SAVED_FILTER_NAME_REQUIRED", domain.ErrFilterNameRequired.Error(), "name"
	case errors.Is(err, domain.ErrFilterNameTooLong):
		status, code, message, field = http.StatusBadRequest, "SAVED_FILTER_NAME_TOO_LONG", domain.ErrFilterNameTooLong.Error(), "name"
	case errors.Is(err, domain.ErrFilterNameReserved):
		status, code, message, field = http.StatusBadRequest, "SAVED_FILTER_NAME_RESERVED", domain.ErrFilterNameReserved.Error(), "name"
	case errors.Is(err, domain.ErrFilterNameExists):
		status, code, message, field = http.StatusConflict, "SAVED_FILTER_NAME_EXISTS", domain.ErrFilterNameExists.Error(), "name"
	case errors.Is(err, domain.ErrFilterNotFound):
		status, code, message = http.StatusNotFound, "SAVED_FILTER_NOT_FOUND", domain.ErrFilterNotFound.Error()
	case errors.Is(err, domain.ErrFilterConflict):
		status, code, message = http.StatusConflict, "SAVED_FILTER_CONFLICT", domain.ErrFilterConflict.Error()
	case errors.Is(err, domain.ErrProjectionInvalid):
		status, code, message, field = http.StatusBadRequest, "PORTFOLIO_PROJECTION_INVALID", domain.ErrProjectionInvalid.Error(), "projection"
	case errors.Is(err, domain.ErrProjectLimit):
		status, code, message = http.StatusBadRequest, "PORTFOLIO_PROJECT_LIMIT", domain.ErrProjectLimit.Error()
	case errors.Is(err, domain.ErrDateRangeInvalid):
		status, code, message, field = http.StatusBadRequest, "PORTFOLIO_DATE_RANGE_INVALID", domain.ErrDateRangeInvalid.Error(), "dateRange"
	}
	httpjson.Write(w, status, errorResponse{Code: code, Message: message, Field: field})
}
