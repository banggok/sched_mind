package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/application"
	"github.com/banggok/sched_mind/backend/internal/capacityoverrides/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/httpjson"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type Service interface {
	List(context.Context, string, application.ListQuery) (listing.Page[domain.CapacityOverride], error)
	Get(context.Context, string, string) (*domain.CapacityOverride, error)
	Create(context.Context, string, application.WriteInput) (*domain.CapacityOverride, error)
	Update(context.Context, string, string, application.WriteInput) (*domain.CapacityOverride, error)
	Delete(context.Context, string, string) error
}
type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/team-members/{teamMemberId}/capacity-overrides", h.list)
	mux.HandleFunc("GET /api/team-members/{teamMemberId}/capacity-overrides/{capacityOverrideId}", h.get)
	mux.HandleFunc("POST /api/team-members/{teamMemberId}/capacity-overrides", h.create)
	mux.HandleFunc("PUT /api/team-members/{teamMemberId}/capacity-overrides/{capacityOverrideId}", h.update)
	mux.HandleFunc("DELETE /api/team-members/{teamMemberId}/capacity-overrides/{capacityOverrideId}", h.delete)
}

type writeRequest struct {
	Description string   `json:"description"`
	StartDate   string   `json:"startDate"`
	EndDate     string   `json:"endDate"`
	Capacity    *float64 `json:"capacity"`
}
type item struct {
	ID           string  `json:"id"`
	TeamMemberID string  `json:"teamMemberId"`
	Description  string  `json:"description"`
	StartDate    string  `json:"startDate"`
	EndDate      string  `json:"endDate"`
	Capacity     float64 `json:"capacity"`
	CreatedAt    string  `json:"createdAt"`
	UpdatedAt    string  `json:"updatedAt"`
}
type listResponse struct {
	Data     []item `json:"data"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
	Total    int64  `json:"total"`
}
type itemResponse struct {
	Data item `json:"data"`
}
type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	q, code, field, err := parseQuery(r)
	if err != nil {
		httpjson.Write(w, 400, errorResponse{Code: code, Message: err.Error(), Field: field})
		return
	}
	result, err := h.service.List(r.Context(), r.PathValue("teamMemberId"), q)
	if err != nil {
		writeError(w, err)
		return
	}
	data := make([]item, 0, len(result.Items))
	for _, value := range result.Items {
		data = append(data, mapItem(value))
	}
	httpjson.Write(w, 200, listResponse{data, result.Page, result.PageSize, result.Total})
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.Get(r.Context(), r.PathValue("teamMemberId"), r.PathValue("capacityOverrideId"))
	if err != nil {
		writeError(w, err)
		return
	}
	if value == nil {
		writeError(w, errors.New("capacity override service returned nil"))
		return
	}
	httpjson.Write(w, 200, itemResponse{mapItem(*value)})
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	input, ok := decode(w, r)
	if !ok {
		return
	}
	value, err := h.service.Create(r.Context(), r.PathValue("teamMemberId"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	if value == nil {
		writeError(w, errors.New("capacity override service returned nil"))
		return
	}
	httpjson.Write(w, 201, itemResponse{mapItem(*value)})
}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	input, ok := decode(w, r)
	if !ok {
		return
	}
	value, err := h.service.Update(r.Context(), r.PathValue("teamMemberId"), r.PathValue("capacityOverrideId"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	if value == nil {
		writeError(w, errors.New("capacity override service returned nil"))
		return
	}
	httpjson.Write(w, 200, itemResponse{mapItem(*value)})
}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("teamMemberId"), r.PathValue("capacityOverrideId")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}
func decode(w http.ResponseWriter, r *http.Request) (application.WriteInput, bool) {
	var payload writeRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		httpjson.Write(w, 400, errorResponse{Code: "INVALID_REQUEST", Message: "Request body is invalid"})
		return application.WriteInput{}, false
	}
	if err := ensureEOF(decoder); err != nil {
		httpjson.Write(w, 400, errorResponse{Code: "INVALID_REQUEST", Message: "Request body is invalid"})
		return application.WriteInput{}, false
	}
	start, ok := parseDate(w, payload.StartDate, "startDate", domain.ErrStartDateRequired)
	if !ok {
		return application.WriteInput{}, false
	}
	end, ok := parseDate(w, payload.EndDate, "endDate", domain.ErrEndDateRequired)
	if !ok {
		return application.WriteInput{}, false
	}
	return application.WriteInput{Description: payload.Description, StartDate: start, EndDate: end, Capacity: payload.Capacity}, true
}
func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON value")
	}
	return nil
}
func parseDate(w http.ResponseWriter, value, field string, required error) (time.Time, bool) {
	if value == "" {
		writeError(w, required)
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil || parsed.Format("2006-01-02") != value {
		httpjson.Write(w, 400, errorResponse{Code: "CAPACITY_OVERRIDE_INVALID_DATE", Message: fieldMessage(field) + " must be a valid date in YYYY-MM-DD format", Field: field})
		return time.Time{}, false
	}
	return parsed, true
}
func fieldMessage(field string) string {
	if field == "startDate" {
		return "Start date"
	}
	return "End date"
}
func parseQuery(r *http.Request) (application.ListQuery, string, string, error) {
	q := application.ListQuery{Query: listing.Query{Page: 1, PageSize: 5}}
	if value := r.URL.Query().Get("effectiveDate"); value != "" {
		effectiveDate, err := time.Parse("2006-01-02", value)
		if err != nil || effectiveDate.Format("2006-01-02") != value {
			return application.ListQuery{}, "INVALID_EFFECTIVE_DATE", "effectiveDate",
				errors.New("effectiveDate must be a valid date in YYYY-MM-DD format")
		}
		q.EffectiveDate = &effectiveDate
	}
	if value := r.URL.Query().Get("page"); value != "" {
		page, err := strconv.Atoi(value)
		if err != nil || page < 1 {
			return application.ListQuery{}, "INVALID_PAGE", "page", errors.New("page must be a positive integer")
		}
		q.Page = page
	}
	if value := r.URL.Query().Get("pageSize"); value != "" {
		size, err := strconv.Atoi(value)
		if err != nil || size < 1 || size > 100 {
			return application.ListQuery{}, "INVALID_PAGE_SIZE", "pageSize", errors.New("pageSize must be between 1 and 100")
		}
		q.PageSize = size
	}
	return q, "", "", nil
}
func mapItem(value domain.CapacityOverride) item {
	return item{value.ID, value.TeamMemberID, value.Description, value.StartDate.Format("2006-01-02"), value.EndDate.Format("2006-01-02"), value.Capacity.Hours(), value.CreatedAt.UTC().Format(time.RFC3339), value.UpdatedAt.UTC().Format(time.RFC3339)}
}
func writeError(w http.ResponseWriter, err error) {
	status, code, message, field := 500, "INTERNAL_ERROR", "An internal error occurred", ""
	switch {
	case errors.Is(err, domain.ErrStartDateRequired):
		status, code, message, field = 400, "CAPACITY_OVERRIDE_START_DATE_REQUIRED", err.Error(), "startDate"
	case errors.Is(err, domain.ErrEndDateRequired):
		status, code, message, field = 400, "CAPACITY_OVERRIDE_END_DATE_REQUIRED", err.Error(), "endDate"
	case errors.Is(err, domain.ErrInvalidDateRange):
		status, code, message, field = 400, "CAPACITY_OVERRIDE_INVALID_DATE_RANGE", err.Error(), "endDate"
	case errors.Is(err, domain.ErrCapacityRequired):
		status, code, message, field = 400, "CAPACITY_OVERRIDE_CAPACITY_REQUIRED", err.Error(), "capacity"
	case errors.Is(err, domain.ErrCapacityNegative):
		status, code, message, field = 400, "CAPACITY_OVERRIDE_CAPACITY_NEGATIVE", err.Error(), "capacity"
	case errors.Is(err, domain.ErrCapacityExceedsLimit):
		status, code, message, field = 400, "CAPACITY_OVERRIDE_CAPACITY_EXCEEDS_LIMIT", err.Error(), "capacity"
	case errors.Is(err, domain.ErrCapacityInvalidIncrement):
		status, code, message, field = 400, "CAPACITY_OVERRIDE_CAPACITY_INVALID_INCREMENT", err.Error(), "capacity"
	case errors.Is(err, domain.ErrDescriptionRequired):
		status, code, message, field = 400, "CAPACITY_OVERRIDE_DESCRIPTION_REQUIRED", err.Error(), "description"
	case errors.Is(err, domain.ErrDescriptionTooLong):
		status, code, message, field = 400, "CAPACITY_OVERRIDE_DESCRIPTION_TOO_LONG", err.Error(), "description"
	case errors.Is(err, domain.ErrOverlaps):
		status, code, message = 409, "CAPACITY_OVERRIDE_OVERLAPS", err.Error()
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = 404, "CAPACITY_OVERRIDE_NOT_FOUND", err.Error()
	case errors.Is(err, domain.ErrTeamMemberNotFound):
		status, code, message = 404, "TEAM_MEMBER_NOT_FOUND", err.Error()
	}
	httpjson.Write(w, status, errorResponse{code, message, field})
}
