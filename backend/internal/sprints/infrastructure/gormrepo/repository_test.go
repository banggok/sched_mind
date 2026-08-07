package gormrepo

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	"github.com/banggok/sched_mind/backend/internal/sprints/application"
	"github.com/banggok/sched_mind/backend/internal/sprints/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type projectTestModel struct {
	ID              string `gorm:"primaryKey"`
	Name            string
	Status          string
	Priority        int
	ScheduleVersion int64
}

func (projectTestModel) TableName() string { return "projects" }

type taskTestModel struct {
	ID              string `gorm:"primaryKey"`
	ProjectID       string
	ParentID        *string
	AssigneeID      *string
	EffortMinutes   *int
	ExecutionStart  *time.Time
	ExecutionEnd    *time.Time
	CommitmentStart *time.Time
	CommitmentEnd   *time.Time
	ActualStart     *time.Time
	ActualEnd       *time.Time
	ParentKey       string
	Position        int
	Name            string
	UpdatedAt       time.Time
}

func (taskTestModel) TableName() string { return "wbs_nodes" }

type roleTestModel struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

func (roleTestModel) TableName() string { return "roles" }

type overrideTestModel struct {
	ID, TeamMemberID, Capacity    string
	StartDate, EndDate, UpdatedAt time.Time
	DeletedAt                     *time.Time
}

func (overrideTestModel) TableName() string { return "capacity_overrides" }

type holidayTestModel struct {
	PublicHolidayID string
	Date            time.Time
}

func (holidayTestModel) TableName() string { return "public_holiday_dates" }

type allocationTestModel struct {
	TaskID, AssigneeID, Timeline string    `gorm:"primaryKey"`
	AllocationDate               time.Time `gorm:"primaryKey"`
	AllocatedMinutes             string
}

func (allocationTestModel) TableName() string { return "task_schedule_allocations" }

func sprintTestRepository(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_foreign_keys=1"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := database.AutoMigrate(&roleTestModel{}, &memberModel{}, &projectTestModel{}, &taskTestModel{}, &overrideTestModel{}, &holidayTestModel{}, &allocationTestModel{}, &sprintModel{}, &sprintMemberModel{}, &sprintTaskModel{}); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return New(database), database
}

func seedEligibleTask(t *testing.T, database *gorm.DB, memberID, taskID string) {
	t.Helper()
	now := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	if err := database.FirstOrCreate(&roleTestModel{ID: "role", Name: "Engineer"}, "id = ?", "role").Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	if err := database.FirstOrCreate(&memberModel{ID: memberID, Name: memberID, RoleID: "role", DailyCapacity: "8.0", BufferPercentage: "20.0", UpdatedAt: now}, "id = ?", memberID).Error; err != nil {
		t.Fatalf("create member: %v", err)
	}
	projectID := "project-" + taskID
	if err := database.Create(&projectTestModel{ID: projectID, Name: projectID, Status: "open", Priority: 1, ScheduleVersion: 1}).Error; err != nil {
		t.Fatalf("create project: %v", err)
	}
	effortMinutes := 480
	if err := database.Create(&taskTestModel{ID: taskID, ProjectID: projectID, AssigneeID: &memberID, EffortMinutes: &effortMinutes, ExecutionStart: &now, ExecutionEnd: &now, CommitmentStart: &now, CommitmentEnd: &now, Name: taskID, Position: 1, UpdatedAt: now}).Error; err != nil {
		t.Fatalf("create task: %v", err)
	}
}

func newTestSprint(t *testing.T, id, memberID, taskID string, start, end time.Time) domain.Sprint {
	t.Helper()
	sprint, err := domain.NewSprint(id, id, start, end, []string{memberID}, []string{taskID}, start)
	if err != nil {
		t.Fatalf("create sprint aggregate: %v", err)
	}
	return *sprint
}

func TestCreatePersistsSprintAndRelationsAtomically_AC45And73(t *testing.T) {
	repository, database := sprintTestRepository(t)
	seedEligibleTask(t, database, "member-1", "task-1")
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	sprint := newTestSprint(t, "sprint-1", "member-1", "task-1", day, day.AddDate(0, 0, 7))

	if err := repository.Create(context.Background(), sprint); err != nil {
		t.Fatalf("create sprint: %v", err)
	}
	stored, err := repository.Find(context.Background(), sprint.ID)
	if err != nil {
		t.Fatalf("read sprint: %v", err)
	}
	if stored.ID != sprint.ID || len(stored.MemberIDs) != 1 || stored.MemberIDs[0] != "member-1" || len(stored.TaskIDs) != 1 || stored.TaskIDs[0] != "task-1" {
		t.Fatalf("unexpected persisted aggregate: %#v", stored)
	}
}

func TestCreateRollsBackWhenTaskIsMissing_AC12And45(t *testing.T) {
	repository, database := sprintTestRepository(t)
	if err := database.Create(&memberModel{ID: "member-1"}).Error; err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	sprint := newTestSprint(t, "sprint-1", "member-1", "missing-task", day, day)

	err := repository.Create(context.Background(), sprint)
	if !errors.Is(err, application.ErrTaskNotFound) {
		t.Fatalf("got %v, want task not found", err)
	}
	var sprintCount, memberCount int64
	if err := database.Model(&sprintModel{}).Count(&sprintCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&sprintMemberModel{}).Count(&memberCount).Error; err != nil {
		t.Fatal(err)
	}
	if sprintCount != 0 || memberCount != 0 {
		t.Fatalf("partial state persisted: sprints=%d members=%d", sprintCount, memberCount)
	}
}

func TestCreateRejectsInclusiveSharedMemberOverlapAndAllowsDisjointMember_AC59To64(t *testing.T) {
	repository, database := sprintTestRepository(t)
	seedEligibleTask(t, database, "member-1", "task-1")
	seedEligibleTask(t, database, "member-2", "task-2")
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	first := newTestSprint(t, "sprint-1", "member-1", "task-1", day, day.AddDate(0, 0, 7))
	if err := repository.Create(context.Background(), first); err != nil {
		t.Fatal(err)
	}

	boundary := newTestSprint(t, "sprint-2", "member-1", "task-1", day.AddDate(0, 0, 7), day.AddDate(0, 0, 10))
	err := repository.Create(context.Background(), boundary)
	if !errors.Is(err, application.ErrMemberOverlap) {
		t.Fatalf("got %v, want member overlap", err)
	}
	var overlap *application.MemberOverlapError
	if !errors.As(err, &overlap) || overlap.SprintID != first.ID || overlap.SprintName != first.Name || !overlap.StartDate.Equal(first.StartDate) || !overlap.EndDate.Equal(first.EndDate) || len(overlap.Members) != 1 || overlap.Members[0].ID != "member-1" || overlap.Members[0].Name != "member-1" {
		t.Fatalf("overlap detail=%#v", overlap)
	}
	disjointMember := newTestSprint(t, "sprint-3", "member-2", "task-2", day, day.AddDate(0, 0, 7))
	if err := repository.Create(context.Background(), disjointMember); err != nil {
		t.Fatalf("same range for disjoint member: %v", err)
	}
}

func TestConcurrentOverlappingCreatesHaveExactlyOneWinner_AC64(t *testing.T) {
	repository, database := sprintTestRepository(t)
	seedEligibleTask(t, database, "member-1", "task-1")
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	first := newTestSprint(t, "sprint-concurrent-1", "member-1", "task-1", day, day.AddDate(0, 0, 7))
	second := newTestSprint(t, "sprint-concurrent-2", "member-1", "task-1", day, day.AddDate(0, 0, 7))
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for _, sprint := range []domain.Sprint{first, second} {
		go func(candidate domain.Sprint) {
			ready.Done()
			<-start
			results <- repository.Create(context.Background(), candidate)
		}(sprint)
	}
	ready.Wait()
	close(start)
	firstResult, secondResult := <-results, <-results
	successes, overlaps := 0, 0
	for _, err := range []error{firstResult, secondResult} {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, application.ErrMemberOverlap):
			overlaps++
		default:
			t.Fatalf("unexpected concurrent result: %v", err)
		}
	}
	if successes != 1 || overlaps != 1 {
		t.Fatalf("successes=%d overlaps=%d", successes, overlaps)
	}
	var sprintCount, memberRelationCount int64
	if err := database.Model(&sprintModel{}).Count(&sprintCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&sprintMemberModel{}).Where("member_id = ?", "member-1").Count(&memberRelationCount).Error; err != nil {
		t.Fatal(err)
	}
	if sprintCount != 1 || memberRelationCount != 1 {
		t.Fatalf("persisted Sprints=%d member relations=%d", sprintCount, memberRelationCount)
	}
}

func TestUpdateRetainsDriftedExistingTaskButRejectsDriftedNewTask_AC46To58(t *testing.T) {
	repository, database := sprintTestRepository(t)
	seedEligibleTask(t, database, "member-1", "retained-task")
	seedEligibleTask(t, database, "member-2", "new-task")
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	sprint := newTestSprint(t, "sprint-1", "member-1", "retained-task", day, day.AddDate(0, 0, 7))
	if err := repository.Create(context.Background(), sprint); err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&taskTestModel{}).Where("id IN ?", []string{"retained-task", "new-task"}).Updates(map[string]any{"execution_start": nil, "execution_end": nil}).Error; err != nil {
		t.Fatal(err)
	}

	sprint.Version = 2
	sprint.TaskIDs = []string{"retained-task"}
	if err := repository.Update(context.Background(), sprint, 1, map[string]struct{}{"retained-task": {}}); err != nil {
		t.Fatalf("retain drifted existing task: %v", err)
	}
	sprint.Version = 3
	sprint.TaskIDs = []string{"retained-task", "new-task"}
	err := repository.Update(context.Background(), sprint, 2, map[string]struct{}{"retained-task": {}})
	if !errors.Is(err, application.ErrTaskUnscheduled) {
		t.Fatalf("got %v, want new task unscheduled", err)
	}
	stored, findErr := repository.Find(context.Background(), sprint.ID)
	if findErr != nil {
		t.Fatal(findErr)
	}
	if stored.Version != 2 || len(stored.TaskIDs) != 1 || stored.TaskIDs[0] != "retained-task" {
		t.Fatalf("failed update changed persisted state: %#v", stored)
	}
}

func TestStartAndDeleteUseVersionAndRemoveOnlySprintRelations_AC67To72(t *testing.T) {
	repository, database := sprintTestRepository(t)
	seedEligibleTask(t, database, "member-1", "task-1")
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	sprint := newTestSprint(t, "sprint-1", "member-1", "task-1", day, day)
	if err := repository.Create(context.Background(), sprint); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Start(context.Background(), sprint.ID, 99, day); !errors.Is(err, application.ErrVersionConflict) {
		t.Fatalf("got %v, want version conflict", err)
	}
	started, err := repository.Start(context.Background(), sprint.ID, 1, day.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if started.Status != domain.StatusStarted || started.Version != 2 || started.StartedAt == nil {
		t.Fatalf("unexpected started sprint: %#v", started)
	}
	if err := repository.Delete(context.Background(), sprint.ID, 1); !errors.Is(err, application.ErrVersionConflict) {
		t.Fatalf("got %v, want stale delete conflict", err)
	}
	if err := repository.Delete(context.Background(), sprint.ID, 2); err != nil {
		t.Fatal(err)
	}
	var taskCount, memberCount, sprintMemberCount, sprintTaskCount int64
	if err := database.Model(&taskTestModel{}).Where("id = ?", "task-1").Count(&taskCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&memberModel{}).Where("id = ?", "member-1").Count(&memberCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&sprintMemberModel{}).Where("sprint_id = ?", sprint.ID).Count(&sprintMemberCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&sprintTaskModel{}).Where("sprint_id = ?", sprint.ID).Count(&sprintTaskCount).Error; err != nil {
		t.Fatal(err)
	}
	if taskCount != 1 || memberCount != 1 {
		t.Fatalf("delete mutated owning entities: tasks=%d members=%d", taskCount, memberCount)
	}
	if sprintMemberCount != 0 || sprintTaskCount != 0 {
		t.Fatalf("delete retained relations: members=%d tasks=%d", sprintMemberCount, sprintTaskCount)
	}
}

func TestDetailComposesLiveCapacityOutsideAllocationAndNeedsReview_SPD17_AC17To21And34To55(t *testing.T) {
	repository, database := sprintTestRepository(t)
	seedEligibleTask(t, database, "member-1", "task-1")
	day := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	if err := database.Model(&memberModel{}).Where("id = ?", "member-1").Updates(map[string]any{"daily_capacity": "8.0", "buffer_percentage": "25.0"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&taskTestModel{}).Where("id = ?", "task-1").Updates(map[string]any{
		"effort_minutes":   450,
		"execution_start":  day.AddDate(0, 0, -1),
		"execution_end":    day.AddDate(0, 0, 4),
		"commitment_start": day,
		"commitment_end":   day.AddDate(0, 0, 3),
	}).Error; err != nil {
		t.Fatal(err)
	}
	sprint := newTestSprint(t, "sprint-1", "member-1", "task-1", day, day.AddDate(0, 0, 4))
	if err := repository.Create(context.Background(), sprint); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&overrideTestModel{ID: "override", TeamMemberID: "member-1", Capacity: "4.0", StartDate: day.AddDate(0, 0, 1), EndDate: day.AddDate(0, 0, 1), UpdatedAt: day}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&holidayTestModel{PublicHolidayID: "holiday", Date: day.AddDate(0, 0, 2)}).Error; err != nil {
		t.Fatal(err)
	}
	allocations := []allocationTestModel{
		{TaskID: "task-1", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day.AddDate(0, 0, -1), AllocatedMinutes: "60"},
		{TaskID: "task-1", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "420"},
	}
	if err := database.Create(&allocations).Error; err != nil {
		t.Fatal(err)
	}

	detail, err := repository.Detail(context.Background(), sprint.ID)
	if err != nil {
		t.Fatalf("detail: %v", err)
	}
	if len(detail.Members) != 1 || detail.Members[0].CapacityMinutes != 1260 {
		t.Fatalf("unexpected capacity projection: %#v", detail.Members)
	}
	if len(detail.Tasks) != 1 || detail.Tasks[0].InSprintAllocationMinutes != 420 || detail.Tasks[0].OutsideAllocationMinutes != 60 || detail.Tasks[0].TotalAllocationMinutes != 480 {
		t.Fatalf("unexpected task allocation: %#v", detail.Tasks)
	}
	task := detail.Tasks[0]
	if task.EffortMinutes == nil || *task.EffortMinutes != 450 ||
		task.ExecutionStart == nil || !task.ExecutionStart.Equal(day.AddDate(0, 0, -1)) ||
		task.ExecutionEnd == nil || !task.ExecutionEnd.Equal(day.AddDate(0, 0, 4)) ||
		task.CommitmentStart == nil || !task.CommitmentStart.Equal(day) ||
		task.CommitmentEnd == nil || !task.CommitmentEnd.Equal(day.AddDate(0, 0, 3)) ||
		task.ParentName != "project-task-1" {
		t.Fatalf("live Task parent and planning metadata were not projected: %#v", task)
	}
	if detail.Members[0].InSprintAllocationMinutes != 420 || detail.Members[0].RemainingMinutes != 900 || detail.Members[0].OvercapacityMinutes != 60 || detail.Totals.SelectedMemberAllocationMinutes != 420 || detail.Totals.RemainingMinutes != 900 || detail.Totals.OvercapacityMinutes != 60 || detail.ProjectionToken == "" {
		t.Fatalf("unexpected selected utilization: %#v", detail)
	}
	if len(detail.Members[0].DailySummaries) != 5 || detail.Members[0].DailySummaries[0].OvercapacityMinutes != 60 {
		t.Fatalf("daily variance must preserve overloaded and unused Dates: %#v", detail.Members[0].DailySummaries)
	}
	metadataToken := detail.ProjectionToken
	if err := database.Model(&taskTestModel{}).Where("id = ?", "task-1").Updates(map[string]any{
		"effort_minutes":   510,
		"commitment_start": day.AddDate(0, 0, 1),
		"commitment_end":   day.AddDate(0, 0, 4),
	}).Error; err != nil {
		t.Fatal(err)
	}
	metadataDrift, err := repository.Detail(context.Background(), sprint.ID)
	if err != nil {
		t.Fatalf("detail after planning metadata drift: %v", err)
	}
	if metadataDrift.ProjectionToken == metadataToken ||
		metadataDrift.Tasks[0].EffortMinutes == nil || *metadataDrift.Tasks[0].EffortMinutes != 510 ||
		metadataDrift.Tasks[0].CommitmentStart == nil || !metadataDrift.Tasks[0].CommitmentStart.Equal(day.AddDate(0, 0, 1)) ||
		metadataDrift.Tasks[0].CommitmentEnd == nil || !metadataDrift.Tasks[0].CommitmentEnd.Equal(day.AddDate(0, 0, 4)) ||
		metadataDrift.Members[0].InSprintAllocationMinutes != 420 ||
		metadataDrift.Totals.SelectedMemberAllocationMinutes != 420 {
		t.Fatalf("planning metadata drift must refresh the token without changing allocation usage: before=%s after=%s detail=%#v", metadataToken, metadataDrift.ProjectionToken, metadataDrift)
	}

	seedEligibleTask(t, database, "member-2", "unused-task")
	if err := database.Model(&taskTestModel{}).Where("id = ?", "task-1").Update("assignee_id", "member-2").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&allocationTestModel{}).Where("task_id = ?", "task-1").Update("assignee_id", "member-2").Error; err != nil {
		t.Fatal(err)
	}
	drifted, err := repository.Detail(context.Background(), sprint.ID)
	if err != nil {
		t.Fatalf("drifted detail: %v", err)
	}
	if len(drifted.Tasks[0].Warnings) == 0 ||
		drifted.Members[0].InSprintAllocationMinutes != 0 ||
		drifted.Totals.NeedsReviewAllocationMinutes != 420 ||
		len(drifted.Totals.NeedsReviewDailyAllocation) != 1 ||
		drifted.Totals.AllTaskInSprintMinutes != 420 ||
		drifted.Totals.AllTaskTotalMinutes != 480 {
		t.Fatalf("drift was not separated from selected-member utilization: %#v", drifted)
	}
	driftedToken := drifted.ProjectionToken
	if err := database.Model(&memberModel{}).Where("id = ?", "member-2").Update("name", "Renamed Assignee").Error; err != nil {
		t.Fatal(err)
	}
	renamed, err := repository.Detail(context.Background(), sprint.ID)
	if err != nil {
		t.Fatalf("detail after non-Sprint Assignee rename: %v", err)
	}
	if renamed.Tasks[0].AssigneeName == nil || *renamed.Tasks[0].AssigneeName != "Renamed Assignee" || renamed.ProjectionToken == driftedToken {
		t.Fatalf("live Assignee display drift must update projection token: before=%s after=%s task=%#v", driftedToken, renamed.ProjectionToken, renamed.Tasks[0])
	}
}

func TestSuggestionSelectsMandatoryZeroAllocationThenFillsWholeTasks_AC22To33(t *testing.T) {
	repository, database := sprintTestRepository(t)
	day := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	seedEligibleTask(t, database, "member-1", "mandatory")
	seedEligibleTask(t, database, "member-1", "fill-high")
	seedEligibleTask(t, database, "member-1", "fill-low")
	seedEligibleTask(t, database, "member-1", "later-zero")
	updates := map[string]time.Time{
		"mandatory": day.AddDate(0, 0, -1), "fill-high": day.AddDate(0, 0, 1),
		"fill-low": day.AddDate(0, 0, 1), "later-zero": day.AddDate(0, 0, 1),
	}
	for taskID, end := range updates {
		start := day
		if taskID == "mandatory" {
			start = day.AddDate(0, 0, -2)
		}
		if err := database.Model(&taskTestModel{}).Where("id = ?", taskID).Updates(map[string]any{"execution_start": start, "execution_end": end}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Model(&projectTestModel{}).Where("id = ?", "project-fill-high").Update("priority", 1).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&projectTestModel{}).Where("id = ?", "project-fill-low").Update("priority", 9).Error; err != nil {
		t.Fatal(err)
	}
	allocations := []allocationTestModel{
		{TaskID: "mandatory", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day.AddDate(0, 0, -1), AllocatedMinutes: "60"},
		{TaskID: "fill-high", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "240"},
		{TaskID: "fill-low", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "240"},
		{TaskID: "later-zero", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day.AddDate(0, 0, 2), AllocatedMinutes: "60"},
	}
	if err := database.Create(&allocations).Error; err != nil {
		t.Fatal(err)
	}

	suggestion, err := repository.Suggest(context.Background(), application.SuggestionInput{StartDate: day, EndDate: day, MemberIDs: []string{"member-1"}})
	if err != nil {
		t.Fatalf("suggest: %v", err)
	}
	if len(suggestion.Tasks) != 3 {
		t.Fatalf("got tasks %#v, want mandatory and two complete fill tasks", suggestion.Tasks)
	}
	if suggestion.Tasks[0].Task.ID != "mandatory" || suggestion.Tasks[0].Reason != "mandatory" || suggestion.Tasks[1].Task.ID != "fill-high" || suggestion.Tasks[2].Task.ID != "fill-low" {
		t.Fatalf("unexpected deterministic suggestion: %#v", suggestion.Tasks)
	}
	if suggestion.Members[0].OvercapacityMinutes != 90 || suggestion.ProjectionToken == "" {
		t.Fatalf("whole-task fill must expose overcapacity and token: %#v", suggestion)
	}
	for _, suggested := range suggestion.Tasks {
		if suggested.Task.EffortMinutes == nil || *suggested.Task.EffortMinutes != 480 ||
			suggested.Task.CommitmentStart == nil || suggested.Task.CommitmentEnd == nil ||
			suggested.Task.ParentName != suggested.Task.ProjectName {
			t.Fatalf("suggestion must include live Parent, Effort, and Commitment metadata: %#v", suggested.Task)
		}
	}
}

func TestSuggestionRejectsUnreadableCanonicalAllocationWithoutPersisting_SPD07_AC77(t *testing.T) {
	repository, database := sprintTestRepository(t)
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	seedEligibleTask(t, database, "member-1", "unreadable-suggestion")
	if err := database.Create(&allocationTestModel{
		TaskID: "unreadable-suggestion", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "0",
	}).Error; err != nil {
		t.Fatal(err)
	}

	_, err := repository.Suggest(context.Background(), application.SuggestionInput{
		StartDate: day, EndDate: day, MemberIDs: []string{"member-1"},
	})
	if !errors.Is(err, schedulingdomain.ErrDataIntegrity) {
		t.Fatalf("unreadable canonical allocation must return a recoverable suggestion error, got %v", err)
	}
	var count int64
	if err := database.Model(&sprintModel{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("read-only suggestion persisted %d Sprint rows", count)
	}
}

func TestCandidatesReturnOnlyUnselectedEligibleTasksIncludingZeroInSprintAllocation_AC40And41(t *testing.T) {
	repository, database := sprintTestRepository(t)
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	seedEligibleTask(t, database, "member-1", "selected")
	seedEligibleTask(t, database, "member-1", "eligible-zero")
	seedEligibleTask(t, database, "member-1", "unscheduled")
	seedEligibleTask(t, database, "member-1", "completed")
	seedEligibleTask(t, database, "member-2", "other-member")
	for _, allocation := range []allocationTestModel{
		{TaskID: "selected", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "60"},
		{TaskID: "eligible-zero", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day.AddDate(0, 0, 2), AllocatedMinutes: "60"},
		{TaskID: "unscheduled", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "60"},
		{TaskID: "completed", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "60"},
		{TaskID: "other-member", AssigneeID: "member-2", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "60"},
	} {
		if err := database.Create(&allocation).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := database.Model(&taskTestModel{}).Where("id = ?", "unscheduled").Updates(map[string]any{"execution_start": nil, "execution_end": nil}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&taskTestModel{}).Where("id = ?", "completed").Updates(map[string]any{"actual_start": day, "actual_end": day}).Error; err != nil {
		t.Fatal(err)
	}
	sprint := newTestSprint(t, "sprint-1", "member-1", "selected", day, day.AddDate(0, 0, 1))
	draftPage, err := repository.DraftCandidates(context.Background(), application.CandidateInput{
		StartDate: day, EndDate: day.AddDate(0, 0, 1), MemberIDs: []string{"member-1"}, ExcludedTaskIDs: []string{"selected"},
	}, listing.Query{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("draft candidates: %v", err)
	}
	if draftPage.Total != 1 || len(draftPage.Items) != 1 || draftPage.Items[0].ID != "eligible-zero" {
		t.Fatalf("unexpected draft candidate page: %#v", draftPage)
	}
	if draftPage.Items[0].EffortMinutes == nil || *draftPage.Items[0].EffortMinutes != 480 ||
		draftPage.Items[0].CommitmentStart == nil || draftPage.Items[0].CommitmentEnd == nil ||
		draftPage.Items[0].ParentName != draftPage.Items[0].ProjectName {
		t.Fatalf("draft candidate omitted live Parent, Effort, or Commitment metadata: %#v", draftPage.Items[0])
	}
	if err := repository.Create(context.Background(), sprint); err != nil {
		t.Fatal(err)
	}

	page, err := repository.Candidates(context.Background(), sprint.ID, listing.Query{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].ID != "eligible-zero" || page.Items[0].InSprintAllocationMinutes != 0 {
		t.Fatalf("unexpected candidate page: %#v", page)
	}
	if page.Items[0].EffortMinutes == nil || *page.Items[0].EffortMinutes != 480 ||
		page.Items[0].CommitmentStart == nil || page.Items[0].CommitmentEnd == nil ||
		page.Items[0].ParentName != page.Items[0].ProjectName {
		t.Fatalf("saved Sprint candidate omitted live Parent, Effort, or Commitment metadata: %#v", page.Items[0])
	}
}

func TestDetailOrdersFlatDailyPlanAcrossProjectsAndWBS_SPD01And02And17_AC34And36And47And74And78(t *testing.T) {
	repository, database := sprintTestRepository(t)
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	if err := database.Create(&roleTestModel{ID: "role", Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&memberModel{ID: "member-1", Name: "Harry", RoleID: "role", DailyCapacity: "8.0", BufferPercentage: "0", UpdatedAt: day}).Error; err != nil {
		t.Fatal(err)
	}
	for _, project := range []projectTestModel{
		{ID: "project-high", Name: "High", Status: "open", Priority: 1, ScheduleVersion: 1},
		{ID: "project-low", Name: "Low", Status: "open", Priority: 2, ScheduleVersion: 1},
	} {
		if err := database.Create(&project).Error; err != nil {
			t.Fatal(err)
		}
	}
	memberID := "member-1"
	groupID := "group-high"
	actual := day
	largeEffort := 960
	smallEffort := 60
	earlyCommitment := day
	lateCommitment := day.AddDate(0, 0, 10)
	nodes := []taskTestModel{
		{ID: groupID, ProjectID: "project-high", Name: "Group", Position: 1, UpdatedAt: day},
		{ID: "high-first", ProjectID: "project-high", ParentID: &groupID, ParentKey: groupID, Name: "High first", Position: 1, AssigneeID: &memberID, EffortMinutes: &largeEffort, ExecutionStart: &day, ExecutionEnd: &day, CommitmentStart: &lateCommitment, CommitmentEnd: &lateCommitment, UpdatedAt: day},
		{ID: "high-second", ProjectID: "project-high", ParentID: &groupID, ParentKey: groupID, Name: "High second", Position: 2, AssigneeID: &memberID, EffortMinutes: &smallEffort, ExecutionStart: &day, ExecutionEnd: &day, CommitmentStart: &earlyCommitment, CommitmentEnd: &earlyCommitment, UpdatedAt: day},
		{ID: "completed", ProjectID: "project-high", Name: "Completed", Position: 2, AssigneeID: &memberID, ExecutionStart: &day, ExecutionEnd: &day, ActualStart: &actual, ActualEnd: &actual, UpdatedAt: day},
		{ID: "unreadable", ProjectID: "project-high", Name: "Unreadable", Position: 3, AssigneeID: &memberID, ExecutionStart: &day, ExecutionEnd: &day, UpdatedAt: day},
		{ID: "low", ProjectID: "project-low", Name: "Low", Position: 1, AssigneeID: &memberID, ExecutionStart: &day, ExecutionEnd: &day, UpdatedAt: day},
	}
	for _, node := range nodes {
		if err := database.Create(&node).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, allocation := range []allocationTestModel{
		{TaskID: "high-first", AssigneeID: memberID, Timeline: "execution", AllocationDate: day, AllocatedMinutes: "60"},
		{TaskID: "high-second", AssigneeID: memberID, Timeline: "execution", AllocationDate: day, AllocatedMinutes: "60"},
		{TaskID: "low", AssigneeID: memberID, Timeline: "execution", AllocationDate: day, AllocatedMinutes: "60"},
		{TaskID: "completed", AssigneeID: memberID, Timeline: "execution", AllocationDate: day.AddDate(0, 0, -1), AllocatedMinutes: "60"},
		{TaskID: "unreadable", AssigneeID: memberID, Timeline: "execution", AllocationDate: day, AllocatedMinutes: "0"},
	} {
		if err := database.Create(&allocation).Error; err != nil {
			t.Fatal(err)
		}
	}
	sprint, err := domain.NewSprint("sprint-daily-plan", "Daily plan", day, day.AddDate(0, 0, 1), []string{memberID}, nil, day)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), *sprint); err != nil {
		t.Fatal(err)
	}
	for _, taskID := range []string{"low", "completed", "high-second", "unreadable", "high-first"} {
		if err := database.Create(&sprintTaskModel{SprintID: sprint.ID, TaskID: taskID}).Error; err != nil {
			t.Fatal(err)
		}
	}

	detail, err := repository.Detail(context.Background(), sprint.ID)
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(detail.Tasks))
	for _, task := range detail.Tasks {
		got = append(got, task.ID)
	}
	want := []string{"high-first", "high-second", "low", "completed", "unreadable"}
	if len(got) != len(want) {
		t.Fatalf("unexpected task count/order: got %v want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("unexpected flat daily-plan order: got %v want %v", got, want)
		}
	}
	if detail.Tasks[0].WBSPath != "1.1" || detail.Tasks[1].WBSPath != "1.2" || detail.Tasks[2].WBSPath != "1" {
		t.Fatalf("unexpected user-facing WBS paths: %#v", detail.Tasks)
	}
	if detail.Tasks[0].ParentName != "Group" || detail.Tasks[1].ParentName != "Group" || detail.Tasks[2].ParentName != "Low" {
		t.Fatalf("Task rows must project the immediate Parent Name and use Project Name only for root WBS nodes: %#v", detail.Tasks[:3])
	}
	if detail.Tasks[0].EffortMinutes == nil || *detail.Tasks[0].EffortMinutes != largeEffort ||
		detail.Tasks[1].EffortMinutes == nil || *detail.Tasks[1].EffortMinutes != smallEffort ||
		detail.Tasks[0].CommitmentEnd == nil || !detail.Tasks[0].CommitmentEnd.Equal(lateCommitment) ||
		detail.Tasks[1].CommitmentEnd == nil || !detail.Tasks[1].CommitmentEnd.Equal(earlyCommitment) {
		t.Fatalf("live metadata must be projected without overriding canonical daily-plan order: %#v", detail.Tasks[:2])
	}
	if detail.Tasks[4].DailyPlanOrderDate != nil ||
		len(detail.Tasks[4].Allocations) != 0 ||
		len(detail.Tasks[4].Warnings) != 1 ||
		detail.Tasks[4].Warnings[0] != "Execution allocation could not be resolved." {
		t.Fatalf("unreadable Task must use deterministic Needs Review fallback without failing the detail read: %#v", detail.Tasks[4])
	}
	initialToken := detail.ProjectionToken
	if err := database.Model(&taskTestModel{}).Where("id = ?", groupID).Update("name", "Renamed Group").Error; err != nil {
		t.Fatal(err)
	}
	renamedParent, err := repository.Detail(context.Background(), sprint.ID)
	if err != nil {
		t.Fatal(err)
	}
	if renamedParent.ProjectionToken == initialToken ||
		renamedParent.Tasks[0].ID != "high-first" || renamedParent.Tasks[1].ID != "high-second" ||
		renamedParent.Tasks[0].ParentName != "Renamed Group" || renamedParent.Tasks[1].ParentName != "Renamed Group" ||
		renamedParent.Members[0].InSprintAllocationMinutes != detail.Members[0].InSprintAllocationMinutes ||
		renamedParent.Totals.SelectedMemberAllocationMinutes != detail.Totals.SelectedMemberAllocationMinutes {
		t.Fatalf("parent rename must refresh live context and token without changing order or usage: before=%s after=%s tasks=%#v", initialToken, renamedParent.ProjectionToken, renamedParent.Tasks)
	}
	renamedToken := renamedParent.ProjectionToken
	if err := database.Model(&taskTestModel{}).Where("id = ?", groupID).Update("position", 4).Error; err != nil {
		t.Fatal(err)
	}
	reordered, err := repository.Detail(context.Background(), sprint.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reordered.ProjectionToken == renamedToken || reordered.Tasks[0].WBSPath != "3.1" || reordered.Tasks[0].ParentName != "Renamed Group" {
		t.Fatalf("ancestor WBS drift must update path and projection token while preserving direct Parent Name: before=%s after=%s tasks=%#v", renamedToken, reordered.ProjectionToken, reordered.Tasks)
	}
}

func TestDetailPreservesCrossMemberRemainingAndOvercapacity_SPD03And04_AC19And79(t *testing.T) {
	repository, database := sprintTestRepository(t)
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	seedEligibleTask(t, database, "member-a", "task-a")
	seedEligibleTask(t, database, "member-b", "task-b")
	if err := database.Model(&memberModel{}).Where("id IN ?", []string{"member-a", "member-b"}).Updates(map[string]any{"daily_capacity": "8.0", "buffer_percentage": "0"}).Error; err != nil {
		t.Fatal(err)
	}
	for _, allocation := range []allocationTestModel{
		{TaskID: "task-a", AssigneeID: "member-a", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "600"},
		{TaskID: "task-b", AssigneeID: "member-b", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "360"},
	} {
		if err := database.Create(&allocation).Error; err != nil {
			t.Fatal(err)
		}
	}
	sprint, err := domain.NewSprint("sprint-variance", "Variance", day, day, []string{"member-a", "member-b"}, []string{"task-a", "task-b"}, day)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), *sprint); err != nil {
		t.Fatal(err)
	}

	detail, err := repository.Detail(context.Background(), sprint.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Totals.CapacityMinutes != 960 || detail.Totals.SelectedMemberAllocationMinutes != 960 || detail.Totals.RemainingMinutes != 120 || detail.Totals.OvercapacityMinutes != 120 {
		t.Fatalf("Sprint totals netted Member variance: %#v", detail.Totals)
	}
	if len(detail.Totals.DailySummaries) != 1 || detail.Totals.DailySummaries[0].RemainingMinutes != 120 || detail.Totals.DailySummaries[0].OvercapacityMinutes != 120 {
		t.Fatalf("daily Sprint summary lost simultaneous variance: %#v", detail.Totals.DailySummaries)
	}
}

func TestDetailTreatsWeekendAllocationAsFullOvercapacity_SPD03_AC19(t *testing.T) {
	repository, database := sprintTestRepository(t)
	day := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)
	seedEligibleTask(t, database, "member-1", "weekend-task")
	if err := database.Model(&taskTestModel{}).Where("id = ?", "weekend-task").Updates(map[string]any{"execution_start": day, "execution_end": day}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&allocationTestModel{TaskID: "weekend-task", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "120"}).Error; err != nil {
		t.Fatal(err)
	}
	sprint, err := domain.NewSprint("sprint-weekend", "Weekend", day, day, []string{"member-1"}, []string{"weekend-task"}, day)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), *sprint); err != nil {
		t.Fatal(err)
	}

	detail, err := repository.Detail(context.Background(), sprint.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Members) != 1 || len(detail.Members[0].DailySummaries) != 1 {
		t.Fatalf("missing weekend daily summary: %#v", detail.Members)
	}
	daily := detail.Members[0].DailySummaries[0]
	if daily.CapacityMinutes != 0 || daily.SelectedAllocationMinutes != 120 || daily.RemainingMinutes != 0 || daily.OvercapacityMinutes != 120 {
		t.Fatalf("zero-capacity allocation was not fully overcapacity: %#v", daily)
	}
}
