package gormrepo

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

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
	ID             string `gorm:"primaryKey"`
	ProjectID      string
	ParentID       *string
	AssigneeID     *string
	ExecutionStart *time.Time
	ExecutionEnd   *time.Time
	ActualStart    *time.Time
	ActualEnd      *time.Time
	ParentKey      string
	Position       int
	Name           string
	UpdatedAt      time.Time
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
	if err := database.Create(&taskTestModel{ID: taskID, ProjectID: projectID, AssigneeID: &memberID, ExecutionStart: &now, ExecutionEnd: &now, Name: taskID, Position: 1, UpdatedAt: now}).Error; err != nil {
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

func TestDetailComposesLiveCapacityOutsideAllocationAndNeedsReview_AC17To21And36To55(t *testing.T) {
	repository, database := sprintTestRepository(t)
	seedEligibleTask(t, database, "member-1", "task-1")
	day := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	if err := database.Model(&memberModel{}).Where("id = ?", "member-1").Updates(map[string]any{"daily_capacity": "8.0", "buffer_percentage": "25.0"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&taskTestModel{}).Where("id = ?", "task-1").Updates(map[string]any{"execution_start": day.AddDate(0, 0, -1), "execution_end": day.AddDate(0, 0, 4)}).Error; err != nil {
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
		{TaskID: "task-1", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "120"},
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
	if len(detail.Tasks) != 1 || detail.Tasks[0].InSprintAllocationMinutes != 120 || detail.Tasks[0].OutsideAllocationMinutes != 60 || detail.Tasks[0].TotalAllocationMinutes != 180 {
		t.Fatalf("unexpected task allocation: %#v", detail.Tasks)
	}
	if detail.Members[0].InSprintAllocationMinutes != 120 || detail.Totals.SelectedMemberAllocationMinutes != 120 || detail.ProjectionToken == "" {
		t.Fatalf("unexpected selected utilization: %#v", detail)
	}

	seedEligibleTask(t, database, "member-2", "unused-task")
	if err := database.Model(&taskTestModel{}).Where("id = ?", "task-1").Update("assignee_id", "member-2").Error; err != nil {
		t.Fatal(err)
	}
	drifted, err := repository.Detail(context.Background(), sprint.ID)
	if err != nil {
		t.Fatalf("drifted detail: %v", err)
	}
	if len(drifted.Tasks[0].Warnings) == 0 || drifted.Members[0].InSprintAllocationMinutes != 0 || drifted.Totals.NeedsReviewAllocationMinutes != 120 {
		t.Fatalf("drift was not separated from selected-member utilization: %#v", drifted)
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
		if err := database.Model(&taskTestModel{}).Where("id = ?", taskID).Updates(map[string]any{"execution_start": day, "execution_end": end}).Error; err != nil {
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
		{TaskID: "fill-high", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "240"},
		{TaskID: "fill-low", AssigneeID: "member-1", Timeline: "execution", AllocationDate: day, AllocatedMinutes: "240"},
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
}

func TestCandidatesReturnOnlyUnselectedEligibleTasksIncludingZeroInSprintAllocation_AC40And41(t *testing.T) {
	repository, database := sprintTestRepository(t)
	day := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	seedEligibleTask(t, database, "member-1", "selected")
	seedEligibleTask(t, database, "member-1", "eligible-zero")
	seedEligibleTask(t, database, "member-1", "unscheduled")
	seedEligibleTask(t, database, "member-1", "completed")
	seedEligibleTask(t, database, "member-2", "other-member")
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
}
