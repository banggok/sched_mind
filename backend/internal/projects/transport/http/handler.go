package projecthttp

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/banggok/sched_mind/backend/internal/projects/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type Service interface {
	List(context.Context, listing.Query) (listing.Page[domain.Project], error)
	Get(context.Context, string) (*domain.Project, error)
	Create(context.Context, string, bool, int) (*domain.Project, error)
	Update(context.Context, string, string, bool, int) (*domain.Project, error)
	ChangeStatus(context.Context, string, domain.Status) (*domain.Project, error)
	MovePriority(context.Context, string, domain.PriorityDirection) (*domain.Project, error)
	Delete(context.Context, string) error
	UpdateSettings(context.Context, string, bool, int) (*domain.Project, error)
}

type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/projects", h.list)
	mux.HandleFunc("GET /api/projects/{projectId}", h.get)
	mux.HandleFunc("POST /api/projects", h.create)
	mux.HandleFunc("PUT /api/projects/{projectId}", h.update)
	mux.HandleFunc("POST /api/projects/{projectId}/status", h.changeStatus)
	mux.HandleFunc("POST /api/projects/{projectId}/priority", h.movePriority)
	mux.HandleFunc("PATCH /api/projects/{projectId}/settings", h.updateSettings)
	mux.HandleFunc("DELETE /api/projects/{projectId}", h.delete)
}

type nameRequest struct {
	Name                string `json:"name"`
	AutomaticScheduling *bool  `json:"automaticScheduling,omitempty"`
	ProjectBuffer       *int   `json:"projectBuffer,omitempty"`
}
type statusRequest struct {
	Status string `json:"status"`
}
type priorityRequest struct {
	Direction string `json:"direction"`
}
type settingsRequest struct {
	AutomaticScheduling *bool `json:"automaticScheduling"`
	ProjectBuffer       *int  `json:"projectBuffer"`
}
type item struct {
	ID                       string     `json:"id"`
	Name                     string     `json:"name"`
	Status                   string     `json:"status"`
	StartDate                *string    `json:"startDate"`
	EndDate                  *string    `json:"endDate"`
	AutoCalculateDate        bool       `json:"autoCalculateDate"`
	AutoDependencyByAssignee bool       `json:"autoDependencyByAssignee"`
	AutomaticScheduling      bool       `json:"automaticScheduling"`
	ProjectBuffer            int        `json:"projectBuffer"`
	ProjectPriority          int        `json:"projectPriority"`
	ClosedAt                 *time.Time `json:"closedAt"`
	CreatedAt                time.Time  `json:"createdAt"`
	UpdatedAt                time.Time  `json:"updatedAt"`
}
type itemResponse struct {
	Data item `json:"data"`
}
type listResponse struct {
	Data     []item `json:"data"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Total    int64  `json:"total"`
}
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	query, err := parseListQuery(r)
	if err != nil {
		writeJSON(w, 400, errorResponse{"INVALID_REQUEST", err.Error(), ""})
		return
	}
	result, err := h.service.List(r.Context(), query)
	if err != nil {
		writeError(w, err)
		return
	}
	data := make([]item, 0, len(result.Items))
	for _, value := range result.Items {
		data = append(data, mapItem(value))
	}
	writeJSON(w, 200, listResponse{data, result.Page, result.PageSize, result.Total})
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.Get(r.Context(), r.PathValue("projectId"))
	h.writeProject(w, value, err, 200)
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var payload nameRequest
	if !decode(w, r, &payload) {
		return
	}
	automatic, buffer := settingsOrDefaults(payload.AutomaticScheduling, payload.ProjectBuffer)
	value, err := h.service.Create(r.Context(), payload.Name, automatic, buffer)
	h.writeProject(w, value, err, 201)
}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var payload nameRequest
	if !decode(w, r, &payload) {
		return
	}
	if payload.AutomaticScheduling == nil || payload.ProjectBuffer == nil {
		writeJSON(w, 400, errorResponse{"INVALID_REQUEST", "automaticScheduling and projectBuffer are required", ""})
		return
	}
	value, err := h.service.Update(r.Context(), r.PathValue("projectId"), payload.Name, *payload.AutomaticScheduling, *payload.ProjectBuffer)
	h.writeProject(w, value, err, 200)
}
func settingsOrDefaults(automatic *bool, buffer *int) (bool, int) {
	automaticValue, bufferValue := true, domain.DefaultProjectBuffer
	if automatic != nil {
		automaticValue = *automatic
	}
	if buffer != nil {
		bufferValue = *buffer
	}
	return automaticValue, bufferValue
}
func (h *Handler) changeStatus(w http.ResponseWriter, r *http.Request) {
	var payload statusRequest
	if !decode(w, r, &payload) {
		return
	}
	status, err := domain.ParseStatus(payload.Status)
	if err != nil {
		writeError(w, err)
		return
	}
	value, err := h.service.ChangeStatus(r.Context(), r.PathValue("projectId"), status)
	h.writeProject(w, value, err, 200)
}
func (h *Handler) movePriority(w http.ResponseWriter, r *http.Request) {
	var payload priorityRequest
	if !decode(w, r, &payload) {
		return
	}
	direction, err := domain.ParsePriorityDirection(payload.Direction)
	if err != nil {
		writeError(w, err)
		return
	}
	value, err := h.service.MovePriority(r.Context(), r.PathValue("projectId"), direction)
	h.writeProject(w, value, err, 200)
}
func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	var payload settingsRequest
	if !decode(w, r, &payload) {
		return
	}
	if payload.AutomaticScheduling == nil || payload.ProjectBuffer == nil {
		writeJSON(w, 400, errorResponse{"INVALID_REQUEST", "automaticScheduling and projectBuffer are required", ""})
		return
	}
	value, err := h.service.UpdateSettings(r.Context(), r.PathValue("projectId"), *payload.AutomaticScheduling, *payload.ProjectBuffer)
	h.writeProject(w, value, err, 200)
}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("projectId")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) writeProject(w http.ResponseWriter, value *domain.Project, err error, status int) {
	if err != nil {
		writeError(w, err)
		return
	}
	if value == nil {
		writeError(w, errors.New("project service returned nil without error"))
		return
	}
	writeJSON(w, status, itemResponse{mapItem(*value)})
}
func decode(w http.ResponseWriter, r *http.Request, target interface{}) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, 400, errorResponse{"INVALID_REQUEST", "Request body is invalid", ""})
		return false
	}
	var extra interface{}
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		writeJSON(w, 400, errorResponse{"INVALID_REQUEST", "Request body is invalid", ""})
		return false
	}
	return true
}
func parseListQuery(r *http.Request) (listing.Query, error) {
	query := listing.Query{Search: r.URL.Query().Get("search"), Page: 1, PageSize: 5}
	var err error
	if value := r.URL.Query().Get("page"); value != "" {
		query.Page, err = strconv.Atoi(value)
		if err != nil || query.Page < 1 {
			return listing.Query{}, errors.New("page must be a positive integer")
		}
	}
	if value := r.URL.Query().Get("pageSize"); value != "" {
		query.PageSize, err = strconv.Atoi(value)
		if err != nil || query.PageSize < 1 || query.PageSize > 100 {
			return listing.Query{}, errors.New("pageSize must be between 1 and 100")
		}
	}
	return query, nil
}
func mapItem(value domain.Project) item {
	var closedAt *time.Time
	if value.ClosedAt != nil {
		utc := value.ClosedAt.UTC()
		closedAt = &utc
	}
	return item{value.ID, value.Name, string(value.Status), dateString(value.StartDate), dateString(value.EndDate), value.AutoCalculateDate, value.AutoDependencyByAssignee, value.AutomaticScheduling, value.ProjectBuffer, value.Priority, closedAt, value.CreatedAt.UTC(), value.UpdatedAt.UTC()}
}
func dateString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format("2006-01-02")
	return &formatted
}
func writeError(w http.ResponseWriter, err error) {
	status, code, message, field := 500, "INTERNAL_ERROR", "An internal error occurred", ""
	switch {
	case errors.Is(err, domain.ErrNameRequired):
		status, code, message, field = 400, "PROJECT_NAME_REQUIRED", domain.ErrNameRequired.Error(), "name"
	case errors.Is(err, domain.ErrNameTooLong):
		status, code, message, field = 400, "PROJECT_NAME_TOO_LONG", domain.ErrNameTooLong.Error(), "name"
	case errors.Is(err, domain.ErrNameExists):
		status, code, message, field = 409, "PROJECT_NAME_ALREADY_EXISTS", domain.ErrNameExists.Error(), "name"
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = 404, "PROJECT_NOT_FOUND", domain.ErrNotFound.Error()
	case errors.Is(err, domain.ErrStatusInvalid):
		status, code, message, field = 400, "PROJECT_STATUS_INVALID", domain.ErrStatusInvalid.Error(), "status"
	case errors.Is(err, domain.ErrStatusTransitionNotAllowed):
		status, code, message, field = 409, "PROJECT_STATUS_TRANSITION_NOT_ALLOWED", domain.ErrStatusTransitionNotAllowed.Error(), "status"
	case errors.Is(err, domain.ErrCannotLockWithoutTasks):
		status, code, message, field = 409, "PROJECT_CANNOT_LOCK_WITHOUT_TASKS", domain.ErrCannotLockWithoutTasks.Error(), "status"
	case errors.Is(err, domain.ErrCannotCloseWithActiveTasks):
		status, code, message, field = 409, "PROJECT_CANNOT_CLOSE_WITH_ACTIVE_TASKS", domain.ErrCannotCloseWithActiveTasks.Error(), "status"
	case errors.Is(err, domain.ErrCannotCloseWithoutTasks):
		status, code, message, field = 409, "PROJECT_CANNOT_CLOSE_WITHOUT_TASKS", domain.ErrCannotCloseWithoutTasks.Error(), "status"
	case errors.Is(err, domain.ErrClosedReadOnly):
		status, code, message = 409, "PROJECT_CLOSED_READ_ONLY", domain.ErrClosedReadOnly.Error()
	case errors.Is(err, domain.ErrHasChildren):
		status, code, message = 409, "PROJECT_HAS_CHILDREN", domain.ErrHasChildren.Error()
	case errors.Is(err, domain.ErrPriorityDirectionInvalid):
		status, code, message, field = 400, "PROJECT_PRIORITY_DIRECTION_INVALID", domain.ErrPriorityDirectionInvalid.Error(), "direction"
	case errors.Is(err, domain.ErrPriorityMoveNotAllowed):
		status, code, message, field = 409, "PROJECT_PRIORITY_MOVE_NOT_ALLOWED", domain.ErrPriorityMoveNotAllowed.Error(), "direction"
	case errors.Is(err, domain.ErrSettingsReadOnly):
		status, code, message = 409, "PROJECT_SETTINGS_READ_ONLY", domain.ErrSettingsReadOnly.Error()
	case errors.Is(err, domain.ErrProjectBufferInvalid):
		status, code, message, field = 400, "PROJECT_BUFFER_INVALID", domain.ErrProjectBufferInvalid.Error(), "projectBuffer"
	}
	writeJSON(w, status, errorResponse{code, message, field})
}
func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
