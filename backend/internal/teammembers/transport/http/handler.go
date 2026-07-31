package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/banggok/sched_mind/backend/internal/shared/httpjson"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/teammembers/application"
	"github.com/banggok/sched_mind/backend/internal/teammembers/domain"
)

type Service interface {
	List(context.Context, listing.Query) (listing.Page[application.TeamMemberRecord], error)
	Get(context.Context, string) (*application.TeamMemberRecord, error)
	Create(context.Context, application.WriteInput) (*application.TeamMemberRecord, error)
	Update(context.Context, string, application.WriteInput) (*application.TeamMemberRecord, error)
	Delete(context.Context, string) error
}

type Handler struct {
	service Service
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/team-members", handler.list)
	mux.HandleFunc("GET /api/team-members/{teamMemberId}", handler.get)
	mux.HandleFunc("POST /api/team-members", handler.create)
	mux.HandleFunc("PUT /api/team-members/{teamMemberId}", handler.update)
	mux.HandleFunc("DELETE /api/team-members/{teamMemberId}", handler.delete)
}

type writeRequest struct {
	Name             string   `json:"name"`
	RoleID           string   `json:"roleId"`
	DailyCapacity    *float64 `json:"dailyCapacity"`
	BufferPercentage *float64 `json:"bufferPercentage"`
}

type roleResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type teamMemberResponse struct {
	ID                    string       `json:"id"`
	Name                  string       `json:"name"`
	Role                  roleResponse `json:"role"`
	DailyCapacity         float64      `json:"dailyCapacity"`
	BufferPercentage      float64      `json:"bufferPercentage"`
	BaseExecutionCapacity float64      `json:"baseExecutionCapacity"`
	CreatedAt             string       `json:"createdAt"`
	UpdatedAt             string       `json:"updatedAt"`
}

type listResponse struct {
	Data     []teamMemberResponse `json:"data"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
	Total    int64                `json:"total"`
}

type itemResponse struct {
	Data teamMemberResponse `json:"data"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func (handler *Handler) list(response http.ResponseWriter, request *http.Request) {
	query, err := listing.ParseHTTPQuery(request)
	if err != nil {
		httpjson.Write(response, http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}
	result, err := handler.service.List(request.Context(), query)
	if err != nil {
		writeError(response, err)
		return
	}
	items := make([]teamMemberResponse, 0, len(result.Items))
	for _, record := range result.Items {
		items = append(items, mapResponse(record))
	}
	httpjson.Write(response, http.StatusOK, listResponse{Data: items, Page: result.Page, PageSize: result.PageSize, Total: result.Total})
}

func (handler *Handler) get(response http.ResponseWriter, request *http.Request) {
	record, err := handler.service.Get(
		request.Context(),
		request.PathValue("teamMemberId"),
	)
	if err != nil {
		writeError(response, err)
		return
	}
	if record == nil {
		writeError(response, errors.New("team member service returned nil without error"))
		return
	}
	httpjson.Write(response, http.StatusOK, itemResponse{Data: mapResponse(*record)})
}

func (handler *Handler) create(response http.ResponseWriter, request *http.Request) {
	input, ok := decodeInput(response, request)
	if !ok {
		return
	}
	record, err := handler.service.Create(request.Context(), input)
	if err != nil {
		writeError(response, err)
		return
	}
	if record == nil {
		writeError(response, errors.New("team member service returned nil without error"))
		return
	}
	httpjson.Write(response, http.StatusCreated, itemResponse{Data: mapResponse(*record)})
}

func (handler *Handler) update(response http.ResponseWriter, request *http.Request) {
	input, ok := decodeInput(response, request)
	if !ok {
		return
	}
	record, err := handler.service.Update(
		request.Context(),
		request.PathValue("teamMemberId"),
		input,
	)
	if err != nil {
		writeError(response, err)
		return
	}
	if record == nil {
		writeError(response, errors.New("team member service returned nil without error"))
		return
	}
	httpjson.Write(response, http.StatusOK, itemResponse{Data: mapResponse(*record)})
}

func (handler *Handler) delete(response http.ResponseWriter, request *http.Request) {
	if err := handler.service.Delete(
		request.Context(),
		request.PathValue("teamMemberId"),
	); err != nil {
		writeError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func decodeInput(
	response http.ResponseWriter,
	request *http.Request,
) (application.WriteInput, bool) {
	var payload writeRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		httpjson.Write(response, http.StatusBadRequest, errorResponse{
			Code: "INVALID_REQUEST", Message: "Request body is invalid",
		})
		return application.WriteInput{}, false
	}
	return application.WriteInput{
		Name:             payload.Name,
		RoleID:           payload.RoleID,
		DailyCapacity:    payload.DailyCapacity,
		BufferPercentage: payload.BufferPercentage,
	}, true
}

func mapResponse(record application.TeamMemberRecord) teamMemberResponse {
	member := record.Member
	return teamMemberResponse{
		ID:                    member.ID,
		Name:                  member.Name,
		Role:                  roleResponse{ID: member.RoleID, Name: record.RoleName},
		DailyCapacity:         member.DailyCapacity.Hours(),
		BufferPercentage:      member.BufferPercentage.Percentage(),
		BaseExecutionCapacity: member.BaseExecutionCapacity(),
		CreatedAt:             member.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:             member.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func writeError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNameRequired):
		httpjson.Write(response, http.StatusBadRequest, errorResponse{
			Code: "TEAM_MEMBER_NAME_REQUIRED", Message: domain.ErrNameRequired.Error(), Field: "name",
		})
	case errors.Is(err, domain.ErrNameTooLong):
		httpjson.Write(response, http.StatusBadRequest, errorResponse{
			Code: "TEAM_MEMBER_NAME_TOO_LONG", Message: domain.ErrNameTooLong.Error(), Field: "name",
		})
	case errors.Is(err, domain.ErrRoleRequired):
		httpjson.Write(response, http.StatusBadRequest, errorResponse{
			Code: "ROLE_REQUIRED", Message: domain.ErrRoleRequired.Error(), Field: "roleId",
		})
	case errors.Is(err, domain.ErrRoleNotFound):
		httpjson.Write(response, http.StatusNotFound, errorResponse{
			Code: "ROLE_NOT_FOUND", Message: domain.ErrRoleNotFound.Error(), Field: "roleId",
		})
	case errors.Is(err, domain.ErrDailyCapacityRequired):
		httpjson.Write(response, http.StatusBadRequest, errorResponse{
			Code: "DAILY_CAPACITY_REQUIRED", Message: domain.ErrDailyCapacityRequired.Error(), Field: "dailyCapacity",
		})
	case errors.Is(err, domain.ErrDailyCapacityNotPositive):
		httpjson.Write(response, http.StatusBadRequest, errorResponse{
			Code: "DAILY_CAPACITY_NOT_POSITIVE", Message: domain.ErrDailyCapacityNotPositive.Error(), Field: "dailyCapacity",
		})
	case errors.Is(err, domain.ErrDailyCapacityExceedsLimit):
		httpjson.Write(response, http.StatusBadRequest, errorResponse{
			Code: "DAILY_CAPACITY_EXCEEDS_LIMIT", Message: domain.ErrDailyCapacityExceedsLimit.Error(), Field: "dailyCapacity",
		})
	case errors.Is(err, domain.ErrDailyCapacityInvalidIncrement):
		httpjson.Write(response, http.StatusBadRequest, errorResponse{
			Code: "DAILY_CAPACITY_INVALID_INCREMENT", Message: domain.ErrDailyCapacityInvalidIncrement.Error(), Field: "dailyCapacity",
		})
	case errors.Is(err, domain.ErrBufferOutOfRange),
		errors.Is(err, domain.ErrBufferPrecision):
		httpjson.Write(response, http.StatusBadRequest, errorResponse{
			Code: "BUFFER_OUT_OF_RANGE", Message: domain.ErrBufferOutOfRange.Error(), Field: "bufferPercentage",
		})
	case errors.Is(err, domain.ErrNotFound):
		httpjson.Write(response, http.StatusNotFound, errorResponse{
			Code: "TEAM_MEMBER_NOT_FOUND", Message: domain.ErrNotFound.Error(),
		})
	case errors.Is(err, domain.ErrAssignedToTask):
		httpjson.Write(response, http.StatusConflict, errorResponse{
			Code: "TEAM_MEMBER_ASSIGNED_TO_TASK", Message: domain.ErrAssignedToTask.Error(),
		})
	default:
		httpjson.Write(response, http.StatusInternalServerError, errorResponse{
			Code: "INTERNAL_ERROR", Message: "An internal error occurred",
		})
	}
}
