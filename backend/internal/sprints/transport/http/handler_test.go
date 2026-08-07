package sprinthttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/sprints/application"
	sprintgormrepo "github.com/banggok/sched_mind/backend/internal/sprints/infrastructure/gormrepo"
	teammemberapp "github.com/banggok/sched_mind/backend/internal/teammembers/application"
	teammembergormrepo "github.com/banggok/sched_mind/backend/internal/teammembers/infrastructure/gormrepo"
	teammemberhttp "github.com/banggok/sched_mind/backend/internal/teammembers/transport/http"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type acceptanceRole struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

func (acceptanceRole) TableName() string { return "roles" }

type acceptanceMember struct {
	ID                                            string `gorm:"primaryKey"`
	Name, RoleID, DailyCapacity, BufferPercentage string
	UpdatedAt                                     time.Time
	DeletedAt                                     gorm.DeletedAt
}

func (acceptanceMember) TableName() string { return "team_members" }

type acceptanceProject struct {
	ID              string `gorm:"primaryKey"`
	Name, Status    string
	Priority        int
	ScheduleVersion int64
}

func (acceptanceProject) TableName() string { return "projects" }

type acceptanceTask struct {
	ID                             string `gorm:"primaryKey"`
	ProjectID, ParentKey, Name     string
	ParentID                       *string
	Position                       int
	AssigneeID                     *string
	EffortMinutes                  *int
	ExecutionStart, ExecutionEnd   *time.Time
	CommitmentStart, CommitmentEnd *time.Time
	ActualStart, ActualEnd         *time.Time
	UpdatedAt                      time.Time
}

func (acceptanceTask) TableName() string { return "wbs_nodes" }

type acceptanceAllocation struct {
	TaskID, AssigneeID, Timeline string    `gorm:"primaryKey"`
	AllocationDate               time.Time `gorm:"primaryKey"`
	AllocatedMinutes             string
}

func (acceptanceAllocation) TableName() string { return "task_schedule_allocations" }

type acceptanceOverride struct {
	ID                 string `gorm:"primaryKey"`
	TeamMemberID       string
	StartDate, EndDate time.Time
	Capacity           string
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt
}

func (acceptanceOverride) TableName() string { return "capacity_overrides" }

type acceptanceHoliday struct {
	PublicHolidayID string
	Date            time.Time
}

func (acceptanceHoliday) TableName() string { return "public_holiday_dates" }

type acceptanceSprint struct {
	ID                   string `gorm:"primaryKey"`
	Name, NameKey        string
	StartDate, EndDate   time.Time
	Status               string
	Version              int64
	StartedAt            *time.Time
	CreatedAt, UpdatedAt time.Time
}

func (acceptanceSprint) TableName() string { return "sprints" }

type acceptanceSprintMember struct {
	SprintID, MemberID string `gorm:"primaryKey"`
}

func (acceptanceSprintMember) TableName() string { return "sprint_members" }

type acceptanceSprintTask struct {
	SprintID, TaskID string `gorm:"primaryKey"`
}

func (acceptanceSprintTask) TableName() string { return "sprint_tasks" }

func acceptanceServer(t *testing.T) (*gorm.DB, *http.ServeMux) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&acceptanceRole{}, &acceptanceMember{}, &acceptanceProject{}, &acceptanceTask{}, &acceptanceAllocation{}, &acceptanceOverride{}, &acceptanceHoliday{}, &acceptanceSprint{}, &acceptanceSprintMember{}, &acceptanceSprintTask{}); err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	if err := database.Create(&acceptanceRole{ID: "role-1", Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceMember{ID: "member-1", Name: "Harry", RoleID: "role-1", DailyCapacity: "8.0", BufferPercentage: "20.0", UpdatedAt: day}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceProject{ID: "project-1", Name: "Alpha", Status: "open", Priority: 1, ScheduleVersion: 7}).Error; err != nil {
		t.Fatal(err)
	}
	memberID := "member-1"
	effortMinutes := 450
	commitmentEnd := day.AddDate(0, 0, 1)
	if err := database.Create(&acceptanceTask{ID: "task-1", ProjectID: "project-1", Name: "API task", Position: 1, AssigneeID: &memberID, EffortMinutes: &effortMinutes, ExecutionStart: &day, ExecutionEnd: &day, CommitmentStart: &day, CommitmentEnd: &commitmentEnd, UpdatedAt: day}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceAllocation{TaskID: "task-1", AssigneeID: memberID, Timeline: "execution", AllocationDate: day, AllocatedMinutes: "120"}).Error; err != nil {
		t.Fatal(err)
	}
	repository := sprintgormrepo.New(database)
	service := application.NewService(repository)
	mux := http.NewServeMux()
	New(service).Register(mux)
	return database, mux
}

func TestHTTPMemberDeleteRetainsSprintTaskAsNeedsReview_AC57(t *testing.T) {
	database, handler := acceptanceServer(t)
	created := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":"Member cleanup","startDate":"2026-08-04","endDate":"2026-08-04","memberIds":["member-1"],"taskIds":["task-1"]}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	sprintID := decodeDataID(t, created)
	if err := database.Model(&acceptanceProject{}).Where("id = ?", "project-1").Update("status", "closed").Error; err != nil {
		t.Fatal(err)
	}
	teammemberhttp.New(teammemberapp.NewService(teammembergormrepo.New(database))).Register(handler)

	deleted := performJSON(t, handler, http.MethodDelete, "/api/team-members/member-1", "")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("member delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	var memberRelations, taskRelations int64
	if err := database.Model(&acceptanceSprintMember{}).Where("sprint_id = ? AND member_id = ?", sprintID, "member-1").Count(&memberRelations).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&acceptanceSprintTask{}).Where("sprint_id = ? AND task_id = ?", sprintID, "task-1").Count(&taskRelations).Error; err != nil {
		t.Fatal(err)
	}
	if memberRelations != 0 || taskRelations != 1 {
		t.Fatalf("member relations=%d task relations=%d", memberRelations, taskRelations)
	}

	detail := performJSON(t, handler, http.MethodGet, "/api/sprints/"+sprintID, "")
	if detail.Code != http.StatusOK ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"members":[]`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"id":"task-1"`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"parentName":"Alpha"`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"Assignee is not included in this Sprint."`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"Project is Closed."`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"selectedMemberAllocationMinutes":0`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"needsReviewAllocationMinutes":120`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"allTaskInSprintMinutes":120`)) {
		t.Fatalf("Sprint detail status=%d body=%s", detail.Code, detail.Body.String())
	}
}

func performJSON(t *testing.T, handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeDataID(t *testing.T, response *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Data struct {
			ID      string `json:"id"`
			Status  string `json:"status"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response %s: %v", response.Body.String(), err)
	}
	if payload.Data.ID == "" {
		t.Fatalf("response has no ID: %s", response.Body.String())
	}
	return payload.Data.ID
}

func TestHTTPCreateDetailStartAndDeletePreservesOwningEntities_SPD17_AC45To47And66To72(t *testing.T) {
	database, handler := acceptanceServer(t)
	create := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":" Sprint Alpha ","startDate":"2026-08-04","endDate":"2026-08-04","memberIds":["member-1"],"taskIds":["task-1"]}`)
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	sprintID := decodeDataID(t, create)
	if !bytes.Contains(create.Body.Bytes(), []byte(`"status":"planned"`)) || !bytes.Contains(create.Body.Bytes(), []byte(`"version":1`)) {
		t.Fatalf("unexpected create response: %s", create.Body.String())
	}

	detail := performJSON(t, handler, http.MethodGet, "/api/sprints/"+sprintID, "")
	if detail.Code != http.StatusOK ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"capacityMinutes":390`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"inSprintAllocationMinutes":120`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"parentName":"Alpha"`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"effortMinutes":450`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"executionStart":"2026-08-04"`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"executionEnd":"2026-08-04"`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"commitmentStart":"2026-08-04"`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"commitmentEnd":"2026-08-05"`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"remainingMinutes":270`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"wbsPath":"1"`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"dailyPlanOrderDate":"2026-08-04"`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"dailySummaries":[{"date":"2026-08-04","capacityMinutes":390,"selectedAllocationMinutes":120,"remainingMinutes":270,"overcapacityMinutes":0}]`)) {
		t.Fatalf("detail status=%d body=%s", detail.Code, detail.Body.String())
	}

	staleStart := performJSON(t, handler, http.MethodPost, "/api/sprints/"+sprintID+"/start", `{"version":9}`)
	if staleStart.Code != http.StatusConflict || !bytes.Contains(staleStart.Body.Bytes(), []byte(`"code":"SPRINT_VERSION_CONFLICT"`)) {
		t.Fatalf("stale start status=%d body=%s", staleStart.Code, staleStart.Body.String())
	}
	start := performJSON(t, handler, http.MethodPost, "/api/sprints/"+sprintID+"/start", `{"version":1}`)
	if start.Code != http.StatusOK || !bytes.Contains(start.Body.Bytes(), []byte(`"status":"started"`)) || !bytes.Contains(start.Body.Bytes(), []byte(`"version":2`)) {
		t.Fatalf("start status=%d body=%s", start.Code, start.Body.String())
	}

	deleted := performJSON(t, handler, http.MethodDelete, "/api/sprints/"+sprintID+"?version=2", "")
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status=%d body=%s", deleted.Code, deleted.Body.String())
	}
	var sprintCount, memberRelationCount, taskRelationCount, memberCount, taskCount int64
	for model, count := range map[any]*int64{&acceptanceSprint{}: &sprintCount, &acceptanceSprintMember{}: &memberRelationCount, &acceptanceSprintTask{}: &taskRelationCount, &acceptanceMember{}: &memberCount, &acceptanceTask{}: &taskCount} {
		if err := database.Model(model).Count(count).Error; err != nil {
			t.Fatal(err)
		}
	}
	if sprintCount != 0 || memberRelationCount != 0 || taskRelationCount != 0 || memberCount != 1 || taskCount != 1 {
		t.Fatalf("unexpected state sprint=%d memberRelations=%d taskRelations=%d members=%d tasks=%d", sprintCount, memberRelationCount, taskRelationCount, memberCount, taskCount)
	}
}

func TestHTTPRejectsInvalidDatesOverlapAndUnscheduledTaskWithoutPartialWrite_AC8And12And41And59(t *testing.T) {
	database, handler := acceptanceServer(t)
	invalidDate := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":"Bad","startDate":"2026-08-05","endDate":"2026-08-04","memberIds":["member-1"]}`)
	if invalidDate.Code != http.StatusBadRequest || !bytes.Contains(invalidDate.Body.Bytes(), []byte(`"code":"SPRINT_DATE_RANGE_INVALID"`)) {
		t.Fatalf("invalid date status=%d body=%s", invalidDate.Code, invalidDate.Body.String())
	}

	first := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":"First","startDate":"2026-08-04","endDate":"2026-08-10","memberIds":["member-1"],"taskIds":["task-1"]}`)
	if first.Code != http.StatusCreated {
		t.Fatalf("first status=%d body=%s", first.Code, first.Body.String())
	}
	firstID := decodeDataID(t, first)
	overlap := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":"Second","startDate":"2026-08-10","endDate":"2026-08-12","memberIds":["member-1"],"taskIds":[]}`)
	if overlap.Code != http.StatusConflict ||
		!bytes.Contains(overlap.Body.Bytes(), []byte(`"code":"SPRINT_MEMBER_OVERLAP"`)) ||
		!bytes.Contains(overlap.Body.Bytes(), []byte(`"sprintId":"`+firstID+`"`)) ||
		!bytes.Contains(overlap.Body.Bytes(), []byte(`"sprintName":"First"`)) ||
		!bytes.Contains(overlap.Body.Bytes(), []byte(`"startDate":"2026-08-04"`)) ||
		!bytes.Contains(overlap.Body.Bytes(), []byte(`"endDate":"2026-08-10"`)) ||
		!bytes.Contains(overlap.Body.Bytes(), []byte(`"id":"member-1","name":"Harry"`)) {
		t.Fatalf("overlap status=%d body=%s", overlap.Code, overlap.Body.String())
	}

	if err := database.Model(&acceptanceTask{}).Where("id = ?", "task-1").Updates(map[string]any{"execution_start": nil, "execution_end": nil}).Error; err != nil {
		t.Fatal(err)
	}
	unscheduled := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":"Unscheduled","startDate":"2026-08-20","endDate":"2026-08-21","memberIds":["member-1"],"taskIds":["task-1"]}`)
	if unscheduled.Code != http.StatusConflict || !bytes.Contains(unscheduled.Body.Bytes(), []byte(`"code":"SPRINT_TASK_UNSCHEDULED"`)) {
		t.Fatalf("unscheduled status=%d body=%s", unscheduled.Code, unscheduled.Body.String())
	}
	var count int64
	if err := database.Model(&acceptanceSprint{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("rejected requests persisted partial Sprints: %d", count)
	}
}

func TestHTTPSuggestionAndCandidatePickerExposeCanonicalAllocationWithoutWriting_SPD17_AC22To44(t *testing.T) {
	database, handler := acceptanceServer(t)
	suggestion := performJSON(t, handler, http.MethodPost, "/api/sprints/suggestion", `{"startDate":"2026-08-04","endDate":"2026-08-04","memberIds":["member-1"]}`)
	if suggestion.Code != http.StatusOK ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"reason":"mandatory"`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"totalAllocationMinutes":120`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"parentName":"Alpha"`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"effortMinutes":450`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"executionStart":"2026-08-04"`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"executionEnd":"2026-08-04"`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"commitmentStart":"2026-08-04"`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"commitmentEnd":"2026-08-05"`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"wbsPath":"1"`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"dailyPlanOrderDate":"2026-08-04"`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"dailySummaries":[{"date":"2026-08-04","capacityMinutes":390,"selectedAllocationMinutes":120,"remainingMinutes":270,"overcapacityMinutes":0}]`)) ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"warnings":[]`)) {
		t.Fatalf("suggestion status=%d body=%s", suggestion.Code, suggestion.Body.String())
	}
	var count int64
	if err := database.Model(&acceptanceSprint{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("suggestion persisted %d Sprints", count)
	}
	draftCandidates := performJSON(t, handler, http.MethodPost, "/api/sprints/task-candidates?page=1&pageSize=20", `{"startDate":"2026-08-04","endDate":"2026-08-04","memberIds":["member-1"],"excludedTaskIds":[]}`)
	if draftCandidates.Code != http.StatusOK ||
		!bytes.Contains(draftCandidates.Body.Bytes(), []byte(`"id":"task-1"`)) ||
		!bytes.Contains(draftCandidates.Body.Bytes(), []byte(`"inSprintAllocationMinutes":120`)) ||
		!bytes.Contains(draftCandidates.Body.Bytes(), []byte(`"parentName":"Alpha"`)) ||
		!bytes.Contains(draftCandidates.Body.Bytes(), []byte(`"effortMinutes":450`)) ||
		!bytes.Contains(draftCandidates.Body.Bytes(), []byte(`"executionStart":"2026-08-04"`)) ||
		!bytes.Contains(draftCandidates.Body.Bytes(), []byte(`"executionEnd":"2026-08-04"`)) ||
		!bytes.Contains(draftCandidates.Body.Bytes(), []byte(`"commitmentStart":"2026-08-04"`)) ||
		!bytes.Contains(draftCandidates.Body.Bytes(), []byte(`"commitmentEnd":"2026-08-05"`)) ||
		!bytes.Contains(draftCandidates.Body.Bytes(), []byte(`"wbsPath":"1"`)) ||
		!bytes.Contains(draftCandidates.Body.Bytes(), []byte(`"dailyPlanOrderDate":"2026-08-04"`)) {
		t.Fatalf("draft candidates status=%d body=%s", draftCandidates.Code, draftCandidates.Body.String())
	}
	if err := database.Model(&acceptanceSprint{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("draft candidate lookup persisted %d Sprints", count)
	}

	created := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":"Picker","startDate":"2026-08-04","endDate":"2026-08-04","memberIds":["member-1"],"taskIds":[]}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	sprintID := decodeDataID(t, created)
	candidates := performJSON(t, handler, http.MethodGet, "/api/sprints/"+sprintID+"/task-candidates?page=1&pageSize=20", "")
	if candidates.Code != http.StatusOK ||
		!bytes.Contains(candidates.Body.Bytes(), []byte(`"id":"task-1"`)) ||
		!bytes.Contains(candidates.Body.Bytes(), []byte(`"inSprintAllocationMinutes":120`)) ||
		!bytes.Contains(candidates.Body.Bytes(), []byte(`"parentName":"Alpha"`)) ||
		!bytes.Contains(candidates.Body.Bytes(), []byte(`"effortMinutes":450`)) ||
		!bytes.Contains(candidates.Body.Bytes(), []byte(`"executionStart":"2026-08-04"`)) ||
		!bytes.Contains(candidates.Body.Bytes(), []byte(`"executionEnd":"2026-08-04"`)) ||
		!bytes.Contains(candidates.Body.Bytes(), []byte(`"commitmentStart":"2026-08-04"`)) ||
		!bytes.Contains(candidates.Body.Bytes(), []byte(`"commitmentEnd":"2026-08-05"`)) ||
		!bytes.Contains(candidates.Body.Bytes(), []byte(`"wbsPath":"1"`)) {
		t.Fatalf("candidates status=%d body=%s", candidates.Code, candidates.Body.String())
	}
	if err := database.Model(&acceptanceTask{}).Where("id = ?", "task-1").Updates(map[string]any{"execution_start": nil, "execution_end": nil}).Error; err != nil {
		t.Fatal(err)
	}
	filtered := performJSON(t, handler, http.MethodGet, "/api/sprints/"+sprintID+"/task-candidates?page=1&pageSize=20", "")
	if filtered.Code != http.StatusOK || bytes.Contains(filtered.Body.Bytes(), []byte(`"id":"task-1"`)) {
		t.Fatalf("unscheduled candidate remained visible: status=%d body=%s", filtered.Code, filtered.Body.String())
	}
}

func TestHTTPSuggestionReturnsStableUnavailableErrorForUnreadableAllocation_SPD07_AC77(t *testing.T) {
	database, handler := acceptanceServer(t)
	if err := database.Model(&acceptanceAllocation{}).
		Where("task_id = ? AND timeline = ?", "task-1", "execution").
		Update("allocated_minutes", "0").Error; err != nil {
		t.Fatal(err)
	}

	suggestion := performJSON(t, handler, http.MethodPost, "/api/sprints/suggestion", `{"startDate":"2026-08-04","endDate":"2026-08-04","memberIds":["member-1"]}`)
	if suggestion.Code != http.StatusServiceUnavailable ||
		!bytes.Contains(suggestion.Body.Bytes(), []byte(`"code":"SPRINT_SUGGESTION_UNAVAILABLE"`)) {
		t.Fatalf("suggestion status=%d body=%s", suggestion.Code, suggestion.Body.String())
	}
	var count int64
	if err := database.Model(&acceptanceSprint{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("failed suggestion persisted %d Sprint rows", count)
	}
}

func TestHTTPDetailUsesBoundedSetBasedQueriesAsTaskCountGrows_SPD07_AC75(t *testing.T) {
	database, handler := acceptanceServer(t)
	created := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":"Query shape","startDate":"2026-08-04","endDate":"2026-08-04","memberIds":["member-1"],"taskIds":["task-1"]}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	sprintID := decodeDataID(t, created)

	var queryCount atomic.Int64
	if err := database.Callback().Query().Before("gorm:query").Register("test:sprint-detail-query-count", func(*gorm.DB) {
		queryCount.Add(1)
	}); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	queryCount.Store(0)
	single := performJSON(t, handler, http.MethodGet, "/api/sprints/"+sprintID, "")
	if single.Code != http.StatusOK {
		t.Fatalf("single-Task detail status=%d body=%s", single.Code, single.Body.String())
	}
	singleTaskQueries := queryCount.Load()
	if singleTaskQueries == 0 {
		t.Fatal("query counter did not observe the Sprint detail read")
	}

	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	memberID := "member-1"
	for index := 2; index <= 12; index++ {
		taskID := "task-" + strconv.Itoa(index)
		if err := database.Create(&acceptanceTask{
			ID: taskID, ProjectID: "project-1", Name: "Task " + strconv.Itoa(index), Position: index,
			AssigneeID: &memberID, ExecutionStart: &day, ExecutionEnd: &day, UpdatedAt: day,
		}).Error; err != nil {
			t.Fatalf("create %s: %v", taskID, err)
		}
		if err := database.Create(&acceptanceAllocation{
			TaskID: taskID, AssigneeID: memberID, Timeline: "execution", AllocationDate: day, AllocatedMinutes: "60",
		}).Error; err != nil {
			t.Fatalf("create %s allocation: %v", taskID, err)
		}
		if err := database.Create(&acceptanceSprintTask{SprintID: sprintID, TaskID: taskID}).Error; err != nil {
			t.Fatalf("attach %s: %v", taskID, err)
		}
	}

	queryCount.Store(0)
	many := performJSON(t, handler, http.MethodGet, "/api/sprints/"+sprintID, "")
	if many.Code != http.StatusOK || !bytes.Contains(many.Body.Bytes(), []byte(`"id":"task-12"`)) {
		t.Fatalf("multi-Task detail status=%d body=%s", many.Code, many.Body.String())
	}
	manyTaskQueries := queryCount.Load()
	if manyTaskQueries != singleTaskQueries {
		t.Fatalf("Sprint detail query count grew with Task cardinality: one Task=%d queries, twelve Tasks=%d queries", singleTaskQueries, manyTaskQueries)
	}
}

func TestHTTPDetailPreservesDailyCrossMemberVarianceWithoutNetting_SPD03And04_AC19And79(t *testing.T) {
	database, handler := acceptanceServer(t)
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	if err := database.Model(&acceptanceMember{}).Where("id = ?", "member-1").Update("buffer_percentage", "0").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceMember{ID: "member-2", Name: "Sally", RoleID: "role-1", DailyCapacity: "8.0", BufferPercentage: "0", UpdatedAt: day}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceProject{ID: "project-2", Name: "Beta", Status: "open", Priority: 2, ScheduleVersion: 1}).Error; err != nil {
		t.Fatal(err)
	}
	memberID := "member-2"
	if err := database.Create(&acceptanceTask{ID: "task-2", ProjectID: "project-2", Name: "UI task", Position: 1, AssigneeID: &memberID, ExecutionStart: &day, ExecutionEnd: &day, UpdatedAt: day}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&acceptanceAllocation{}).Where("task_id = ?", "task-1").Update("allocated_minutes", "600").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceAllocation{TaskID: "task-2", AssigneeID: memberID, Timeline: "execution", AllocationDate: day, AllocatedMinutes: "360"}).Error; err != nil {
		t.Fatal(err)
	}

	created := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":"Variance","startDate":"2026-08-04","endDate":"2026-08-04","memberIds":["member-1","member-2"],"taskIds":["task-1","task-2"]}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	detail := performJSON(t, handler, http.MethodGet, "/api/sprints/"+decodeDataID(t, created), "")
	if detail.Code != http.StatusOK ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"capacityMinutes":960,"selectedMemberAllocationMinutes":960,"remainingMinutes":120,"overcapacityMinutes":120`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"dailySummaries":[{"date":"2026-08-04","capacityMinutes":960,"selectedAllocationMinutes":960,"remainingMinutes":120,"overcapacityMinutes":120}]`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"name":"Harry"`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"name":"Sally"`)) {
		t.Fatalf("daily variance was netted in HTTP detail: status=%d body=%s", detail.Code, detail.Body.String())
	}
}

func TestHTTPDetailPreservesCrossDateVarianceWithoutNetting_SPD03_AC19And79(t *testing.T) {
	database, handler := acceptanceServer(t)
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	if err := database.Model(&acceptanceMember{}).Where("id = ?", "member-1").Update("buffer_percentage", "0").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&acceptanceAllocation{}).Where("task_id = ?", "task-1").Update("allocated_minutes", "600").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceOverride{
		ID: "day-two-capacity", TeamMemberID: "member-1", StartDate: day.AddDate(0, 0, 1), EndDate: day.AddDate(0, 0, 1), Capacity: "2.0", UpdatedAt: day,
	}).Error; err != nil {
		t.Fatal(err)
	}

	created := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":"Cross date","startDate":"2026-08-04","endDate":"2026-08-05","memberIds":["member-1"],"taskIds":["task-1"]}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", created.Code, created.Body.String())
	}
	detail := performJSON(t, handler, http.MethodGet, "/api/sprints/"+decodeDataID(t, created), "")
	if detail.Code != http.StatusOK ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"capacityMinutes":600,"selectedMemberAllocationMinutes":600,"remainingMinutes":120,"overcapacityMinutes":120`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`{"date":"2026-08-04","capacityMinutes":480,"selectedAllocationMinutes":600,"remainingMinutes":0,"overcapacityMinutes":120}`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`{"date":"2026-08-05","capacityMinutes":120,"selectedAllocationMinutes":0,"remainingMinutes":120,"overcapacityMinutes":0}`)) {
		t.Fatalf("cross-Date variance was netted: status=%d body=%s", detail.Code, detail.Body.String())
	}
}

func TestHTTPAllowsZeroCapacityAllocationAndReportsFullOvercapacity_SPD03_AC19And20(t *testing.T) {
	database, handler := acceptanceServer(t)
	weekend := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	if err := database.Model(&acceptanceTask{}).Where("id = ?", "task-1").Updates(map[string]any{
		"execution_start": weekend,
		"execution_end":   weekend,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Where("task_id = ?", "task-1").Delete(&acceptanceAllocation{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceAllocation{
		TaskID: "task-1", AssigneeID: "member-1", Timeline: "execution", AllocationDate: weekend, AllocatedMinutes: "120",
	}).Error; err != nil {
		t.Fatal(err)
	}

	created := performJSON(t, handler, http.MethodPost, "/api/sprints", `{"name":"Weekend","startDate":"2026-08-08","endDate":"2026-08-08","memberIds":["member-1"],"taskIds":["task-1"]}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("zero-capacity Save was blocked: status=%d body=%s", created.Code, created.Body.String())
	}
	detail := performJSON(t, handler, http.MethodGet, "/api/sprints/"+decodeDataID(t, created), "")
	if detail.Code != http.StatusOK ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"capacityMinutes":0,"selectedMemberAllocationMinutes":120,"remainingMinutes":0,"overcapacityMinutes":120`)) ||
		!bytes.Contains(detail.Body.Bytes(), []byte(`"dailySummaries":[{"date":"2026-08-08","capacityMinutes":0,"selectedAllocationMinutes":120,"remainingMinutes":0,"overcapacityMinutes":120}]`)) {
		t.Fatalf("zero-capacity allocation was not fully overcapacity: status=%d body=%s", detail.Code, detail.Body.String())
	}
}
