package wbshttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/wbs/application"
	wbsgormrepo "github.com/banggok/sched_mind/backend/internal/wbs/infrastructure/gormrepo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type groupAcceptanceProject struct {
	ID                  string `gorm:"primaryKey"`
	Name                string
	NameKey             string
	Status              string
	AutomaticScheduling bool
	SchedulingStartDate *time.Time
	ProjectBuffer       int
	ScheduleVersion     int64
	UpdatedAt           time.Time
}

func (groupAcceptanceProject) TableName() string { return "projects" }

type groupAcceptanceNode struct {
	ID, ProjectID, ParentKey, Name, NameKey                      string
	ParentID                                                     *string
	Position                                                     int
	RoleID, AssigneeID                                           *string
	EffortMinutes                                                *int
	LagDays                                                      int
	CapacityAllocationPercentage                                 int `gorm:"default:100"`
	ExecutionStart, ExecutionEnd, CommitmentStart, CommitmentEnd *time.Time
	ActualStart, ActualEnd                                       *time.Time
	ExecutionUnscheduledReason, CommitmentUnscheduledReason      *string
	GroupSchedulingSource                                        string `gorm:"default:inherit"`
	GroupAutomaticScheduling                                     *bool
	GroupSchedulingStartDate                                     *time.Time
	GroupLocalStatus                                             string `gorm:"default:open"`
	GroupSchedulingVersion                                       int64
	GroupLockedAutomaticScheduling                               *bool
	GroupLockedSchedulingStartDate                               *time.Time
	CreatedAt, UpdatedAt                                         time.Time
}

func (groupAcceptanceNode) TableName() string { return "wbs_nodes" }

type groupAcceptanceDependency struct {
	ID, BlockingTaskID, BlockedTaskID string
	CreatedAt, UpdatedAt              time.Time
}

func (groupAcceptanceDependency) TableName() string { return "task_dependencies" }

type groupAcceptanceAllocation struct {
	TaskID                   string
	AssigneeID               string
	Timeline                 string
	AllocationDate           time.Time
	AllocatedMinutes         string
	RemainingCapacityMinutes string
	Sequence                 int
}

func (groupAcceptanceAllocation) TableName() string { return "task_schedule_allocations" }

type groupAcceptanceScheduler struct {
	scheduleCalls int
}

func (scheduler *groupAcceptanceScheduler) RecalculateProjectSchedule(context.Context, string) error {
	scheduler.scheduleCalls++
	return nil
}
func (*groupAcceptanceScheduler) RecalculateProjectForecast(context.Context, string) error {
	return nil
}
func (*groupAcceptanceScheduler) InvalidatePortfolio(context.Context, []string) error { return nil }

func TestGroupSchedulingAcceptancePersistsOverrideAndReturnsEffectiveAndInheritedValues_US44_AC5_AC7_AC8_AC12(t *testing.T) {
	database, mux, scheduler := groupAcceptanceSetup(t, true)

	response := groupAcceptanceRequest(
		t,
		mux,
		http.MethodPut,
		"/api/projects/project/wbs/group/scheduling",
		`{"expectedVersion":0,"name":"Platform Stream","schedulingSource":"override","automaticScheduling":false,"schedulingStartDate":null}`,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("override status=%d body=%s", response.Code, response.Body.String())
	}
	if scheduler.scheduleCalls != 1 {
		t.Fatalf("schedule calls=%d, want 1 after effective ON-to-OFF change", scheduler.scheduleCalls)
	}
	var payload struct {
		Data struct {
			Name       string `json:"name"`
			Scheduling struct {
				Version                      int64                     `json:"version"`
				Source                       string                    `json:"source"`
				AutomaticScheduling          *bool                     `json:"automaticScheduling"`
				SchedulingStartDate          *string                   `json:"schedulingStartDate"`
				EffectiveAutomaticScheduling bool                      `json:"effectiveAutomaticScheduling"`
				EffectiveSchedulingStartDate *string                   `json:"effectiveSchedulingStartDate"`
				InheritedAutomaticScheduling bool                      `json:"inheritedAutomaticScheduling"`
				InheritedSchedulingStartDate *string                   `json:"inheritedSchedulingStartDate"`
				InheritedAutomaticSource     struct{ ID, Name string } `json:"inheritedAutomaticSource"`
				InheritedStartDateSource     struct{ ID, Name string } `json:"inheritedStartDateSource"`
			} `json:"scheduling"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.Name != "Platform Stream" {
		t.Fatalf("response name=%q, want atomic Group name update", payload.Data.Name)
	}
	scheduling := payload.Data.Scheduling
	if scheduling.Version != 1 {
		t.Fatalf("version=%d, want 1 after confirmed override", scheduling.Version)
	}
	if scheduling.Source != "override" || scheduling.AutomaticScheduling == nil || *scheduling.AutomaticScheduling {
		t.Fatalf("confirmed scheduling=%+v, want custom OFF", scheduling)
	}
	if scheduling.SchedulingStartDate != nil {
		t.Fatalf("own start=%v, want explicit null/inherit", scheduling.SchedulingStartDate)
	}
	if scheduling.EffectiveAutomaticScheduling || scheduling.EffectiveSchedulingStartDate == nil || *scheduling.EffectiveSchedulingStartDate != "2026-08-10" {
		t.Fatalf("effective scheduling=%+v, want OFF with inherited 10 Aug anchor", scheduling)
	}
	if !scheduling.InheritedAutomaticScheduling || scheduling.InheritedSchedulingStartDate == nil || *scheduling.InheritedSchedulingStartDate != "2026-08-10" {
		t.Fatalf("inherited candidate=%+v, want Project ON/10 Aug", scheduling)
	}
	if scheduling.InheritedAutomaticSource.ID != "project" || scheduling.InheritedStartDateSource.ID != "project" {
		t.Fatalf("inherited owners=%+v/%+v, want Project", scheduling.InheritedAutomaticSource, scheduling.InheritedStartDateSource)
	}

	var stored groupAcceptanceNode
	if err := database.First(&stored, "id = ?", "group").Error; err != nil {
		t.Fatal(err)
	}
	if stored.Name != "Platform Stream" || stored.GroupSchedulingSource != "override" || stored.GroupAutomaticScheduling == nil || *stored.GroupAutomaticScheduling || stored.GroupSchedulingStartDate != nil {
		t.Fatalf("stored atomic Group update=%#v", stored)
	}

	response = groupAcceptanceRequest(
		t,
		mux,
		http.MethodPut,
		"/api/projects/project/wbs/group/scheduling",
		`{"expectedVersion":1,"schedulingSource":"inherit"}`,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("reset status=%d body=%s", response.Code, response.Body.String())
	}
	if scheduler.scheduleCalls != 2 {
		t.Fatalf("schedule calls=%d, want reset to live Project ON to recalculate", scheduler.scheduleCalls)
	}

	response = groupAcceptanceRequest(t, mux, http.MethodGet, "/api/projects/project/wbs", "")
	if response.Code != http.StatusOK {
		t.Fatalf("tree status=%d body=%s", response.Code, response.Body.String())
	}
	var tree struct {
		Data []struct {
			ID         string `json:"id"`
			Scheduling struct {
				Source                       string `json:"source"`
				EffectiveAutomaticScheduling bool   `json:"effectiveAutomaticScheduling"`
			} `json:"scheduling"`
			Children []struct {
				ID         string `json:"id"`
				Scheduling struct {
					EffectiveAutomaticScheduling bool `json:"effectiveAutomaticScheduling"`
				} `json:"scheduling"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &tree); err != nil {
		t.Fatal(err)
	}
	if len(tree.Data) != 1 || tree.Data[0].ID != "group" || tree.Data[0].Scheduling.Source != "inherit" || !tree.Data[0].Scheduling.EffectiveAutomaticScheduling {
		t.Fatalf("confirmed tree=%+v, want Group inherited ON", tree.Data)
	}
	if len(tree.Data[0].Children) != 1 || !tree.Data[0].Children[0].Scheduling.EffectiveAutomaticScheduling {
		t.Fatalf("confirmed child scheduling=%+v, want inherited ON", tree.Data[0].Children)
	}
}

func TestGroupSchedulingAcceptanceRejectsStaleOwnerVersionWithoutOverwrite_US44_AC37(t *testing.T) {
	database, mux, _ := groupAcceptanceSetup(t, true)
	first := groupAcceptanceRequest(
		t, mux, http.MethodPut, "/api/projects/project/wbs/group/scheduling",
		`{"expectedVersion":0,"schedulingSource":"override","automaticScheduling":false,"schedulingStartDate":null}`,
	)
	if first.Code != http.StatusOK {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	stale := groupAcceptanceRequest(
		t, mux, http.MethodPut, "/api/projects/project/wbs/group/scheduling",
		`{"expectedVersion":0,"schedulingSource":"override","automaticScheduling":true,"schedulingStartDate":"2026-09-01"}`,
	)
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), "GROUP_SCHEDULING_STALE") {
		t.Fatalf("stale status=%d body=%s", stale.Code, stale.Body.String())
	}
	var stored groupAcceptanceNode
	if err := database.First(&stored, "id = ?", "group").Error; err != nil {
		t.Fatal(err)
	}
	if stored.GroupSchedulingVersion != 1 || stored.GroupAutomaticScheduling == nil || *stored.GroupAutomaticScheduling || stored.GroupSchedulingStartDate != nil {
		t.Fatalf("stale request overwrote confirmed Group scheduling: %#v", stored)
	}
}

func TestGroupSchedulingAcceptanceRepresentationOnlyOverrideSkipsScheduler_US44_AC8(t *testing.T) {
	_, mux, scheduler := groupAcceptanceSetup(t, true)
	response := groupAcceptanceRequest(
		t,
		mux,
		http.MethodPut,
		"/api/projects/project/wbs/group/scheduling",
		`{"expectedVersion":0,"schedulingSource":"override","automaticScheduling":true,"schedulingStartDate":"2026-08-10"}`,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if scheduler.scheduleCalls != 0 {
		t.Fatalf("representation-only override scheduled %d times, want 0", scheduler.scheduleCalls)
	}
}

func TestGroupLockAcceptanceProtectsPlanningAndAllowsActualDate_US44_AC14_AC20_AC21_AC35(t *testing.T) {
	database, mux, scheduler := groupAcceptanceSetup(t, true)

	response := groupAcceptanceRequest(t, mux, http.MethodPost, "/api/projects/project/wbs/group/status", `{"status":"locked","expectedVersion":0}`)
	if response.Code != http.StatusOK {
		t.Fatalf("lock status=%d body=%s", response.Code, response.Body.String())
	}
	if scheduler.scheduleCalls != 0 {
		t.Fatalf("Group Lock invoked scheduler %d times, want 0", scheduler.scheduleCalls)
	}

	response = groupAcceptanceRequest(t, mux, http.MethodPut, "/api/projects/project/wbs/task", `{"name":"Renamed"}`)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "GROUP_LOCKED_READ_ONLY") {
		t.Fatalf("locked rename status=%d body=%s", response.Code, response.Body.String())
	}
	var stored groupAcceptanceNode
	if err := database.First(&stored, "id = ?", "task").Error; err != nil {
		t.Fatal(err)
	}
	if stored.Name != "Task" {
		t.Fatalf("locked planning mutation changed name=%q", stored.Name)
	}
	baselineStart, baselineEnd := stored.ExecutionStart, stored.ExecutionEnd

	response = groupAcceptanceRequest(t, mux, http.MethodPost, "/api/projects/project/wbs/task/actual-date", `{"actualStart":"2026-08-11","actualEnd":"2026-08-13"}`)
	if response.Code != http.StatusOK {
		t.Fatalf("Actual Date status=%d body=%s", response.Code, response.Body.String())
	}
	if err := database.First(&stored, "id = ?", "task").Error; err != nil {
		t.Fatal(err)
	}
	if stored.ActualStart == nil || stored.ActualEnd == nil {
		t.Fatalf("Actual Date was not persisted: %#v", stored)
	}
	if stored.ExecutionStart == nil || baselineStart == nil || !stored.ExecutionStart.Equal(*baselineStart) || stored.ExecutionEnd == nil || baselineEnd == nil || !stored.ExecutionEnd.Equal(*baselineEnd) {
		t.Fatalf("locked baseline changed after Actual Date: %#v", stored)
	}

	response = groupAcceptanceRequest(t, mux, http.MethodPost, "/api/projects/project/wbs/task/reopen", "")
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "GROUP_LOCKED_READ_ONLY") {
		t.Fatalf("Task Reopen under Group lock status=%d body=%s", response.Code, response.Body.String())
	}
}

func groupAcceptanceSetup(t *testing.T, scheduled bool) (*gorm.DB, http.Handler, *groupAcceptanceScheduler) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&groupAcceptanceProject{}, &groupAcceptanceNode{}, &groupAcceptanceDependency{}, &groupAcceptanceAllocation{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	anchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	if err := database.Create(&groupAcceptanceProject{
		ID: "project", Name: "Alpha", NameKey: "alpha", Status: "open",
		AutomaticScheduling: true, SchedulingStartDate: &anchor, ProjectBuffer: 20,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	groupID := "group"
	task := groupAcceptanceNode{
		ID: "task", ProjectID: "project", ParentID: &groupID, ParentKey: groupID,
		Name: "Task", NameKey: "task", Position: 1, CapacityAllocationPercentage: 100,
		GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now,
	}
	if scheduled {
		start := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
		end := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
		task.ExecutionStart, task.ExecutionEnd = &start, &end
		task.CommitmentStart, task.CommitmentEnd = &start, &end
	}
	if err := database.Create(&[]groupAcceptanceNode{
		{
			ID: groupID, ProjectID: "project", ParentKey: "", Name: "Platform", NameKey: "platform",
			Position: 1, CapacityAllocationPercentage: 100, GroupSchedulingSource: "inherit",
			GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now,
		},
		task,
	}).Error; err != nil {
		t.Fatal(err)
	}
	scheduler := &groupAcceptanceScheduler{}
	service := application.NewServiceWithDependencies(
		wbsgormrepo.New(database),
		scheduler,
		func() time.Time { return now.Add(time.Hour) },
		func() (string, error) { return "generated-id", nil },
	)
	mux := http.NewServeMux()
	New(service).Register(mux)
	return database, mux, scheduler
}

func groupAcceptanceRequest(t *testing.T, mux http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	var request *http.Request
	if body == "" {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	return response
}
