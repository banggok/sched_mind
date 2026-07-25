package rolehttp

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/roles/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
)

type serviceStub struct {
	roles       []domain.Role
	listQuery   listing.Query
	createRole  *domain.Role
	createError error
	updateRole  *domain.Role
	updateError error
	deleteError error
}

func (service *serviceStub) List(_ context.Context, query listing.Query) (listing.Page[domain.Role], error) {
	service.listQuery = query
	return listing.Page[domain.Role]{
		Items: service.roles, Page: query.Page, PageSize: query.PageSize, Total: int64(len(service.roles)),
	}, nil
}

func (service *serviceStub) Create(context.Context, string) (*domain.Role, error) {
	return service.createRole, service.createError
}

func (service *serviceStub) Update(context.Context, string, string) (*domain.Role, error) {
	return service.updateRole, service.updateError
}

func (service *serviceStub) Delete(context.Context, string) error {
	return service.deleteError
}

func TestListRolesResponse(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	service := &serviceStub{roles: []domain.Role{
		domain.RehydrateRole("role-id", "Backend", now, now),
	}}
	router := http.NewServeMux()
	New(service).Register(router)

	request := httptest.NewRequest(http.MethodGet, "/api/roles?search=back&page=2&pageSize=10", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	var payload listResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data) != 1 || payload.Data[0].Name != "Backend" {
		t.Fatalf("data = %#v, want Backend role", payload.Data)
	}
	if service.listQuery != (listing.Query{Search: "back", Page: 2, PageSize: 10}) {
		t.Fatalf("list query = %#v, want parsed search and pagination", service.listQuery)
	}
	if payload.Page != 2 || payload.PageSize != 10 || payload.Total != 1 {
		t.Fatalf("metadata = page %d size %d total %d", payload.Page, payload.PageSize, payload.Total)
	}
}

func TestListRolesRejectsInvalidPagination(t *testing.T) {
	t.Parallel()

	router := http.NewServeMux()
	New(&serviceStub{}).Register(router)
	request := httptest.NewRequest(http.MethodGet, "/api/roles?page=0", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", response.Code)
	}
}

func TestCreateRoleStatusAndErrorContract(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		service    *serviceStub
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name: "created",
			service: &serviceStub{createRole: rolePointer(domain.RehydrateRole(
				"role-id", "Backend", time.Now(), time.Now(),
			))},
			body:       `{"name":"Backend"}`,
			wantStatus: http.StatusCreated,
		},
		{
			name: "validation failure",
			service: &serviceStub{
				createError: domain.ErrNameRequired,
			},
			body:       `{"name":" "}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "ROLE_NAME_REQUIRED",
		},
		{
			name: "duplicate",
			service: &serviceStub{
				createError: domain.ErrNameExists,
			},
			body:       `{"name":"backend"}`,
			wantStatus: http.StatusConflict,
			wantCode:   "ROLE_NAME_ALREADY_EXISTS",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			router := http.NewServeMux()
			New(test.service).Register(router)
			request := httptest.NewRequest(
				http.MethodPost,
				"/api/roles",
				bytes.NewBufferString(test.body),
			)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			if test.wantCode != "" {
				var payload errorResponse
				if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
					t.Fatalf("decode error response: %v", err)
				}
				if payload.Code != test.wantCode {
					t.Fatalf("code = %q, want %q", payload.Code, test.wantCode)
				}
			}
		})
	}
}

func rolePointer(role domain.Role) *domain.Role {
	return &role
}

func TestUpdateAndDeleteFailures(t *testing.T) {
	t.Parallel()

	router := http.NewServeMux()
	New(&serviceStub{
		updateError: domain.ErrNotFound,
		deleteError: domain.ErrInUse,
	}).Register(router)

	updateRequest := httptest.NewRequest(
		http.MethodPut,
		"/api/roles/missing",
		bytes.NewBufferString(`{"name":"QA"}`),
	)
	updateResponse := httptest.NewRecorder()
	router.ServeHTTP(updateResponse, updateRequest)
	if updateResponse.Code != http.StatusNotFound {
		t.Fatalf("update status = %d, want 404", updateResponse.Code)
	}

	deleteRequest := httptest.NewRequest(
		http.MethodDelete,
		"/api/roles/role-id",
		nil,
	)
	deleteResponse := httptest.NewRecorder()
	router.ServeHTTP(deleteResponse, deleteRequest)
	if deleteResponse.Code != http.StatusConflict {
		t.Fatalf("delete status = %d, want 409", deleteResponse.Code)
	}
	var payload errorResponse
	if err := json.NewDecoder(deleteResponse.Body).Decode(&payload); err != nil {
		t.Fatalf("decode delete response: %v", err)
	}
	if payload.Code != "ROLE_IN_USE" {
		t.Fatalf("code = %q, want ROLE_IN_USE", payload.Code)
	}
}

func TestCreateRejectsNilServiceResult(t *testing.T) {
	t.Parallel()

	router := http.NewServeMux()
	New(&serviceStub{}).Register(router)
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/roles",
		bytes.NewBufferString(`{"name":"Backend"}`),
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", response.Code)
	}
}
