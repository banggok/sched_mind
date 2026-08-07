package gormrepo

import (
	"context"
	"errors"
	"math/big"
	"testing"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"gorm.io/gorm"
)

type recommendationRoleRecord struct {
	ID string `gorm:"primaryKey"`
}

func (recommendationRoleRecord) TableName() string { return "roles" }

func prepareRecommendationRepository(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	repository, database := schedulerRepository(t)
	if err := database.AutoMigrate(&recommendationRoleRecord{}); err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&recommendationRoleRecord{ID: "role"}).Error; err != nil {
		t.Fatal(err)
	}
	return repository, database
}

func seedRecommendationMember(t *testing.T, database *gorm.DB, id, name, roleID, dailyCapacity string) {
	t.Helper()
	if err := database.Create(&memberModel{
		ID:               id,
		Name:             name,
		RoleID:           roleID,
		DailyCapacity:    dailyCapacity,
		BufferPercentage: "0",
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func recommendationInput(projectID, taskID string) schedulingdomain.AssigneeRecommendationInput {
	return schedulingdomain.AssigneeRecommendationInput{
		ProjectID:                    projectID,
		TaskID:                       taskID,
		RoleID:                       "role",
		EffortMinutes:                480,
		LagDays:                      0,
		CapacityAllocationPercentage: 50,
		CalculatedOn:                 mustDate("2026-08-05"),
	}
}

func TestRecommendAssigneesAutomaticUsesSharedSchedulerAndRollsBack_D04_D05_D06_D07_D10_D11_AC4_AC6_AC11_AC13_AC18_AC19_AC29_AC30_AC33(t *testing.T) {
	repository, database := prepareRecommendationRepository(t)
	seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
	seedRecommendationMember(t, database, "a", "Ayu", "role", "8")
	seedRecommendationMember(t, database, "b", "Bima", "role", "8")

	higher := schedulableTask("higher", "project", 1, "a", 720, 0)
	higher.CapacityAllocationPercentage = 100
	seedTask(t, database, higher)
	current := schedulableTask("task", "project", 2, "a", 480, 0)
	current.CapacityAllocationPercentage = 40
	current.ExecutionStart = datePointer(mustDate("2026-08-04"))
	current.ExecutionEnd = datePointer(mustDate("2026-08-05"))
	current.CommitmentStart = datePointer(mustDate("2026-08-04"))
	current.CommitmentEnd = datePointer(mustDate("2026-08-05"))
	seedTask(t, database, current)
	lower := schedulableTask("lower", "project", 3, "b", 960, 0)
	lower.CapacityAllocationPercentage = 100
	seedTask(t, database, lower)
	if err := database.Create(&allocationModel{
		TaskID: "task", AssigneeID: "a", Timeline: "execution",
		AllocationDate: mustDate("2026-08-04"), AllocatedMinutes: "240", RemainingCapacityMinutes: "240", Sequence: 1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	beforeTask := loadScheduledTask(t, database, "task")
	var beforeAllocationCount int64
	if err := database.Model(&allocationModel{}).Where("task_id = ?", "task").Count(&beforeAllocationCount).Error; err != nil {
		t.Fatal(err)
	}
	input := recommendationInput("project", "task")
	input.CapacityAllocationPercentage = 40
	result, err := repository.RecommendAssignees(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != schedulingdomain.RecommendationAutomatic || len(result.Items) != 2 {
		t.Fatalf("result=%#v", result)
	}
	if result.Items[0].MemberID != "b" || result.Items[1].MemberID != "a" {
		t.Fatalf("ranked candidates=%#v, want Bima before Ayu", result.Items)
	}
	assertDate(t, "Bima projected finish", result.Items[0].ExecutionEnd, "2026-08-05")
	assertDate(t, "Ayu projected finish", result.Items[1].ExecutionEnd, "2026-08-06")
	if result.Items[0].RemainingExecutionCapacityMinutes != 360 || result.Items[1].RemainingExecutionCapacityMinutes != 360 {
		t.Fatalf("remaining metrics=%#v", result.Items)
	}
	if result.ProjectScheduleVersions["project"] != 0 {
		t.Fatalf("snapshot versions=%v", result.ProjectScheduleVersions)
	}

	afterTask := loadScheduledTask(t, database, "task")
	if afterTask.AssigneeID == nil || *afterTask.AssigneeID != *beforeTask.AssigneeID ||
		afterTask.CapacityAllocationPercentage != beforeTask.CapacityAllocationPercentage ||
		!afterTask.ExecutionStart.Equal(*beforeTask.ExecutionStart) ||
		!afterTask.ExecutionEnd.Equal(*beforeTask.ExecutionEnd) {
		t.Fatalf("recommendation mutated Task: before=%#v after=%#v", beforeTask, afterTask)
	}
	var afterAllocationCount int64
	if err := database.Model(&allocationModel{}).Where("task_id = ?", "task").Count(&afterAllocationCount).Error; err != nil {
		t.Fatal(err)
	}
	if afterAllocationCount != beforeAllocationCount {
		t.Fatalf("allocation count=%d, want %d", afterAllocationCount, beforeAllocationCount)
	}
}

func TestRecommendAssigneesManualUsesDraftStartKeepsOtherFixedAndIgnoresManualEnd_D04_D08_D09_D10_AC5_AC8_AC9_AC10_AC34(t *testing.T) {
	repository, database := prepareRecommendationRepository(t)
	manualProject := automaticProject("manual", 1, "2026-07-01", 0)
	manualProject.AutomaticScheduling = false
	seedProject(t, database, manualProject)
	seedRecommendationMember(t, database, "a", "Ayu", "role", "8")
	seedRecommendationMember(t, database, "b", "Bima", "role", "8")

	fixed := schedulableTask("fixed", "manual", 1, "a", 480, 0)
	fixed.ExecutionStart = datePointer(mustDate("2026-08-10"))
	fixed.ExecutionEnd = datePointer(mustDate("2026-08-10"))
	fixed.CommitmentStart = fixed.ExecutionStart
	fixed.CommitmentEnd = fixed.ExecutionEnd
	seedTask(t, database, fixed)
	current := schedulableTask("task", "manual", 2, "a", 480, 0)
	current.ExecutionStart = datePointer(mustDate("2026-07-01"))
	current.ExecutionEnd = datePointer(mustDate("2026-12-31"))
	current.CommitmentStart = current.ExecutionStart
	current.CommitmentEnd = current.ExecutionEnd
	current.CapacityAllocationPercentage = 100
	seedTask(t, database, current)

	input := recommendationInput("manual", "task")
	input.CapacityAllocationPercentage = 100
	draftStart := mustDate("2026-08-10")
	input.ExecutionStart = &draftStart
	result, err := repository.RecommendAssignees(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != schedulingdomain.RecommendationManualAdvisory || len(result.Items) != 2 {
		t.Fatalf("result=%#v", result)
	}
	if result.Items[0].MemberID != "b" || result.Items[1].MemberID != "a" {
		t.Fatalf("ranked candidates=%#v", result.Items)
	}
	assertDate(t, "Bima estimated finish", result.Items[0].ExecutionEnd, "2026-08-10")
	assertDate(t, "Ayu estimated finish", result.Items[1].ExecutionEnd, "2026-08-11")
	stored := loadScheduledTask(t, database, "task")
	assertDate(t, "confirmed manual start", stored.ExecutionStart, "2026-07-01")
	assertDate(t, "confirmed manual end", stored.ExecutionEnd, "2026-12-31")
}

func TestRecommendAssigneesManualFallsBackToApplicationToday_D08_D09_AC9_AC34(t *testing.T) {
	repository, database := prepareRecommendationRepository(t)
	manualProject := automaticProject("manual", 1, "2026-07-01", 0)
	manualProject.AutomaticScheduling = false
	manualProject.SchedulingStartDate = nil
	seedProject(t, database, manualProject)
	seedRecommendationMember(t, database, "a", "Ayu", "role", "8")
	seedTask(t, database, taskModel{
		ID: "task", ProjectID: "manual", ParentKey: "", Position: 1, Name: "Task",
		CapacityAllocationPercentage: 100,
	})
	input := recommendationInput("manual", "task")
	input.CapacityAllocationPercentage = 100
	result, err := repository.RecommendAssignees(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	assertDate(t, "today fallback finish", result.Items[0].ExecutionEnd, "2026-08-05")
	if result.CalculatedOnDate.Format("2006-01-02") != "2026-08-05" {
		t.Fatalf("calculated date=%v", result.CalculatedOnDate)
	}
}

func TestRecommendAssigneesManualUsesLaterOfProjectStartAndApplicationToday_D08_D09_AC9_AC34(t *testing.T) {
	tests := []struct {
		name         string
		projectStart string
		wantFinish   string
	}{
		{name: "future Project Start wins", projectStart: "2026-08-10", wantFinish: "2026-08-10"},
		{name: "past Project Start yields to application today", projectStart: "2026-07-01", wantFinish: "2026-08-05"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, database := prepareRecommendationRepository(t)
			project := automaticProject("manual", 1, test.projectStart, 0)
			project.AutomaticScheduling = false
			seedProject(t, database, project)
			seedRecommendationMember(t, database, "a", "Ayu", "role", "8")
			seedTask(t, database, taskModel{
				ID: "task", ProjectID: "manual", ParentKey: "", Position: 1, Name: "Task",
				CapacityAllocationPercentage: 100,
			})

			input := recommendationInput("manual", "task")
			input.CapacityAllocationPercentage = 100
			result, err := repository.RecommendAssignees(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			assertDate(t, "manual advisory finish", result.Items[0].ExecutionEnd, test.wantFinish)
		})
	}
}

func TestRecommendAssigneesAutomaticWithoutAnchorDoesNotInventToday_D07_AC7(t *testing.T) {
	repository, database := prepareRecommendationRepository(t)
	project := automaticProject("project", 1, "2026-08-03", 0)
	project.SchedulingStartDate = nil
	seedProject(t, database, project)
	seedRecommendationMember(t, database, "a", "Ayu", "role", "8")
	seedTask(t, database, taskModel{
		ID: "task", ProjectID: "project", ParentKey: "", Position: 1, Name: "Task",
		CapacityAllocationPercentage: 100,
	})
	input := recommendationInput("project", "task")
	result, err := repository.RecommendAssignees(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].RankGroup != schedulingdomain.RecommendationNoCompletion || result.Items[0].ExecutionEnd != nil {
		t.Fatalf("items=%#v", result.Items)
	}
	if result.Items[0].ReasonCode == nil || *result.Items[0].ReasonCode != recommendationReasonAutomaticAnchorMissing {
		t.Fatalf("reason=%v", result.Items[0].ReasonCode)
	}
}

func TestRecommendAssigneesRanksFeasibleBeforeAddedOvercapacityAndIsolatesNoCapacity_D12_D13_AC16_AC20_AC21_AC23_AC32(t *testing.T) {
	repository, database := prepareRecommendationRepository(t)
	seedProject(t, database, automaticProject("candidate", 1, "2026-08-03", 0))
	lowerManual := automaticProject("lower-manual", 2, "2026-08-03", 0)
	lowerManual.AutomaticScheduling = false
	seedProject(t, database, lowerManual)
	seedRecommendationMember(t, database, "over", "Alif", "role", "8")
	seedRecommendationMember(t, database, "feasible", "Dewi", "role", "8")
	seedRecommendationMember(t, database, "none", "Gabby", "role", "8")
	if err := database.Create(&capacityOverrideModel{
		ID: "none-zero", TeamMemberID: "none", StartDate: mustDate("2026-08-03"),
		EndDate: mustDate("2026-08-03").AddDate(0, 0, maximumScheduleDays), Capacity: "0",
	}).Error; err != nil {
		t.Fatal(err)
	}

	fixed := schedulableTask("manual-fixed", "lower-manual", 1, "over", 480, 0)
	fixed.ExecutionStart = datePointer(mustDate("2026-08-03"))
	fixed.ExecutionEnd = datePointer(mustDate("2026-08-03"))
	fixed.CommitmentStart = fixed.ExecutionStart
	fixed.CommitmentEnd = fixed.ExecutionEnd
	seedTask(t, database, fixed)
	seedTask(t, database, taskModel{
		ID: "task", ProjectID: "candidate", ParentKey: "", Position: 1, Name: "Task",
		CapacityAllocationPercentage: 100,
	})
	input := recommendationInput("candidate", "task")
	input.EffortMinutes = 240
	input.CapacityAllocationPercentage = 100
	result, err := repository.RecommendAssignees(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 3 {
		t.Fatalf("items=%#v", result.Items)
	}
	if result.Items[0].MemberID != "feasible" || result.Items[0].RankGroup != schedulingdomain.RecommendationFeasible {
		t.Fatalf("first=%#v", result.Items[0])
	}
	if result.Items[1].MemberID != "over" || result.Items[1].RankGroup != schedulingdomain.RecommendationOvercapacity || result.Items[1].IncrementalOvercapacityMinutes != 240 {
		t.Fatalf("overcapacity=%#v", result.Items[1])
	}
	if result.Items[2].MemberID != "none" || result.Items[2].RankGroup != schedulingdomain.RecommendationNoCompletion || result.Items[2].ReasonCode == nil {
		t.Fatalf("no completion=%#v", result.Items[2])
	}
}

func TestIncrementalOvercapacityCountsOnlyPositivePerDateIncrease_D12_AC20_AC21(t *testing.T) {
	state := &portfolioState{
		projects: map[string]projectModel{
			"project": {ID: "project", Status: "open", AutomaticScheduling: true},
		},
		members: map[string]memberModel{
			"member": {ID: "member", DailyCapacity: "8", BufferPercentage: "0"},
		},
		holidays:  make(map[string]struct{}),
		overrides: make(map[string][]capacityOverrideModel),
	}
	dateOne := mustDate("2026-08-03")
	dateTwo := mustDate("2026-08-04")
	calendar := func(first, second int64) *allocationCalendar {
		return &allocationCalendar{
			state:    state,
			timeline: schedulingdomain.Execution,
			allocations: map[string]map[string][]dailyAllocation{
				"member": {
					schedulingdomain.DateKey(dateOne): {{TaskID: "one", MemberID: "member", Date: dateOne, Minutes: big.NewRat(first, 1)}},
					schedulingdomain.DateKey(dateTwo): {{TaskID: "two", MemberID: "member", Date: dateTwo, Minutes: big.NewRat(second, 1)}},
				},
			},
		}
	}

	baseline := calendar(600, 480)
	unchanged := calendar(600, 480)
	value, err := incrementalOvercapacityMinutes(baseline, unchanged, "member", "project")
	if err != nil || value != 0 {
		t.Fatalf("unchanged baseline overcapacity=%d err=%v", value, err)
	}

	shifted := calendar(480, 600)
	value, err = incrementalOvercapacityMinutes(baseline, shifted, "member", "project")
	if err != nil || value != 120 {
		t.Fatalf("cross-date incremental overcapacity=%d err=%v, want 120", value, err)
	}
}

func TestRecommendAssigneesLoadsOnlyActiveMembersForSelectedRole_D18_AC12(t *testing.T) {
	repository, database := prepareRecommendationRepository(t)
	if err := database.Create(&recommendationRoleRecord{ID: "other-role"}).Error; err != nil {
		t.Fatal(err)
	}
	seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
	seedRecommendationMember(t, database, "active", "Ayu", "role", "8")
	seedRecommendationMember(t, database, "other", "Bima", "other-role", "8")
	deletedAt := mustDate("2026-08-01")
	if err := database.Create(&memberModel{
		ID: "deleted", Name: "Dewi", RoleID: "role", DailyCapacity: "8", BufferPercentage: "0", DeletedAt: &deletedAt,
	}).Error; err != nil {
		t.Fatal(err)
	}
	seedTask(t, database, taskModel{
		ID: "task", ProjectID: "project", ParentKey: "", Position: 1, Name: "Task",
		CapacityAllocationPercentage: 100,
	})

	result, err := repository.RecommendAssignees(context.Background(), recommendationInput("project", "task"))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].MemberID != "active" {
		t.Fatalf("candidate items=%#v", result.Items)
	}
}

func TestRecommendAssigneesSchedulerFailureRollsBackTaskDependencyAllocationAndVersion_D06_D17_AC31_AC33(t *testing.T) {
	repository, database := prepareRecommendationRepository(t)
	project := automaticProject("project", 1, "2026-08-03", 0)
	project.ScheduleVersion = 9
	seedProject(t, database, project)
	seedRecommendationMember(t, database, "candidate", "Ayu", "role", "8")
	seedRecommendationMember(t, database, "broken", "Broken", "other-role", "not-a-capacity")
	seedTask(t, database, schedulableTask("blocking", "project", 1, "broken", 480, 0))
	current := schedulableTask("task", "project", 2, "candidate", 480, 0)
	current.CapacityAllocationPercentage = 40
	current.ExecutionStart = datePointer(mustDate("2026-08-04"))
	current.ExecutionEnd = datePointer(mustDate("2026-08-05"))
	current.CommitmentStart = current.ExecutionStart
	current.CommitmentEnd = current.ExecutionEnd
	seedTask(t, database, current)
	seedDependency(t, database, dependencyModel{ID: "dependency", BlockingTaskID: "blocking", BlockedTaskID: "task"})
	if err := database.Create(&allocationModel{
		TaskID: "task", AssigneeID: "candidate", Timeline: "execution",
		AllocationDate: mustDate("2026-08-04"), AllocatedMinutes: "240", RemainingCapacityMinutes: "240", Sequence: 1,
	}).Error; err != nil {
		t.Fatal(err)
	}

	_, err := repository.RecommendAssignees(context.Background(), recommendationInput("project", "task"))
	if !errors.Is(err, schedulingdomain.ErrAssigneeRecommendationUnavailable) {
		t.Fatalf("error=%v, want unavailable", err)
	}

	storedTask := loadScheduledTask(t, database, "task")
	if storedTask.AssigneeID == nil || *storedTask.AssigneeID != "candidate" || storedTask.CapacityAllocationPercentage != 40 {
		t.Fatalf("stored Task=%#v", storedTask)
	}
	assertDate(t, "stored execution start", storedTask.ExecutionStart, "2026-08-04")
	assertDate(t, "stored execution end", storedTask.ExecutionEnd, "2026-08-05")
	var storedDependency dependencyModel
	if err := database.First(&storedDependency, "id = ?", "dependency").Error; err != nil {
		t.Fatal(err)
	}
	if storedDependency.BlockingTaskID != "blocking" || storedDependency.BlockedTaskID != "task" {
		t.Fatalf("dependency=%#v", storedDependency)
	}
	var allocationCount int64
	if err := database.Model(&allocationModel{}).Where("task_id = ?", "task").Count(&allocationCount).Error; err != nil {
		t.Fatal(err)
	}
	if allocationCount != 1 {
		t.Fatalf("allocation count=%d, want 1", allocationCount)
	}
	var storedProject projectModel
	if err := database.First(&storedProject, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	if storedProject.ScheduleVersion != 9 {
		t.Fatalf("schedule version=%d, want 9", storedProject.ScheduleVersion)
	}
}

func TestRecommendAssigneesRejectsInvalidRoleAndNonRecommendableLifecycle_D03_AC3_AC1(t *testing.T) {
	repository, database := prepareRecommendationRepository(t)
	seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
	seedRecommendationMember(t, database, "a", "Ayu", "role", "8")
	seedTask(t, database, taskModel{ID: "task", ProjectID: "project", ParentKey: "", Position: 1, Name: "Task", CapacityAllocationPercentage: 100})

	invalid := recommendationInput("project", "task")
	invalid.RoleID = "unknown"
	if _, err := repository.RecommendAssignees(context.Background(), invalid); err != schedulingdomain.ErrAssigneeRecommendationInputInvalid {
		t.Fatalf("invalid role error=%v", err)
	}

	unknownTask := recommendationInput("project", "unknown")
	if _, err := repository.RecommendAssignees(context.Background(), unknownTask); err != schedulingdomain.ErrAssigneeRecommendationInputInvalid {
		t.Fatalf("invalid Task context error=%v", err)
	}

	actual := mustDate("2026-08-03")
	if err := database.Model(&taskModel{}).Where("id = ?", "task").Updates(map[string]any{"actual_start": actual, "actual_end": actual}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repository.RecommendAssignees(context.Background(), recommendationInput("project", "task")); err != schedulingdomain.ErrTaskNotRecommendable {
		t.Fatalf("completed Task error=%v", err)
	}

	if err := database.Model(&projectModel{}).Where("id = ?", "project").Update("status", "closed").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := repository.RecommendAssignees(context.Background(), recommendationInput("project", "task")); err != schedulingdomain.ErrTaskNotRecommendable {
		t.Fatalf("closed Project error=%v", err)
	}
}

func TestRecommendAssigneesUsesGroupOverrideOnWhenProjectIsManual_US44_AC34(t *testing.T) {
	repository, database := prepareRecommendationRepository(t)
	project := automaticProject("project", 1, "2026-07-01", 0)
	project.AutomaticScheduling = false
	seedProject(t, database, project)
	seedRecommendationMember(t, database, "a", "Ayu", "role", "8")

	groupID := "group"
	groupAutomatic := true
	groupAnchor := mustDate("2026-08-10")
	seedTask(t, database, taskModel{
		ID:                       groupID,
		ProjectID:                "project",
		ParentKey:                "",
		Position:                 1,
		Name:                     "Platform",
		GroupSchedulingSource:    "override",
		GroupAutomaticScheduling: &groupAutomatic,
		GroupSchedulingStartDate: &groupAnchor,
		GroupLocalStatus:         "open",
	})
	seedTask(t, database, taskModel{
		ID:                           "task",
		ProjectID:                    "project",
		ParentID:                     &groupID,
		ParentKey:                    groupID,
		Position:                     1,
		Name:                         "Task",
		CapacityAllocationPercentage: 100,
		GroupSchedulingSource:        "inherit",
		GroupLocalStatus:             "open",
	})

	input := recommendationInput("project", "task")
	input.CapacityAllocationPercentage = 100
	result, err := repository.RecommendAssignees(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != schedulingdomain.RecommendationAutomatic {
		t.Fatalf("mode=%q, want automatic from Group override", result.Mode)
	}
	if len(result.Items) != 1 {
		t.Fatalf("items=%#v, want one candidate", result.Items)
	}
	assertDate(t, "Group automatic projected finish", result.Items[0].ExecutionEnd, "2026-08-10")
}

func TestRecommendAssigneesUsesGroupOverrideOffWhenProjectIsAutomatic_US44_AC34(t *testing.T) {
	repository, database := prepareRecommendationRepository(t)
	seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
	seedRecommendationMember(t, database, "a", "Ayu", "role", "8")

	groupID := "group"
	groupAutomatic := false
	seedTask(t, database, taskModel{
		ID:                       groupID,
		ProjectID:                "project",
		ParentKey:                "",
		Position:                 1,
		Name:                     "Platform",
		GroupSchedulingSource:    "override",
		GroupAutomaticScheduling: &groupAutomatic,
		GroupLocalStatus:         "open",
	})
	seedTask(t, database, taskModel{
		ID:                           "task",
		ProjectID:                    "project",
		ParentID:                     &groupID,
		ParentKey:                    groupID,
		Position:                     1,
		Name:                         "Task",
		CapacityAllocationPercentage: 100,
		GroupSchedulingSource:        "inherit",
		GroupLocalStatus:             "open",
	})

	input := recommendationInput("project", "task")
	input.CapacityAllocationPercentage = 100
	draftStart := mustDate("2026-08-10")
	input.ExecutionStart = &draftStart
	result, err := repository.RecommendAssignees(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Mode != schedulingdomain.RecommendationManualAdvisory {
		t.Fatalf("mode=%q, want manual advisory from Group override", result.Mode)
	}
}
