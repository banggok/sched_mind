package gormrepo

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	schedulingapplication "github.com/banggok/sched_mind/backend/internal/scheduling/application"
	wbsapplication "github.com/banggok/sched_mind/backend/internal/wbs/application"
	wbsgormrepo "github.com/banggok/sched_mind/backend/internal/wbs/infrastructure/gormrepo"
	wbshttp "github.com/banggok/sched_mind/backend/internal/wbs/transport/http"
	"gorm.io/gorm"
)

func TestAssigneeRecommendationHTTPAcceptanceRanksFromConcreteSchedulerWithoutSideEffectsAndPreservesPercentageOnSave_US65_AC1_AC4_AC6_AC12_AC13_AC14_AC15_AC25_AC29_AC30_AC33(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.FixedZone("APP_TIMEZONE", 7*60*60))
	anchor := mustDate("2026-08-03")
	roleID := "role"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	members := []acceptanceMemberRecord{
		{ID: "a", Name: "Ayu", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
		{ID: "b", Name: "Bima", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&members).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceProjectRecord{
		ID: "project", Name: "Project", NameKey: "project", Status: "open", Priority: 1,
		AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor,
		ProjectBuffer: 0, ScheduleVersion: 7, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	higherEffort, taskEffort, lowerEffort := 720, 480, 960
	oldStart, oldEnd := mustDate("2026-08-04"), mustDate("2026-08-05")
	tasks := []acceptanceTaskRecord{
		{
			ID: "higher", ProjectID: "project", ParentKey: "", Name: "Higher", NameKey: "higher",
			Position: 1, RoleID: &roleID, AssigneeID: stringPointer("a"), EffortMinutes: &higherEffort,
			CapacityAllocationPercentage: 100, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "task", ProjectID: "project", ParentKey: "", Name: "Candidate", NameKey: "candidate",
			Position: 2, RoleID: &roleID, AssigneeID: stringPointer("a"), EffortMinutes: &taskEffort,
			CapacityAllocationPercentage: 40,
			ExecutionStart:               &oldStart, ExecutionEnd: &oldEnd, CommitmentStart: &oldStart, CommitmentEnd: &oldEnd,
			CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: "lower", ProjectID: "project", ParentKey: "", Name: "Lower", NameKey: "lower",
			Position: 3, RoleID: &roleID, AssigneeID: stringPointer("b"), EffortMinutes: &lowerEffort,
			CapacityAllocationPercentage: 100, CreatedAt: now, UpdatedAt: now,
		},
	}
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceAllocationRecord{
		TaskID: "task", AssigneeID: "a", Timeline: "execution", AllocationDate: oldStart,
		AllocatedMinutes: "240", RemainingCapacityMinutes: "240", Sequence: 1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	scheduler := schedulingapplication.NewService(NewWithDependencies(
		database,
		func() time.Time { return now },
		func() (string, error) { return "generated", nil },
	))
	wbsService := wbsapplication.NewServiceWithDependencies(
		wbsgormrepo.New(database),
		scheduler,
		func() time.Time { return now },
		func() (string, error) { return "unused", nil },
	)
	mux := http.NewServeMux()
	wbshttp.New(wbsService).Register(mux)

	recommendationRequest := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/project/wbs/task/assignee-recommendations",
		strings.NewReader(`{"roleId":"role","effortHours":8,"capacityAllocationPercentage":40,"lag":0}`),
	)
	recommendationRequest.Header.Set("Content-Type", "application/json")
	recommendationResponse := httptest.NewRecorder()
	mux.ServeHTTP(recommendationResponse, recommendationRequest)
	if recommendationResponse.Code != http.StatusOK {
		t.Fatalf("recommendation status=%d body=%s", recommendationResponse.Code, recommendationResponse.Body.String())
	}
	var recommendationPayload struct {
		Data struct {
			CalculatedOnDate string `json:"calculatedOnDate"`
			Snapshot         struct {
				ProjectScheduleVersions map[string]int64 `json:"projectScheduleVersions"`
			} `json:"snapshot"`
			Mode  string `json:"mode"`
			Items []struct {
				MemberID                        string  `json:"memberId"`
				ExecutionEnd                    *string `json:"executionEnd"`
				RemainingExecutionCapacityHours float64 `json:"remainingExecutionCapacityHours"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recommendationResponse.Body.Bytes(), &recommendationPayload); err != nil {
		t.Fatal(err)
	}
	result := recommendationPayload.Data
	if result.CalculatedOnDate != "2026-08-05" || result.Mode != "automatic" || result.Snapshot.ProjectScheduleVersions["project"] != 7 {
		t.Fatalf("metadata=%#v", result)
	}
	if len(result.Items) != 2 || result.Items[0].MemberID != "b" || result.Items[1].MemberID != "a" {
		t.Fatalf("ranked items=%#v", result.Items)
	}
	if result.Items[0].ExecutionEnd == nil || *result.Items[0].ExecutionEnd != "2026-08-05" || result.Items[0].RemainingExecutionCapacityHours != 6 {
		t.Fatalf("Bima recommendation=%#v", result.Items[0])
	}
	if result.Items[1].ExecutionEnd == nil || *result.Items[1].ExecutionEnd != "2026-08-06" || result.Items[1].RemainingExecutionCapacityHours != 6 {
		t.Fatalf("Ayu recommendation=%#v", result.Items[1])
	}
	assertRecommendationAcceptanceState(t, database, "a", 40, oldStart, oldEnd, 7, 1)

	assignRequest := httptest.NewRequest(
		http.MethodPut,
		"/api/projects/project/wbs/task/executable",
		strings.NewReader(`{"roleId":"role","assigneeId":"b","effortHours":8,"lag":0}`),
	)
	assignRequest.Header.Set("Content-Type", "application/json")
	assignResponse := httptest.NewRecorder()
	mux.ServeHTTP(assignResponse, assignRequest)
	if assignResponse.Code != http.StatusOK {
		t.Fatalf("assign status=%d body=%s", assignResponse.Code, assignResponse.Body.String())
	}
	var assigned acceptanceTaskRecord
	if err := database.First(&assigned, "id = ?", "task").Error; err != nil {
		t.Fatal(err)
	}
	if assigned.AssigneeID == nil || *assigned.AssigneeID != "b" || assigned.CapacityAllocationPercentage != 40 {
		t.Fatalf("assigned Task=%#v", assigned)
	}

	clearRequest := httptest.NewRequest(
		http.MethodPut,
		"/api/projects/project/wbs/task/executable",
		strings.NewReader(`{"roleId":"role","assigneeId":null,"effortHours":8,"lag":0}`),
	)
	clearRequest.Header.Set("Content-Type", "application/json")
	clearResponse := httptest.NewRecorder()
	mux.ServeHTTP(clearResponse, clearRequest)
	if clearResponse.Code != http.StatusOK {
		t.Fatalf("clear status=%d body=%s", clearResponse.Code, clearResponse.Body.String())
	}
	var cleared acceptanceTaskRecord
	if err := database.First(&cleared, "id = ?", "task").Error; err != nil {
		t.Fatal(err)
	}
	if cleared.AssigneeID != nil || cleared.CapacityAllocationPercentage != 40 {
		t.Fatalf("cleared Task=%#v", cleared)
	}
}

func TestAssigneeRecommendationManualHTTPAcceptanceUsesAdvisoryAnchorAndKeepsManualDates_US65_AC5_AC8_AC9_AC10_AC25_AC34(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 5, 23, 45, 0, 0, time.FixedZone("APP_TIMEZONE", 7*60*60))
	projectStart := mustDate("2026-08-10")
	manualStart := mustDate("2026-07-01")
	manualEnd := mustDate("2026-12-31")
	roleID := "role"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceMemberRecord{
		ID: "member", Name: "Ayu", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceProjectRecord{
		ID: "project", Name: "Project", NameKey: "project", Status: "open", Priority: 1,
		AutomaticScheduling: false, AutoCalculateDate: true, SchedulingStartDate: &projectStart,
		ProjectBuffer: 0, ScheduleVersion: 3, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	effort := 480
	if err := database.Create(&acceptanceTaskRecord{
		ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", Position: 1,
		RoleID: &roleID, AssigneeID: stringPointer("member"), EffortMinutes: &effort,
		CapacityAllocationPercentage: 100,
		ExecutionStart:               &manualStart, ExecutionEnd: &manualEnd, CommitmentStart: &manualStart, CommitmentEnd: &manualEnd,
		CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	mux := assigneeRecommendationAcceptanceMux(database, now)
	response := performRecommendationAcceptanceRequest(t, mux, `{"roleId":"role","effortHours":8,"capacityAllocationPercentage":100,"lag":0,"executionEnd":"2026-12-31"}`)
	if response.Mode != "manual-advisory" || response.CalculatedOnDate != "2026-08-05" || len(response.Items) != 1 {
		t.Fatalf("manual metadata=%#v", response)
	}
	if response.Items[0].ExecutionEnd == nil || *response.Items[0].ExecutionEnd != "2026-08-10" {
		t.Fatalf("Project Start advisory item=%#v", response.Items[0])
	}

	response = performRecommendationAcceptanceRequest(t, mux, `{"roleId":"role","effortHours":8,"capacityAllocationPercentage":100,"lag":0,"executionStart":"2026-08-12"}`)
	if response.Items[0].ExecutionEnd == nil || *response.Items[0].ExecutionEnd != "2026-08-12" {
		t.Fatalf("explicit manual Start item=%#v", response.Items[0])
	}
	if err := database.Model(&acceptanceProjectRecord{}).Where("id = ?", "project").Update("scheduling_start_date", nil).Error; err != nil {
		t.Fatal(err)
	}
	response = performRecommendationAcceptanceRequest(t, mux, `{"roleId":"role","effortHours":8,"capacityAllocationPercentage":100,"lag":0}`)
	if response.Items[0].ExecutionEnd == nil || *response.Items[0].ExecutionEnd != "2026-08-05" {
		t.Fatalf("application today fallback item=%#v", response.Items[0])
	}
	pastProjectStart := mustDate("2026-08-01")
	if err := database.Model(&acceptanceProjectRecord{}).Where("id = ?", "project").Update("scheduling_start_date", pastProjectStart).Error; err != nil {
		t.Fatal(err)
	}
	response = performRecommendationAcceptanceRequest(t, mux, `{"roleId":"role","effortHours":8,"capacityAllocationPercentage":100,"lag":0}`)
	if response.Items[0].ExecutionEnd == nil || *response.Items[0].ExecutionEnd != "2026-08-05" {
		t.Fatalf("past Project Start fallback item=%#v", response.Items[0])
	}
	stored := loadAcceptanceTask(t, database, "task")
	if stored.ExecutionStart == nil || !stored.ExecutionStart.Equal(manualStart) || stored.ExecutionEnd == nil || !stored.ExecutionEnd.Equal(manualEnd) {
		t.Fatalf("manual dates changed=%#v", stored)
	}
	var project acceptanceProjectRecord
	if err := database.First(&project, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	if project.ScheduleVersion != 3 || project.SchedulingStartDate == nil || !project.SchedulingStartDate.Equal(pastProjectStart) {
		t.Fatalf("manual Project changed=%#v", project)
	}
}

func TestAssigneeRecommendationClosedProjectHTTPAcceptanceRejectsLifecycle_US65_AC1(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	roleID := "role"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceMemberRecord{
		ID: "member", Name: "Ayu", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceProjectRecord{
		ID: "project", Name: "Project", NameKey: "project", Status: "closed", Priority: 1,
		AutomaticScheduling: true, AutoCalculateDate: true, ScheduleVersion: 4, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	effort := 480
	if err := database.Create(&acceptanceTaskRecord{
		ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", Position: 1,
		RoleID: &roleID, EffortMinutes: &effort, CapacityAllocationPercentage: 100, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/project/wbs/task/assignee-recommendations",
		strings.NewReader(`{"roleId":"role","effortHours":8,"capacityAllocationPercentage":100,"lag":0}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	assigneeRecommendationAcceptanceMux(database, now).ServeHTTP(response, request)
	if response.Code != http.StatusConflict || !strings.Contains(response.Body.String(), "TASK_NOT_RECOMMENDABLE") {
		t.Fatalf("closed Project status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestAssigneeRecommendationAutomaticMissingAnchorHTTPAcceptanceReturnsPrerequisiteWithoutToday_US65_AC7_AC23_AC33(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.FixedZone("APP_TIMEZONE", 7*60*60))
	roleID := "role"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceMemberRecord{
		ID: "member", Name: "Ayu", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceProjectRecord{
		ID: "project", Name: "Project", NameKey: "project", Status: "open", Priority: 1,
		AutomaticScheduling: true, AutoCalculateDate: true, ProjectBuffer: 0, ScheduleVersion: 4, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	effort := 480
	if err := database.Create(&acceptanceTaskRecord{
		ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", Position: 1,
		RoleID: &roleID, EffortMinutes: &effort, CapacityAllocationPercentage: 100, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	response := performRecommendationAcceptanceRequest(
		t,
		assigneeRecommendationAcceptanceMux(database, now),
		`{"roleId":"role","effortHours":8,"capacityAllocationPercentage":100,"lag":0}`,
	)
	if response.Mode != "automatic" || len(response.Items) != 1 || response.Items[0].ExecutionEnd != nil || response.Items[0].ReasonCode == nil || *response.Items[0].ReasonCode != recommendationReasonAutomaticAnchorMissing {
		t.Fatalf("missing-anchor response=%#v", response)
	}
	var project acceptanceProjectRecord
	if err := database.First(&project, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	if project.SchedulingStartDate != nil || project.ScheduleVersion != 4 {
		t.Fatalf("automatic missing-anchor Project changed=%#v", project)
	}
}

func TestAssigneeRecommendationRankingHTTPAcceptanceGroupsIncrementalOvercapacityAndCandidateFailure_US65_AC16_AC20_AC21_AC23_AC32(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	candidateAnchor := mustDate("2026-08-04")
	manualAnchor := mustDate("2026-08-03")
	roleID := "role"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	members := []acceptanceMemberRecord{
		{ID: "baseline", Name: "Ayu", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
		{ID: "over", Name: "Bima", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
		{ID: "none", Name: "Dewi", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&members).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceOverrideRecord{
		ID: "none-zero", TeamMemberID: "none", StartDate: candidateAnchor,
		EndDate: candidateAnchor.AddDate(0, 0, maximumScheduleDays), Capacity: "0",
	}).Error; err != nil {
		t.Fatal(err)
	}
	projects := []acceptanceProjectRecord{
		{ID: "project", Name: "Candidate", NameKey: "candidate", Status: "open", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &candidateAnchor, CreatedAt: now, UpdatedAt: now},
		{ID: "manual", Name: "Manual", NameKey: "manual", Status: "open", Priority: 2, AutomaticScheduling: false, AutoCalculateDate: true, SchedulingStartDate: &manualAnchor, CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&projects).Error; err != nil {
		t.Fatal(err)
	}
	baselineEffort, candidateEffort, overFixedEffort := 300, 240, 480
	manualTasks := []acceptanceTaskRecord{
		{ID: "baseline-one", ProjectID: "manual", ParentKey: "", Name: "Baseline One", NameKey: "baseline one", Position: 1, RoleID: &roleID, AssigneeID: stringPointer("baseline"), EffortMinutes: &baselineEffort, CapacityAllocationPercentage: 100, ExecutionStart: &manualAnchor, ExecutionEnd: &manualAnchor, CommitmentStart: &manualAnchor, CommitmentEnd: &manualAnchor, CreatedAt: now, UpdatedAt: now},
		{ID: "baseline-two", ProjectID: "manual", ParentKey: "", Name: "Baseline Two", NameKey: "baseline two", Position: 2, RoleID: &roleID, AssigneeID: stringPointer("baseline"), EffortMinutes: &baselineEffort, CapacityAllocationPercentage: 100, ExecutionStart: &manualAnchor, ExecutionEnd: &manualAnchor, CommitmentStart: &manualAnchor, CommitmentEnd: &manualAnchor, CreatedAt: now, UpdatedAt: now},
		{ID: "over-fixed", ProjectID: "manual", ParentKey: "", Name: "Over Fixed", NameKey: "over fixed", Position: 3, RoleID: &roleID, AssigneeID: stringPointer("over"), EffortMinutes: &overFixedEffort, CapacityAllocationPercentage: 100, ExecutionStart: &candidateAnchor, ExecutionEnd: &candidateAnchor, CommitmentStart: &candidateAnchor, CommitmentEnd: &candidateAnchor, CreatedAt: now, UpdatedAt: now},
		{ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", Position: 1, RoleID: &roleID, EffortMinutes: &candidateEffort, CapacityAllocationPercentage: 100, CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&manualTasks).Error; err != nil {
		t.Fatal(err)
	}

	response := performRecommendationAcceptanceRequest(
		t,
		assigneeRecommendationAcceptanceMux(database, now),
		`{"roleId":"role","effortHours":4,"capacityAllocationPercentage":100,"lag":0}`,
	)
	if len(response.Items) != 3 {
		t.Fatalf("items=%#v", response.Items)
	}
	if response.Items[0].MemberID != "baseline" || response.Items[0].RankGroup != "feasible" || response.Items[0].IncrementalOvercapacityHours != 0 {
		t.Fatalf("baseline-overcapacity candidate=%#v", response.Items[0])
	}
	if response.Items[1].MemberID != "over" || response.Items[1].RankGroup != "overcapacity" || response.Items[1].IncrementalOvercapacityHours != 4 {
		t.Fatalf("incremental-overcapacity candidate=%#v", response.Items[1])
	}
	if response.Items[2].MemberID != "none" || response.Items[2].RankGroup != "no-completion" || response.Items[2].ReasonCode == nil {
		t.Fatalf("candidate-specific failure=%#v", response.Items[2])
	}
}

func TestAssigneeRecommendationTieBreakHTTPAcceptanceUsesRemainingCapacityThenNormalizedNameAndID_US65_AC18_AC22(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)
	anchor := mustDate("2026-08-03")
	roleID := "role"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	members := []acceptanceMemberRecord{
		{ID: "b", Name: "Same", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
		{ID: "a", Name: "same", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
		{ID: "low", Name: "Zed", RoleID: roleID, DailyCapacity: "4", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&members).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceProjectRecord{
		ID: "project", Name: "Project", NameKey: "project", Status: "open", Priority: 1,
		AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	effort := 240
	if err := database.Create(&acceptanceTaskRecord{
		ID: "task", ProjectID: "project", ParentKey: "", Name: "Task", NameKey: "task", Position: 1,
		RoleID: &roleID, EffortMinutes: &effort, CapacityAllocationPercentage: 100, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	response := performRecommendationAcceptanceRequest(
		t,
		assigneeRecommendationAcceptanceMux(database, now),
		`{"roleId":"role","effortHours":4,"capacityAllocationPercentage":100,"lag":0}`,
	)
	if len(response.Items) != 3 || response.Items[0].MemberID != "a" || response.Items[1].MemberID != "b" || response.Items[2].MemberID != "low" {
		t.Fatalf("tie-break order=%#v", response.Items)
	}
	if response.Items[0].RemainingExecutionCapacityHours != 4 || response.Items[1].RemainingExecutionCapacityHours != 4 || response.Items[2].RemainingExecutionCapacityHours != 0 {
		t.Fatalf("remaining capacity tie-break=%#v", response.Items)
	}
}

type assigneeRecommendationAcceptanceResponse struct {
	CalculatedOnDate string `json:"calculatedOnDate"`
	Snapshot         struct {
		ProjectScheduleVersions map[string]int64 `json:"projectScheduleVersions"`
	} `json:"snapshot"`
	Mode  string `json:"mode"`
	Items []struct {
		MemberID                        string  `json:"memberId"`
		RankGroup                       string  `json:"rankGroup"`
		ExecutionEnd                    *string `json:"executionEnd"`
		RemainingExecutionCapacityHours float64 `json:"remainingExecutionCapacityHours"`
		IncrementalOvercapacityHours    float64 `json:"incrementalOvercapacityHours"`
		ReasonCode                      *string `json:"reasonCode"`
	} `json:"items"`
}

func assigneeRecommendationAcceptanceMux(database *gorm.DB, now time.Time) http.Handler {
	scheduler := schedulingapplication.NewService(NewWithDependencies(
		database,
		func() time.Time { return now },
		func() (string, error) { return "generated", nil },
	))
	wbsService := wbsapplication.NewServiceWithDependencies(
		wbsgormrepo.New(database),
		scheduler,
		func() time.Time { return now },
		func() (string, error) { return "unused", nil },
	)
	mux := http.NewServeMux()
	wbshttp.New(wbsService).Register(mux)
	return mux
}

func performRecommendationAcceptanceRequest(t *testing.T, handler http.Handler, body string) assigneeRecommendationAcceptanceResponse {
	t.Helper()
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/project/wbs/task/assignee-recommendations",
		strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("recommendation status=%d body=%s", response.Code, response.Body.String())
	}
	var payload struct {
		Data assigneeRecommendationAcceptanceResponse `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	return payload.Data
}

func assertRecommendationAcceptanceState(
	t *testing.T,
	database *gorm.DB,
	wantAssignee string,
	wantPercentage int,
	wantStart time.Time,
	wantEnd time.Time,
	wantVersion int64,
	wantAllocationCount int64,
) {
	t.Helper()
	var task acceptanceTaskRecord
	if err := database.First(&task, "id = ?", "task").Error; err != nil {
		t.Fatal(err)
	}
	if task.AssigneeID == nil || *task.AssigneeID != wantAssignee || task.CapacityAllocationPercentage != wantPercentage || task.ExecutionStart == nil || !task.ExecutionStart.Equal(wantStart) || task.ExecutionEnd == nil || !task.ExecutionEnd.Equal(wantEnd) {
		t.Fatalf("recommendation mutated Task=%#v", task)
	}
	var project acceptanceProjectRecord
	if err := database.First(&project, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	if project.ScheduleVersion != wantVersion {
		t.Fatalf("schedule version=%d, want %d", project.ScheduleVersion, wantVersion)
	}
	var count int64
	if err := database.Model(&acceptanceAllocationRecord{}).Where("task_id = ?", "task").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != wantAllocationCount {
		t.Fatalf("allocation count=%d, want %d", count, wantAllocationCount)
	}
}
