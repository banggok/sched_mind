package rolehttp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/banggok/sched_mind/backend/internal/roles/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type Service interface {
	List(context.Context, listing.Query) (listing.Page[domain.Role], error)
	Create(context.Context, string) (*domain.Role, error)
	Update(context.Context, string, string) (*domain.Role, error)
	Delete(context.Context, string) error
}

type Handler struct {
	service Service
}

type roleResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type listResponse struct {
	Data     []roleResponse `json:"data"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	Total    int64          `json:"total"`
}

type nameRequest struct {
	Name string `json:"name"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func New(service Service) *Handler {
	return &Handler{service: service}
}

func (handler *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/roles", handler.list)
	mux.HandleFunc("POST /api/roles", handler.create)
	mux.HandleFunc("PUT /api/roles/{roleId}", handler.update)
	mux.HandleFunc("DELETE /api/roles/{roleId}", handler.delete)
}

func (handler *Handler) list(response http.ResponseWriter, request *http.Request) {
	query, err := parseListQuery(request)
	if err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{Code: "INVALID_REQUEST", Message: err.Error()})
		return
	}
	result, err := handler.service.List(request.Context(), query)
	if err != nil {
		writeError(response, err)
		return
	}

	data := make([]roleResponse, 0, len(result.Items))
	for _, role := range result.Items {
		data = append(data, mapRole(role))
	}
	writeJSON(response, http.StatusOK, listResponse{Data: data, Page: result.Page, PageSize: result.PageSize, Total: result.Total})
}

func parseListQuery(request *http.Request) (listing.Query, error) {
	query := listing.Query{Search: request.URL.Query().Get("search"), Page: 1, PageSize: 5}
	var err error
	if value := request.URL.Query().Get("page"); value != "" {
		query.Page, err = strconv.Atoi(value)
		if err != nil || query.Page < 1 {
			return listing.Query{}, errors.New("page must be a positive integer")
		}
	}
	if value := request.URL.Query().Get("pageSize"); value != "" {
		query.PageSize, err = strconv.Atoi(value)
		if err != nil || query.PageSize < 1 || query.PageSize > 100 {
			return listing.Query{}, errors.New("pageSize must be between 1 and 100")
		}
	}
	return query, nil
}

func (handler *Handler) create(response http.ResponseWriter, request *http.Request) {
	var payload nameRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Request body is invalid",
		})
		return
	}

	role, err := handler.service.Create(request.Context(), payload.Name)
	if err != nil {
		writeError(response, err)
		return
	}
	if role == nil {
		writeError(response, errors.New("role service returned nil without error"))
		return
	}
	writeJSON(response, http.StatusCreated, mapRole(*role))
}

func (handler *Handler) update(response http.ResponseWriter, request *http.Request) {
	var payload nameRequest
	if err := decodeJSON(request, &payload); err != nil {
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Code:    "INVALID_REQUEST",
			Message: "Request body is invalid",
		})
		return
	}

	role, err := handler.service.Update(
		request.Context(),
		request.PathValue("roleId"),
		payload.Name,
	)
	if err != nil {
		writeError(response, err)
		return
	}
	if role == nil {
		writeError(response, errors.New("role service returned nil without error"))
		return
	}
	writeJSON(response, http.StatusOK, mapRole(*role))
}

func (handler *Handler) delete(response http.ResponseWriter, request *http.Request) {
	if err := handler.service.Delete(
		request.Context(),
		request.PathValue("roleId"),
	); err != nil {
		writeError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

func decodeJSON(request *http.Request, target any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func mapRole(role domain.Role) roleResponse {
	return roleResponse{
		ID:        role.ID,
		Name:      role.Name,
		CreatedAt: role.CreatedAt,
		UpdatedAt: role.UpdatedAt,
	}
}

func writeError(response http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNameRequired):
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Code: "ROLE_NAME_REQUIRED", Message: domain.ErrNameRequired.Error(), Field: "name",
		})
	case errors.Is(err, domain.ErrNameTooLong):
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Code: "ROLE_NAME_TOO_LONG", Message: domain.ErrNameTooLong.Error(), Field: "name",
		})
	case errors.Is(err, domain.ErrNameInvalid):
		writeJSON(response, http.StatusBadRequest, errorResponse{
			Code: "ROLE_NAME_INVALID", Message: domain.ErrNameInvalid.Error(), Field: "name",
		})
	case errors.Is(err, domain.ErrNameExists):
		writeJSON(response, http.StatusConflict, errorResponse{
			Code: "ROLE_NAME_ALREADY_EXISTS", Message: domain.ErrNameExists.Error(), Field: "name",
		})
	case errors.Is(err, domain.ErrNotFound):
		writeJSON(response, http.StatusNotFound, errorResponse{
			Code: "ROLE_NOT_FOUND", Message: domain.ErrNotFound.Error(),
		})
	case errors.Is(err, domain.ErrInUse):
		writeJSON(response, http.StatusConflict, errorResponse{
			Code: "ROLE_IN_USE", Message: domain.ErrInUse.Error(),
		})
	default:
		writeJSON(response, http.StatusInternalServerError, errorResponse{
			Code: "INTERNAL_ERROR", Message: "An internal error occurred",
		})
	}
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(payload)
}
