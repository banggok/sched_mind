package http

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/banggok/sched_mind/backend/internal/publicholidays/application"
	"github.com/banggok/sched_mind/backend/internal/publicholidays/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/httpjson"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type Service interface {
	List(context.Context, application.ListQuery) (listing.Page[domain.PublicHoliday], error)
	Get(context.Context, string) (*domain.PublicHoliday, error)
	Create(context.Context, application.WriteInput) (*domain.PublicHoliday, error)
	Update(context.Context, string, application.WriteInput) (*domain.PublicHoliday, error)
	Delete(context.Context, string) error
	CalendarDates(context.Context, time.Time, time.Time) ([]time.Time, error)
}
type Handler struct{ service Service }

func New(service Service) *Handler { return &Handler{service: service} }
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/public-holidays", h.list)
	mux.HandleFunc("GET /api/public-holidays/calendar", h.calendar)
	mux.HandleFunc("GET /api/public-holidays/{publicHolidayId}", h.get)
	mux.HandleFunc("POST /api/public-holidays", h.create)
	mux.HandleFunc("PUT /api/public-holidays/{publicHolidayId}", h.update)
	mux.HandleFunc("DELETE /api/public-holidays/{publicHolidayId}", h.delete)
}

type writeRequest struct {
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Description string `json:"description"`
}
type item struct {
	ID          string `json:"id"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
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
	query, code, field, err := parseQuery(r)
	if err != nil {
		httpjson.Write(w, 400, errorResponse{code, err.Error(), field})
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
	httpjson.Write(w, 200, listResponse{data, result.Page, result.PageSize, result.Total})
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	value, err := h.service.Get(r.Context(), r.PathValue("publicHolidayId"))
	if err != nil {
		writeError(w, err)
		return
	}
	if value == nil {
		writeError(w, errors.New("public holiday service returned nil"))
		return
	}
	httpjson.Write(w, 200, itemResponse{mapItem(*value)})
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	input, ok := decode(w, r)
	if !ok {
		return
	}
	value, err := h.service.Create(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	if value == nil {
		writeError(w, errors.New("public holiday service returned nil"))
		return
	}
	httpjson.Write(w, 201, itemResponse{mapItem(*value)})
}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	input, ok := decode(w, r)
	if !ok {
		return
	}
	value, err := h.service.Update(r.Context(), r.PathValue("publicHolidayId"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	if value == nil {
		writeError(w, errors.New("public holiday service returned nil"))
		return
	}
	httpjson.Write(w, 200, itemResponse{mapItem(*value)})
}
func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.service.Delete(r.Context(), r.PathValue("publicHolidayId")); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) calendar(w http.ResponseWriter, r *http.Request) {
	start, ok := parseDate(w, r.URL.Query().Get("startDate"), "startDate", domain.ErrStartDateRequired)
	if !ok {
		return
	}
	end, ok := parseDate(w, r.URL.Query().Get("endDate"), "endDate", domain.ErrEndDateRequired)
	if !ok {
		return
	}
	if end.Before(start) {
		writeError(w, domain.ErrInvalidDateRange)
		return
	}
	dates, err := h.service.CalendarDates(r.Context(), start, end)
	if err != nil {
		writeError(w, err)
		return
	}
	values := make([]string, 0, len(dates))
	for _, date := range dates {
		values = append(values, date.Format("2006-01-02"))
	}
	httpjson.Write(w, 200, map[string]any{"data": values})
}
func decode(w http.ResponseWriter, r *http.Request) (application.WriteInput, bool) {
	var payload writeRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		httpjson.Write(w, 400, errorResponse{Code: "INVALID_REQUEST", Message: "Request body is invalid"})
		return application.WriteInput{}, false
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
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
	return application.WriteInput{StartDate: start, EndDate: end, Description: payload.Description}, true
}
func parseDate(w http.ResponseWriter, value, field string, required error) (time.Time, bool) {
	if value == "" {
		writeError(w, required)
		return time.Time{}, false
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil || date.Format("2006-01-02") != value {
		httpjson.Write(w, 400, errorResponse{Code: "INVALID_HOLIDAY_DATE", Message: field + " must be a valid date in YYYY-MM-DD format", Field: field})
		return time.Time{}, false
	}
	return date, true
}
func parseQuery(r *http.Request) (application.ListQuery, string, string, error) {
	query := application.ListQuery{Query: listing.Query{Page: 1, PageSize: 5}}
	if value := r.URL.Query().Get("holidayDate"); value != "" {
		date, err := time.Parse("2006-01-02", value)
		if err != nil || date.Format("2006-01-02") != value {
			return application.ListQuery{}, "INVALID_HOLIDAY_DATE", "holidayDate", errors.New("holidayDate must be a valid date in YYYY-MM-DD format")
		}
		query.HolidayDate = &date
	}
	if value := r.URL.Query().Get("page"); value != "" {
		page, err := strconv.Atoi(value)
		if err != nil || page < 1 {
			return application.ListQuery{}, "INVALID_PAGE", "page", errors.New("page must be a positive integer")
		}
		query.Page = page
	}
	if value := r.URL.Query().Get("pageSize"); value != "" {
		size, err := strconv.Atoi(value)
		if err != nil || size < 1 || size > 100 {
			return application.ListQuery{}, "INVALID_PAGE_SIZE", "pageSize", errors.New("pageSize must be between 1 and 100")
		}
		query.PageSize = size
	}
	return query, "", "", nil
}
func mapItem(value domain.PublicHoliday) item {
	return item{value.ID, value.StartDate.Format("2006-01-02"), value.EndDate.Format("2006-01-02"), value.Description, value.CreatedAt.UTC().Format(time.RFC3339), value.UpdatedAt.UTC().Format(time.RFC3339)}
}
func writeError(w http.ResponseWriter, err error) {
	status, code, message, field := 500, "INTERNAL_ERROR", "An internal error occurred", ""
	switch {
	case errors.Is(err, domain.ErrStartDateRequired):
		status, code, message, field = 400, "PUBLIC_HOLIDAY_START_DATE_REQUIRED", err.Error(), "startDate"
	case errors.Is(err, domain.ErrEndDateRequired):
		status, code, message, field = 400, "PUBLIC_HOLIDAY_END_DATE_REQUIRED", err.Error(), "endDate"
	case errors.Is(err, domain.ErrInvalidDateRange):
		status, code, message, field = 400, "PUBLIC_HOLIDAY_INVALID_DATE_RANGE", err.Error(), "endDate"
	case errors.Is(err, domain.ErrNoWorkingDates):
		status, code, message, field = 400, "PUBLIC_HOLIDAY_NO_WORKING_DATES", err.Error(), "startDate"
	case errors.Is(err, domain.ErrDescriptionRequired):
		status, code, message, field = 400, "PUBLIC_HOLIDAY_DESCRIPTION_REQUIRED", err.Error(), "description"
	case errors.Is(err, domain.ErrDescriptionTooLong):
		status, code, message, field = 400, "PUBLIC_HOLIDAY_DESCRIPTION_TOO_LONG", err.Error(), "description"
	case errors.Is(err, domain.ErrDateAlreadyExists):
		status, code, message, field = 409, "PUBLIC_HOLIDAY_DATE_ALREADY_EXISTS", err.Error(), "startDate"
	case errors.Is(err, domain.ErrNotFound):
		status, code, message = 404, "PUBLIC_HOLIDAY_NOT_FOUND", err.Error()
	}
	httpjson.Write(w, status, errorResponse{code, message, field})
}
