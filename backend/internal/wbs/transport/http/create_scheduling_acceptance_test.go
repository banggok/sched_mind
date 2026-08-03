package wbshttp

import (
	"context"
	"encoding/json"
	"errors"
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

type createAcceptanceProjectRecord struct {
	ID                  string
	Status              string
	AutomaticScheduling bool
}

func (createAcceptanceProjectRecord) TableName() string { return "projects" }

type createAcceptanceWBSRecord struct {
	ID, ProjectID, ParentKey, Name, NameKey                                              string
	ParentID                                                                             *string
	Position                                                                             int
	RoleID, AssigneeID                                                                   *string
	EffortMinutes                                                                        *int
	LagDays                                                                              int
	CapacityAllocationPercentage                                                         int `gorm:"default:100"`
	ExecutionStart, ExecutionEnd, CommitmentStart, CommitmentEnd, ActualStart, ActualEnd *time.Time
	ExecutionUnscheduledReason, CommitmentUnscheduledReason                              *string
	CreatedAt, UpdatedAt                                                                 time.Time
}

func (createAcceptanceWBSRecord) TableName() string { return "wbs_nodes" }

type createSchedulerSpy struct {
	scheduleCalls   int
	forecastCalls   int
	invalidateCalls int
}

func (spy *createSchedulerSpy) RecalculateProjectSchedule(context.Context, string) error {
	spy.scheduleCalls++
	return errors.New("scheduler must not run for empty Task create")
}

func (spy *createSchedulerSpy) RecalculateProjectForecast(context.Context, string) error {
	spy.forecastCalls++
	return errors.New("forecast must not run for Task create")
}

func (spy *createSchedulerSpy) InvalidatePortfolio(context.Context, []string) error {
	spy.invalidateCalls++
	return errors.New("portfolio invalidation must not run for empty Task create")
}

func TestCreateTaskAcceptanceDefaultsCapacityPercentageAndSkipsUnneededScheduler_US63_AC1_US6_AC29_US4_AC23(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&createAcceptanceProjectRecord{}, &createAcceptanceWBSRecord{}); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 7, 31, 9, 0, 0, 0, time.UTC)
	actualEnd := time.Date(2026, 7, 30, 0, 0, 0, 0, time.UTC)
	effort := 480
	assigneeID := "member"
	if err := database.Create(&createAcceptanceProjectRecord{
		ID:                  "project",
		Status:              "open",
		AutomaticScheduling: true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&createAcceptanceWBSRecord{
		ID:              "completed",
		ProjectID:       "project",
		ParentKey:       "",
		Name:            "Completed Task",
		NameKey:         "completed task",
		Position:        1,
		AssigneeID:      &assigneeID,
		EffortMinutes:   &effort,
		ExecutionStart:  &actualEnd,
		ExecutionEnd:    &actualEnd,
		CommitmentStart: &actualEnd,
		CommitmentEnd:   &actualEnd,
		ActualStart:     &actualEnd,
		ActualEnd:       &actualEnd,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	scheduler := &createSchedulerSpy{}
	service := application.NewServiceWithDependencies(
		wbsgormrepo.New(database),
		scheduler,
		func() time.Time { return now.Add(time.Hour) },
		func() (string, error) { return "new-task", nil },
	)
	mux := http.NewServeMux()
	New(service).Register(mux)

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/project/wbs",
		strings.NewReader(`{"name":"New Task","confirmConversion":false}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if scheduler.scheduleCalls != 0 || scheduler.forecastCalls != 0 || scheduler.invalidateCalls != 0 {
		t.Fatalf(
			"schedule=%d forecast=%d invalidate=%d, want all 0",
			scheduler.scheduleCalls,
			scheduler.forecastCalls,
			scheduler.invalidateCalls,
		)
	}

	var payload struct {
		Data struct {
			ID         string `json:"id"`
			Name       string `json:"name"`
			Executable struct {
				ExecutionTimeline            timelineItem `json:"executionTimeline"`
				CommitmentTimeline           timelineItem `json:"commitmentTimeline"`
				CapacityAllocationPercentage int          `json:"capacityAllocationPercentage"`
			} `json:"executable"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Data.ID != "new-task" || payload.Data.Name != "New Task" {
		t.Fatalf("response data=%#v", payload.Data)
	}
	if payload.Data.Executable.CapacityAllocationPercentage != 100 {
		t.Fatalf("confirmed percentage=%d, want 100", payload.Data.Executable.CapacityAllocationPercentage)
	}
	if payload.Data.Executable.ExecutionTimeline.Start != nil ||
		payload.Data.Executable.ExecutionTimeline.End != nil ||
		payload.Data.Executable.CommitmentTimeline.Start != nil ||
		payload.Data.Executable.CommitmentTimeline.End != nil {
		t.Fatalf("new Task must have empty generated dates: %#v", payload.Data.Executable)
	}

	var stored createAcceptanceWBSRecord
	if err := database.First(&stored, "id = ?", "new-task").Error; err != nil {
		t.Fatal(err)
	}
	if stored.AssigneeID != nil || stored.EffortMinutes != nil || stored.ExecutionStart != nil || stored.CommitmentStart != nil {
		t.Fatalf("stored new Task must remain empty and unscheduled: %#v", stored)
	}
	if stored.CapacityAllocationPercentage != 100 {
		t.Fatalf("stored percentage=%d, want 100", stored.CapacityAllocationPercentage)
	}

	var completed createAcceptanceWBSRecord
	if err := database.First(&completed, "id = ?", "completed").Error; err != nil {
		t.Fatal(err)
	}
	if completed.ActualEnd == nil || !completed.ActualEnd.Equal(actualEnd) {
		t.Fatalf("completed Task changed: %#v", completed)
	}
}
