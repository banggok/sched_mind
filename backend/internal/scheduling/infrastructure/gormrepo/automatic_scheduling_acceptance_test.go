package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	dependencyapplication "github.com/banggok/sched_mind/backend/internal/dependencies/application"
	dependencyrepo "github.com/banggok/sched_mind/backend/internal/dependencies/infrastructure/gormrepo"
	projectapplication "github.com/banggok/sched_mind/backend/internal/projects/application"
	projectdomain "github.com/banggok/sched_mind/backend/internal/projects/domain"
	projectrepo "github.com/banggok/sched_mind/backend/internal/projects/infrastructure/gormrepo"
	schedulingapplication "github.com/banggok/sched_mind/backend/internal/scheduling/application"
	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
	wbsapplication "github.com/banggok/sched_mind/backend/internal/wbs/application"
	wbsgormrepo "github.com/banggok/sched_mind/backend/internal/wbs/infrastructure/gormrepo"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type acceptanceProjectRecord struct {
	ID                       string `gorm:"primaryKey"`
	Name                     string
	NameKey                  string
	Status                   string
	Priority                 int
	AutomaticScheduling      bool
	AutoCalculateDate        bool
	SchedulingStartDate      *time.Time
	ProjectBuffer            int
	StartDate                *time.Time
	EndDate                  *time.Time
	ScheduleVersion          int64
	ClosedAt                 *time.Time
	LockedExecutionSnapshot  *string
	LockedCommitmentSnapshot *string
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func TestAutomaticSchedulingCapacityAndUnscheduledAcceptance_AC1_AC4_AC5_AC6_AC7_AC8_AC9_AC10_AC13_AC14_AC33_AC34(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	roleID := "role"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	members := []acceptanceMemberRecord{
		{ID: "exact", Name: "Exact", RoleID: roleID, DailyCapacity: "6", BufferPercentage: "30", CreatedAt: now, UpdatedAt: now},
		{ID: "zero", Name: "Zero", RoleID: roleID, DailyCapacity: "0", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
		{ID: "manual", Name: "Manual", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
		{ID: "lag", Name: "Lag", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&members).Error; err != nil {
		t.Fatal(err)
	}
	anchor := mustDate("2026-08-03")
	friday := mustDate("2026-08-07")
	manualStart := mustDate("2026-08-12")
	manualEnd := mustDate("2026-08-13")
	projects := []acceptanceProjectRecord{
		{ID: "exact", Name: "Exact", NameKey: "exact", Status: "open", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 20, CreatedAt: now, UpdatedAt: now},
		{ID: "missing", Name: "Missing", NameKey: "missing", Status: "open", Priority: 2, AutomaticScheduling: true, AutoCalculateDate: true, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
		{ID: "zero", Name: "Zero", NameKey: "zero", Status: "open", Priority: 3, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
		{ID: "manual", Name: "Manual", NameKey: "manual", Status: "open", Priority: 4, AutomaticScheduling: false, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
		{ID: "lag", Name: "Lag", NameKey: "lag", Status: "open", Priority: 5, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &friday, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&projects).Error; err != nil {
		t.Fatal(err)
	}
	// Effort remains valid under US-4.1 (30-minute increments). Execution
	// capacities are calculated from the same raw Daily Capacity. Execution
	// rounds after Member Buffer; Commitment applies both buffers and then rounds.
	exactEffort, smallEffort, manualEffort := 570, 60, 480
	tasks := []acceptanceTaskRecord{
		{ID: "exact-task", ProjectID: "exact", ParentKey: "", Name: "Exact Task", NameKey: "exact task", Position: 1, RoleID: &roleID, AssigneeID: stringPointer("exact"), EffortMinutes: &exactEffort, CreatedAt: now, UpdatedAt: now},
		{ID: "missing-task", ProjectID: "missing", ParentKey: "", Name: "Missing Task", NameKey: "missing task", Position: 1, RoleID: &roleID, AssigneeID: stringPointer("exact"), EffortMinutes: &smallEffort, CreatedAt: now, UpdatedAt: now},
		{ID: "zero-task", ProjectID: "zero", ParentKey: "", Name: "Zero Task", NameKey: "zero task", Position: 1, RoleID: &roleID, AssigneeID: stringPointer("zero"), EffortMinutes: &smallEffort, CreatedAt: now, UpdatedAt: now},
		{ID: "manual-task", ProjectID: "manual", ParentKey: "", Name: "Manual Task", NameKey: "manual task", Position: 1, RoleID: &roleID, AssigneeID: stringPointer("manual"), EffortMinutes: &manualEffort, ExecutionStart: &manualStart, ExecutionEnd: &manualEnd, CommitmentStart: &manualStart, CommitmentEnd: &manualEnd, CreatedAt: now, UpdatedAt: now},
		{ID: "lag-task", ProjectID: "lag", ParentKey: "", Name: "Lag Task", NameKey: "lag task", Position: 1, RoleID: &roleID, AssigneeID: stringPointer("lag"), EffortMinutes: &smallEffort, LagDays: 1, CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceOverrideRecord{ID: "override", TeamMemberID: "exact", StartDate: anchor, EndDate: anchor, Capacity: "8"}).Error; err != nil {
		t.Fatal(err)
	}
	holiday := mustDate("2026-08-04")
	if err := database.Create(&acceptanceHolidayDateRecord{PublicHolidayID: "holiday", Date: holiday}).Error; err != nil {
		t.Fatal(err)
	}

	scheduler := schedulingapplication.NewService(NewWithDependencies(database, func() time.Time { return now }, func() (string, error) { return "automatic", nil }))
	if err := scheduler.RecalculateActiveProjects(context.Background()); err != nil {
		t.Fatal(err)
	}

	exact := loadAcceptanceTask(t, database, "exact-task")
	assertDate(t, "exact execution start", exact.ExecutionStart, "2026-08-03")
	assertDate(t, "exact execution end", exact.ExecutionEnd, "2026-08-05")
	assertDate(t, "exact commitment start", exact.CommitmentStart, "2026-08-03")
	assertDate(t, "exact commitment end", exact.CommitmentEnd, "2026-08-06")
	executionRows := loadAcceptanceAllocations(t, database, "exact-task", string(schedulingdomain.Execution))
	if len(executionRows) != 2 || executionRows[0].AllocatedMinutes != "330.000000" || executionRows[1].AllocatedMinutes != "240.000000" || schedulingdomain.DateKey(executionRows[1].AllocationDate) != "2026-08-05" {
		t.Fatalf("exact execution allocations=%#v", executionRows)
	}
	commitmentRows := loadAcceptanceAllocations(t, database, "exact-task", string(schedulingdomain.Commitment))
	if len(commitmentRows) != 3 || commitmentRows[0].AllocatedMinutes != "270.000000" || commitmentRows[1].AllocatedMinutes != "210.000000" || commitmentRows[2].AllocatedMinutes != "90.000000" {
		t.Fatalf("exact commitment allocations=%#v", commitmentRows)
	}
	missing := loadAcceptanceTask(t, database, "missing-task")
	if missing.ExecutionStart != nil || missing.ExecutionUnscheduledReason == nil || *missing.ExecutionUnscheduledReason != reasonMissingAnchor {
		t.Fatalf("missing anchor projection=%#v", missing)
	}
	zero := loadAcceptanceTask(t, database, "zero-task")
	if zero.ExecutionStart != nil || zero.CommitmentStart != nil || zero.ExecutionUnscheduledReason == nil || *zero.ExecutionUnscheduledReason != reasonZeroCapacity {
		t.Fatalf("zero-capacity projection=%#v", zero)
	}
	manual := loadAcceptanceTask(t, database, "manual-task")
	assertDate(t, "manual execution start", manual.ExecutionStart, "2026-08-12")
	assertDate(t, "manual commitment end", manual.CommitmentEnd, "2026-08-13")
	if rows := loadAcceptanceAllocations(t, database, "manual-task", string(schedulingdomain.Execution)); len(rows) != 2 || rows[0].AllocatedMinutes != "240.000000" || rows[1].AllocatedMinutes != "240.000000" {
		t.Fatalf("manual task fixed allocations=%#v", rows)
	}
	lagged := loadAcceptanceTask(t, database, "lag-task")
	assertDate(t, "lag without dependency skips weekend", lagged.ExecutionStart, "2026-08-10")
}

func TestManualAndAutomaticCrossProjectPriorityAcceptance_US63_AC19(t *testing.T) {
	tests := []struct {
		name               string
		automaticPriority  int
		manualPriority     int
		wantAutomaticEnd   string
		wantAutomaticDaily []string
	}{
		{
			name:               "lower-priority manual overlap is accepted without shifting or warning",
			automaticPriority:  1,
			manualPriority:     2,
			wantAutomaticEnd:   "2026-08-04",
			wantAutomaticDaily: []string{"480.000000", "480.000000"},
		},
		{
			name:               "higher-priority manual task shifts automatic work through shared allocation",
			automaticPriority:  2,
			manualPriority:     1,
			wantAutomaticEnd:   "2026-08-05",
			wantAutomaticDaily: []string{"240.000000", "240.000000", "480.000000"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			database := acceptanceDatabase(t)
			now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
			anchor := mustDate("2026-08-03")
			manualEnd := mustDate("2026-08-04")
			roleID, memberID := "role", "member"
			if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
				t.Fatal(err)
			}
			if err := database.Create(&acceptanceMemberRecord{
				ID: memberID, Name: "Rani", RoleID: roleID,
				DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now,
			}).Error; err != nil {
				t.Fatal(err)
			}
			projects := []acceptanceProjectRecord{
				{ID: "automatic", Name: "Automatic", NameKey: "automatic", Status: "open", Priority: tc.automaticPriority, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
				{ID: "manual", Name: "Manual", NameKey: "manual", Status: "open", Priority: tc.manualPriority, AutomaticScheduling: false, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
			}
			if err := database.Create(&projects).Error; err != nil {
				t.Fatal(err)
			}
			automaticEffort, manualEffort := 960, 480
			tasks := []acceptanceTaskRecord{
				{ID: "automatic-task", ProjectID: "automatic", ParentKey: "", Name: "Automatic Task", NameKey: "automatic task", Position: 1, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &automaticEffort, CapacityAllocationPercentage: 100, CreatedAt: now, UpdatedAt: now},
				{ID: "manual-task", ProjectID: "manual", ParentKey: "", Name: "Manual Task", NameKey: "manual task", Position: 1, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &manualEffort, CapacityAllocationPercentage: 100, ExecutionStart: &anchor, ExecutionEnd: &manualEnd, CommitmentStart: &anchor, CommitmentEnd: &manualEnd, CreatedAt: now, UpdatedAt: now},
			}
			if err := database.Create(&tasks).Error; err != nil {
				t.Fatal(err)
			}

			scheduler := schedulingapplication.NewService(NewWithDependencies(
				database,
				func() time.Time { return now },
				func() (string, error) { return "automatic", nil },
			))
			if err := scheduler.RecalculateActiveProjects(context.Background()); err != nil {
				t.Fatal(err)
			}

			automatic := loadAcceptanceTask(t, database, "automatic-task")
			assertDate(t, "automatic start", automatic.ExecutionStart, "2026-08-03")
			assertDate(t, "automatic end", automatic.ExecutionEnd, tc.wantAutomaticEnd)
			rows := loadAcceptanceAllocations(t, database, "automatic-task", string(schedulingdomain.Execution))
			if len(rows) != len(tc.wantAutomaticDaily) {
				t.Fatalf("automatic allocations = %#v, want %d rows", rows, len(tc.wantAutomaticDaily))
			}
			for index, want := range tc.wantAutomaticDaily {
				if rows[index].AllocatedMinutes != want {
					t.Fatalf("automatic allocation[%d] = %s, want %s", index, rows[index].AllocatedMinutes, want)
				}
			}

			wbsService := wbsapplication.NewServiceWithDependencies(
				wbsgormrepo.New(database),
				scheduler,
				func() time.Time { return now },
				func() (string, error) { return "unused", nil },
			)
			manualAllocations, err := wbsService.Allocations(context.Background(), "manual", "manual-task")
			if err != nil {
				t.Fatal(err)
			}
			if len(manualAllocations.Execution) != 2 {
				t.Fatalf("manual allocation detail = %#v, want two rows", manualAllocations.Execution)
			}
			for _, row := range manualAllocations.Execution {
				if row.AllocatedMinutes != 240 {
					t.Fatalf("manual allocated minutes = %d, want 240", row.AllocatedMinutes)
				}
				if row.OvercapacityMinutes != 0 {
					t.Fatalf("manual overcapacity warning = %d, want none", row.OvercapacityMinutes)
				}
			}
		})
	}
}

func TestAutomaticSchedulingLifecycleAndPriorityAcceptance_AC29_AC30_AC31_AC32_AC35(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	anchor := mustDate("2026-08-03")
	roleID, memberID := "role", "member"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceMemberRecord{ID: memberID, Name: "Rani", RoleID: roleID, DailyCapacity: "5", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	projects := []acceptanceProjectRecord{
		{ID: "first", Name: "First", NameKey: "first", Status: "open", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
		{ID: "second", Name: "Second", NameKey: "second", Status: "open", Priority: 2, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&projects).Error; err != nil {
		t.Fatal(err)
	}
	effort := 300
	tasks := []acceptanceTaskRecord{
		{ID: "first-task", ProjectID: "first", ParentKey: "", Name: "First Task", NameKey: "first task", Position: 1, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &effort, CreatedAt: now, UpdatedAt: now},
		{ID: "second-task", ProjectID: "second", ParentKey: "", Name: "Second Task", NameKey: "second task", Position: 1, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &effort, CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	automaticID := 0
	scheduler := schedulingapplication.NewService(NewWithDependencies(database, func() time.Time { return now }, func() (string, error) {
		automaticID++
		return fmt.Sprintf("automatic-%d", automaticID), nil
	}))
	if err := scheduler.RecalculateActiveProjects(context.Background()); err != nil {
		t.Fatal(err)
	}
	assertDate(t, "first before priority move", loadAcceptanceTask(t, database, "first-task").ExecutionStart, "2026-08-03")
	assertDate(t, "second before priority move", loadAcceptanceTask(t, database, "second-task").ExecutionStart, "2026-08-04")

	projectService := projectapplication.NewServiceWithDependencies(projectrepo.New(database), scheduler, func() time.Time { return now.Add(time.Hour) }, func() (string, error) { return "unused", nil })
	_, err := projectService.MovePriority(context.Background(), "second", projectdomain.PriorityUp)
	priorityImpact := requireOpenProjectImpact(t, err, "first")
	if _, err := projectService.MovePriority(acceptanceImpactContext(t, priorityImpact.Token), "second", projectdomain.PriorityUp); err != nil {
		t.Fatalf("confirm priority impact: %v", err)
	}
	assertDate(t, "second after priority move", loadAcceptanceTask(t, database, "second-task").ExecutionStart, "2026-08-03")
	assertDate(t, "first after priority move", loadAcceptanceTask(t, database, "first-task").ExecutionStart, "2026-08-04")

	locked, err := projectService.ChangeStatus(context.Background(), "second", projectdomain.StatusLocked)
	if err != nil || locked == nil || locked.LockedExecutionSnapshot == nil || locked.LockedCommitmentSnapshot == nil {
		t.Fatalf("lock: %#v %v", locked, err)
	}
	lockedStart := loadAcceptanceTask(t, database, "second-task").ExecutionStart
	if err := database.Model(&acceptanceMemberRecord{}).Where("id = ?", memberID).Update("daily_capacity", "4").Error; err != nil {
		t.Fatal(err)
	}
	if err := scheduler.RecalculateActiveProjects(context.Background()); err != nil {
		t.Fatal(err)
	}
	lockedAfterCapacity := loadAcceptanceTask(t, database, "second-task")
	if lockedStart == nil || lockedAfterCapacity.ExecutionStart == nil || !lockedAfterCapacity.ExecutionStart.Equal(*lockedStart) {
		t.Fatalf("locked baseline moved: before=%v after=%v", lockedStart, lockedAfterCapacity.ExecutionStart)
	}

	actualEnd := mustDate("2026-08-03")
	if err := database.Model(&acceptanceTaskRecord{}).Where("id = ?", "second-task").Updates(map[string]any{"actual_start": actualEnd, "actual_end": actualEnd}).Error; err != nil {
		t.Fatal(err)
	}
	_, err = projectService.ChangeStatus(context.Background(), "second", projectdomain.StatusClosed)
	closeImpact := requireOpenProjectImpact(t, err, "first")
	closed, err := projectService.ChangeStatus(acceptanceImpactContext(t, closeImpact.Token), "second", projectdomain.StatusClosed)
	if err != nil || closed == nil || closed.Status != projectdomain.StatusClosed {
		t.Fatalf("confirm close impact: %#v %v", closed, err)
	}
	firstAfterClose := loadAcceptanceTask(t, database, "first-task")
	assertDate(t, "open project uses capacity after Closed exclusion", firstAfterClose.ExecutionStart, "2026-08-03")
	completedClosed := loadAcceptanceTask(t, database, "second-task")
	assertDate(t, "completed closed date retained", completedClosed.ExecutionStart, "2026-08-03")

	beforeFailedReopen := closed.UpdatedAt
	failingScheduler := &acceptanceFailingScheduler{err: fmt.Errorf("schedule failed")}
	failingProjectService := projectapplication.NewServiceWithDependencies(projectrepo.New(database), failingScheduler, func() time.Time { return now.Add(2 * time.Hour) }, func() (string, error) { return "unused", nil })
	if _, err := failingProjectService.ChangeStatus(context.Background(), "second", projectdomain.StatusOpen); err == nil {
		t.Fatal("expected reopen scheduling failure")
	}
	var afterFailedReopen acceptanceProjectRecord
	if err := database.First(&afterFailedReopen, "id = ?", "second").Error; err != nil {
		t.Fatal(err)
	}
	if afterFailedReopen.Status != "closed" || !afterFailedReopen.UpdatedAt.Equal(beforeFailedReopen) {
		t.Fatalf("failed reopen persisted partial state=%#v", afterFailedReopen)
	}
}

func TestAutomaticSchedulingOrderingAndAssigneeAcceptance_AC12_AC15_AC16_AC17_AC18_AC19_AC20_AC21_AC25_AC26_AC32(t *testing.T) {
	t.Run("dependency readiness overrides priority and different assignee starts next date", func(t *testing.T) {
		database := acceptanceDatabase(t)
		now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
		anchor := mustDate("2026-08-03")
		roleID, blockerMember, blockedMember := "role", "blocker-member", "blocked-member"
		if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
			t.Fatal(err)
		}
		members := []acceptanceMemberRecord{
			{ID: blockerMember, Name: "Blocker", RoleID: roleID, DailyCapacity: "5", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
			{ID: blockedMember, Name: "Blocked", RoleID: roleID, DailyCapacity: "5", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
		}
		if err := database.Create(&members).Error; err != nil {
			t.Fatal(err)
		}
		projects := []acceptanceProjectRecord{
			{ID: "high", Name: "High", NameKey: "high", Status: "open", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
			{ID: "low", Name: "Low", NameKey: "low", Status: "open", Priority: 2, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
		}
		if err := database.Create(&projects).Error; err != nil {
			t.Fatal(err)
		}
		effort := 300
		tasks := []acceptanceTaskRecord{
			{ID: "blocker", ProjectID: "low", ParentKey: "", Name: "Blocker", NameKey: "blocker", Position: 1, RoleID: &roleID, AssigneeID: &blockerMember, EffortMinutes: &effort, CreatedAt: now, UpdatedAt: now},
			{ID: "blocked", ProjectID: "high", ParentKey: "", Name: "Blocked", NameKey: "blocked", Position: 1, RoleID: &roleID, AssigneeID: &blockedMember, EffortMinutes: &effort, CreatedAt: now, UpdatedAt: now},
		}
		if err := database.Create(&tasks).Error; err != nil {
			t.Fatal(err)
		}
		if err := database.Create(&acceptanceDependencyRecord{ID: "manual", BlockingTaskID: "blocker", BlockedTaskID: "blocked", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
		scheduler := schedulingapplication.NewService(NewWithDependencies(database, func() time.Time { return now }, func() (string, error) { return "automatic", nil }))
		if err := scheduler.RecalculateActiveProjects(context.Background()); err != nil {
			t.Fatal(err)
		}
		assertDate(t, "lower-priority blocker", loadAcceptanceTask(t, database, "blocker").ExecutionStart, "2026-08-03")
		assertDate(t, "higher-priority blocked task", loadAcceptanceTask(t, database, "blocked").ExecutionStart, "2026-08-04")
	})

	t.Run("fit uses gap while higher-priority readiness displaces whole lower task", func(t *testing.T) {
		database := acceptanceDatabase(t)
		now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
		anchor := mustDate("2026-08-03")
		roleID, sharedMember, blockerMember := "role", "shared", "blocker-member"
		if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
			t.Fatal(err)
		}
		members := []acceptanceMemberRecord{
			{ID: sharedMember, Name: "Shared", RoleID: roleID, DailyCapacity: "5", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
			{ID: blockerMember, Name: "Blocker", RoleID: roleID, DailyCapacity: "5", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
		}
		if err := database.Create(&members).Error; err != nil {
			t.Fatal(err)
		}
		projects := []acceptanceProjectRecord{
			{ID: "high", Name: "High", NameKey: "high", Status: "open", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
			{ID: "medium", Name: "Medium", NameKey: "medium", Status: "open", Priority: 2, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
			{ID: "low", Name: "Low", NameKey: "low", Status: "open", Priority: 3, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
		}
		if err := database.Create(&projects).Error; err != nil {
			t.Fatal(err)
		}
		candidateEffort, lowerEffort, blockerEffort := 480, 600, 300
		tasks := []acceptanceTaskRecord{
			{ID: "candidate", ProjectID: "high", ParentKey: "", Name: "Candidate", NameKey: "candidate", Position: 1, RoleID: &roleID, AssigneeID: &sharedMember, EffortMinutes: &candidateEffort, CreatedAt: now, UpdatedAt: now},
			{ID: "lower", ProjectID: "medium", ParentKey: "", Name: "Lower", NameKey: "lower", Position: 1, RoleID: &roleID, AssigneeID: &sharedMember, EffortMinutes: &lowerEffort, CreatedAt: now, UpdatedAt: now},
			{ID: "blocker", ProjectID: "low", ParentKey: "", Name: "Blocker", NameKey: "blocker", Position: 1, RoleID: &roleID, AssigneeID: &blockerMember, EffortMinutes: &blockerEffort, CreatedAt: now, UpdatedAt: now},
		}
		if err := database.Create(&tasks).Error; err != nil {
			t.Fatal(err)
		}
		if err := database.Create(&acceptanceDependencyRecord{ID: "manual", BlockingTaskID: "blocker", BlockedTaskID: "candidate", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
		scheduler := schedulingapplication.NewService(NewWithDependencies(database, func() time.Time { return now }, func() (string, error) { return "unused", nil }))
		if err := scheduler.RecalculateActiveProjects(context.Background()); err != nil {
			t.Fatal(err)
		}
		candidate := loadAcceptanceTask(t, database, "candidate")
		lower := loadAcceptanceTask(t, database, "lower")
		assertDate(t, "newly ready candidate", candidate.ExecutionStart, "2026-08-04")
		assertDate(t, "lower preserves pre-readiness allocation", lower.ExecutionStart, "2026-08-03")
		var automaticCount int64
		if err := database.Model(&acceptanceDependencyRecord{}).Where("blocking_task_id = ? AND blocked_task_id = ?", "candidate", "lower").Count(&automaticCount).Error; err != nil {
			t.Fatal(err)
		}
		if automaticCount != 0 {
			t.Fatalf("priority displacement invented serial ownership count=%d", automaticCount)
		}
	})

	t.Run("assignee change and clear reconcile old and new scheduling scopes", func(t *testing.T) {
		database := acceptanceDatabase(t)
		now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
		anchor := mustDate("2026-08-03")
		roleID, memberA, memberB := "role", "member-a", "member-b"
		if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
			t.Fatal(err)
		}
		members := []acceptanceMemberRecord{
			{ID: memberA, Name: "A", RoleID: roleID, DailyCapacity: "5", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
			{ID: memberB, Name: "B", RoleID: roleID, DailyCapacity: "5", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now},
		}
		if err := database.Create(&members).Error; err != nil {
			t.Fatal(err)
		}
		project := acceptanceProjectRecord{ID: "project", Name: "Project", NameKey: "project", Status: "open", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now}
		if err := database.Create(&project).Error; err != nil {
			t.Fatal(err)
		}
		effort := 300
		tasks := []acceptanceTaskRecord{
			{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1, RoleID: &roleID, AssigneeID: &memberA, EffortMinutes: &effort, CreatedAt: now, UpdatedAt: now},
			{ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2, RoleID: &roleID, AssigneeID: &memberA, EffortMinutes: &effort, CreatedAt: now, UpdatedAt: now},
		}
		if err := database.Create(&tasks).Error; err != nil {
			t.Fatal(err)
		}
		automaticID := 0
		scheduler := schedulingapplication.NewService(NewWithDependencies(database, func() time.Time { return now }, func() (string, error) {
			automaticID++
			return fmt.Sprintf("automatic-%d", automaticID), nil
		}))
		if err := scheduler.RecalculateActiveProjects(context.Background()); err != nil {
			t.Fatal(err)
		}
		assertDate(t, "WBS first task", loadAcceptanceTask(t, database, "a").ExecutionStart, "2026-08-03")
		assertDate(t, "WBS second task", loadAcceptanceTask(t, database, "b").ExecutionStart, "2026-08-04")
		var initialCount int64
		if err := database.Model(&acceptanceDependencyRecord{}).Where("blocking_task_id = ? AND blocked_task_id = ?", "a", "b").Count(&initialCount).Error; err != nil {
			t.Fatal(err)
		}
		if initialCount != 0 {
			t.Fatalf("resource-derived dependency count=%d, want 0", initialCount)
		}

		service := wbsapplication.NewServiceWithDependencies(wbsgormrepo.New(database), scheduler, func() time.Time { return now.Add(time.Hour) }, func() (string, error) { return "unused", nil })
		if _, err := service.UpdateExecutable(context.Background(), "project", "b", wbsapplication.WriteExecutableInput{RoleID: &roleID, AssigneeID: &memberB, EffortMinutes: &effort, LagDays: 0}); err != nil {
			t.Fatal(err)
		}
		var endpointCount int64
		if err := database.Model(&acceptanceDependencyRecord{}).Where("blocking_task_id = ? AND blocked_task_id = ?", "a", "b").Count(&endpointCount).Error; err != nil {
			t.Fatal(err)
		}
		if endpointCount != 0 {
			t.Fatalf("stale automatic relation count=%d", endpointCount)
		}
		assertDate(t, "reassigned task uses new member scope", loadAcceptanceTask(t, database, "b").ExecutionStart, "2026-08-03")

		if _, err := service.UpdateExecutable(context.Background(), "project", "b", wbsapplication.WriteExecutableInput{RoleID: &roleID, AssigneeID: nil, EffortMinutes: &effort, LagDays: 0}); err != nil {
			t.Fatal(err)
		}
		cleared := loadAcceptanceTask(t, database, "b")
		if cleared.ExecutionStart != nil || cleared.ExecutionUnscheduledReason == nil || *cleared.ExecutionUnscheduledReason != reasonMissingAssignee {
			t.Fatalf("cleared assignee projection=%#v", cleared)
		}
	})

	t.Run("duplicate priority is rejected without an invented tie breaker", func(t *testing.T) {
		database := acceptanceDatabase(t)
		now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
		anchor := mustDate("2026-08-03")
		roleID, memberID := "role", "member"
		if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
			t.Fatal(err)
		}
		if err := database.Create(&acceptanceMemberRecord{ID: memberID, Name: "Member", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
		projects := []acceptanceProjectRecord{
			{ID: "a", Name: "A", NameKey: "a", Status: "open", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
			{ID: "b", Name: "B", NameKey: "b", Status: "open", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
		}
		if err := database.Create(&projects).Error; err != nil {
			t.Fatal(err)
		}
		effort := 60
		tasks := []acceptanceTaskRecord{
			{ID: "a-task", ProjectID: "a", ParentKey: "", Name: "A", NameKey: "a", Position: 1, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &effort, CreatedAt: now, UpdatedAt: now},
			{ID: "b-task", ProjectID: "b", ParentKey: "", Name: "B", NameKey: "b", Position: 1, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &effort, CreatedAt: now, UpdatedAt: now},
		}
		if err := database.Create(&tasks).Error; err != nil {
			t.Fatal(err)
		}
		scheduler := schedulingapplication.NewService(New(database))
		err := scheduler.RecalculateActiveProjects(context.Background())
		if !errors.Is(err, schedulingdomain.ErrDataIntegrity) {
			t.Fatalf("duplicate priority error=%v", err)
		}
		if loadAcceptanceTask(t, database, "a-task").ExecutionStart != nil || loadAcceptanceTask(t, database, "b-task").ExecutionStart != nil {
			t.Fatal("invalid ordering persisted generated dates")
		}
	})
}

func TestAutomaticSchedulingConcurrentApplicationAcceptance_AC36(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	anchor := mustDate("2026-08-03")
	roleID, memberID := "role", "member"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceMemberRecord{ID: memberID, Name: "Member", RoleID: roleID, DailyCapacity: "8", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now}).Error; err != nil {
		t.Fatal(err)
	}
	project := acceptanceProjectRecord{ID: "project", Name: "Project", NameKey: "project", Status: "open", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now}
	if err := database.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	effort := 240
	tasks := []acceptanceTaskRecord{
		{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &effort, CreatedAt: now, UpdatedAt: now},
		{ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &effort, CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	automaticID := 0
	repository := NewWithDependencies(database, func() time.Time { return now }, func() (string, error) {
		automaticID++
		return fmt.Sprintf("automatic-%d", automaticID), nil
	})
	service := schedulingapplication.NewService(repository)
	start := make(chan struct{})
	errorsByRun := make([]error, 2)
	var wait sync.WaitGroup
	for index := range errorsByRun {
		wait.Add(1)
		go func(run int) {
			defer wait.Done()
			<-start
			errorsByRun[run] = service.RecalculateActiveProjects(context.Background())
		}(index)
	}
	close(start)
	wait.Wait()
	for index, err := range errorsByRun {
		if err != nil {
			t.Fatalf("concurrent application run %d error=%v", index, err)
		}
	}
	var storedProject acceptanceProjectRecord
	if err := database.First(&storedProject, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	if storedProject.ScheduleVersion != 1 {
		t.Fatalf("schedule version=%d, want one confirmed dirty projection and one serialized no-op", storedProject.ScheduleVersion)
	}
	var rows []acceptanceAllocationRecord
	if err := database.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		key := row.TaskID + "|" + row.Timeline + "|" + schedulingdomain.DateKey(row.AllocationDate)
		if seen[key] {
			t.Fatalf("duplicate allocation projection=%s", key)
		}
		seen[key] = true
	}
}

type acceptanceFailingScheduler struct{ err error }

func (scheduler *acceptanceFailingScheduler) RecalculateActiveProjects(context.Context) error {
	return scheduler.err
}
func (scheduler *acceptanceFailingScheduler) RecalculateProjectSchedule(context.Context, string) error {
	return scheduler.err
}
func (scheduler *acceptanceFailingScheduler) MarkProjectUnscheduled(context.Context, string, string) error {
	return scheduler.err
}

func (acceptanceProjectRecord) TableName() string { return "projects" }

type acceptanceTaskRecord struct {
	ID                           string `gorm:"primaryKey"`
	ProjectID                    string
	ParentID                     *string
	ParentKey                    string
	Name                         string
	NameKey                      string
	Position                     int
	RoleID                       *string
	AssigneeID                   *string
	EffortMinutes                *int
	LagDays                      int
	CapacityAllocationPercentage int `gorm:"default:100"`
	ExecutionStart               *time.Time
	ExecutionEnd                 *time.Time
	CommitmentStart              *time.Time
	CommitmentEnd                *time.Time
	ActualStart                  *time.Time
	ActualEnd                    *time.Time
	ExecutionUnscheduledReason   *string
	CommitmentUnscheduledReason  *string
	CreatedAt                    time.Time
	UpdatedAt                    time.Time
}

func (acceptanceTaskRecord) TableName() string { return "wbs_nodes" }

type acceptanceRoleRecord struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

func (acceptanceRoleRecord) TableName() string { return "roles" }

type acceptanceMemberRecord struct {
	ID               string `gorm:"primaryKey"`
	Name             string
	RoleID           string
	DailyCapacity    string
	BufferPercentage string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

func (acceptanceMemberRecord) TableName() string { return "team_members" }

type acceptanceOverrideRecord struct {
	ID           string `gorm:"primaryKey"`
	TeamMemberID string
	StartDate    time.Time
	EndDate      time.Time
	Capacity     string
	DeletedAt    *time.Time
}

func (acceptanceOverrideRecord) TableName() string { return "capacity_overrides" }

type acceptanceHolidayDateRecord struct {
	PublicHolidayID string
	Date            time.Time
}

func (acceptanceHolidayDateRecord) TableName() string { return "public_holiday_dates" }

type acceptanceDependencyRecord struct {
	ID             string `gorm:"primaryKey"`
	BlockingTaskID string
	BlockedTaskID  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (acceptanceDependencyRecord) TableName() string { return "task_dependencies" }

type acceptanceAllocationRecord struct {
	TaskID                   string
	AssigneeID               string
	Timeline                 string
	AllocationDate           time.Time
	AllocatedMinutes         string
	RemainingCapacityMinutes string
	Sequence                 int
}

func (acceptanceAllocationRecord) TableName() string { return "task_schedule_allocations" }

func acceptanceDatabase(t *testing.T) *gorm.DB {
	t.Helper()
	database, err := gorm.Open(
		sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_busy_timeout=5000"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&acceptanceRoleRecord{},
		&acceptanceProjectRecord{},
		&acceptanceMemberRecord{},
		&acceptanceTaskRecord{},
		&acceptanceOverrideRecord{},
		&acceptanceHolidayDateRecord{},
		&acceptanceDependencyRecord{},
		&acceptanceAllocationRecord{},
	); err != nil {
		t.Fatal(err)
	}
	return database
}

func TestCreateWBSAcceptanceSkipsSchedulerForNameOnlyTaskWhenProjectHasCompletedHistoricalOverCapacityTask_US6_AC29_US4_AC23(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	anchor := mustDate("2026-08-03")
	roleID, memberID := "role", "member"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceMemberRecord{
		ID: memberID, Name: "Rani", RoleID: roleID,
		DailyCapacity: "5", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceProjectRecord{
		ID: "project", Name: "Project", NameKey: "project", Status: "open", Priority: 1,
		AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor,
		ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	historicalEffort := 600
	if err := database.Create(&acceptanceTaskRecord{
		ID: "completed", ProjectID: "project", ParentKey: "", Name: "Completed", NameKey: "completed",
		Position: 1, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &historicalEffort,
		ExecutionStart: &anchor, ExecutionEnd: &anchor, CommitmentStart: &anchor, CommitmentEnd: &anchor,
		ActualStart: &anchor, ActualEnd: &anchor, CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	scheduler := schedulingapplication.NewService(NewWithDependencies(
		database,
		func() time.Time { return now },
		func() (string, error) { return "automatic", nil },
	))
	wbsService := wbsapplication.NewServiceWithDependencies(
		wbsgormrepo.New(database),
		scheduler,
		func() time.Time { return now.Add(time.Hour) },
		func() (string, error) { return "new-task", nil },
	)

	created, err := wbsService.Create(context.Background(), "project", nil, "New Task", false)
	if err != nil {
		t.Fatal(err)
	}
	if created == nil || created.ID != "new-task" {
		t.Fatalf("created WBS = %#v, want new-task", created)
	}
	storedCompleted := loadAcceptanceTask(t, database, "completed")
	assertDate(t, "completed execution start", storedCompleted.ExecutionStart, "2026-08-03")
	assertDate(t, "completed execution end", storedCompleted.ExecutionEnd, "2026-08-03")
	assertDate(t, "completed Actual End", storedCompleted.ActualEnd, "2026-08-03")
	storedNewTask := loadAcceptanceTask(t, database, "new-task")
	if storedNewTask.AssigneeID != nil ||
		storedNewTask.EffortMinutes != nil ||
		storedNewTask.ExecutionStart != nil ||
		storedNewTask.ExecutionEnd != nil ||
		storedNewTask.CommitmentStart != nil ||
		storedNewTask.CommitmentEnd != nil ||
		storedNewTask.ExecutionUnscheduledReason != nil ||
		storedNewTask.CommitmentUnscheduledReason != nil {
		t.Fatalf("new Task projection = %#v, want empty unscheduled state without scheduler projection", storedNewTask)
	}
}

func TestAutomaticSchedulingWorkflowGeneratesDatesOwnershipAndLag_AC2_AC9_AC11_AC16_AC22_AC23_AC24_AC27_AC28_AC35_AC37(t *testing.T) {
	database := acceptanceDatabase(t)
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	anchor := mustDate("2026-08-03")
	roleID, memberID := "role", "member"
	if err := database.Create(&acceptanceRoleRecord{ID: roleID, Name: "Engineer"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&acceptanceMemberRecord{
		ID: memberID, Name: "Rani", RoleID: roleID,
		DailyCapacity: "5", BufferPercentage: "0", CreatedAt: now, UpdatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	projects := []acceptanceProjectRecord{
		{ID: "high", Name: "High", NameKey: "high", Status: "open", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
		{ID: "low", Name: "Low", NameKey: "low", Status: "open", Priority: 2, AutomaticScheduling: true, AutoCalculateDate: true, SchedulingStartDate: &anchor, ProjectBuffer: 0, CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&projects).Error; err != nil {
		t.Fatal(err)
	}
	highEffort, lowEffort := 240, 300
	tasks := []acceptanceTaskRecord{
		{ID: "high-task", ProjectID: "high", ParentKey: "", Name: "High Task", NameKey: "high task", Position: 1, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &highEffort, CreatedAt: now, UpdatedAt: now},
		{ID: "low-task", ProjectID: "low", ParentKey: "", Name: "Low Task", NameKey: "low task", Position: 1, RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &lowEffort, CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}

	automaticID := 0
	schedulingRepository := NewWithDependencies(database, func() time.Time { return now }, func() (string, error) {
		automaticID++
		return fmt.Sprintf("automatic-%d", automaticID), nil
	})
	scheduler := schedulingapplication.NewService(schedulingRepository)
	if err := scheduler.RecalculateActiveProjects(context.Background()); err != nil {
		t.Fatal(err)
	}

	beforeLag := loadAcceptanceTask(t, database, "low-task")
	assertDate(t, "initial low task start", beforeLag.ExecutionStart, "2026-08-03")
	assertDate(t, "initial low task end", beforeLag.ExecutionEnd, "2026-08-04")
	var initialRelationCount int64
	if err := database.Model(&acceptanceDependencyRecord{}).Where("blocking_task_id = ? AND blocked_task_id = ?", "high-task", "low-task").Count(&initialRelationCount).Error; err != nil {
		t.Fatal(err)
	}
	if initialRelationCount != 0 {
		t.Fatalf("parallel initial schedule created serial relation count=%d", initialRelationCount)
	}

	dependencyService := dependencyapplication.NewServiceWithDependencies(
		dependencyrepo.New(database),
		scheduler,
		func() time.Time { return now.Add(time.Hour) },
		func() (string, error) { return "manual-request", nil },
	)
	shared, err := dependencyService.Create(context.Background(), "high-task", "low-task")
	if err != nil {
		t.Fatal(err)
	}
	if shared == nil {
		t.Fatalf("manual create result=%#v, want manual-only relation", shared)
	}
	var endpointCount int64
	if err := database.Model(&acceptanceDependencyRecord{}).
		Where("blocking_task_id = ? AND blocked_task_id = ?", "high-task", "low-task").
		Count(&endpointCount).Error; err != nil {
		t.Fatal(err)
	}
	if endpointCount != 1 {
		t.Fatalf("visible endpoint count=%d, want 1", endpointCount)
	}

	wbsService := wbsapplication.NewServiceWithDependencies(
		wbsgormrepo.New(database),
		scheduler,
		func() time.Time { return now.Add(2 * time.Hour) },
		func() (string, error) { return "unused", nil },
	)
	if _, err := wbsService.UpdateExecutable(context.Background(), "low", "low-task", wbsapplication.WriteExecutableInput{
		RoleID: &roleID, AssigneeID: &memberID, EffortMinutes: &lowEffort, LagDays: 1,
	}); err != nil {
		t.Fatal(err)
	}

	afterLag := loadAcceptanceTask(t, database, "low-task")
	if afterLag.LagDays != 1 {
		t.Fatalf("persisted Lag=%d, want 1", afterLag.LagDays)
	}
	assertDate(t, "lagged low task start", afterLag.ExecutionStart, "2026-08-04")
	assertDate(t, "lagged low task end", afterLag.ExecutionEnd, "2026-08-04")
	afterRelation := loadAcceptanceDependency(t, database, "high-task", "low-task")
	if afterRelation.ID != shared.ID {
		t.Fatalf("dependency after Lag=%#v, want same manual relation", afterRelation)
	}
	endpointCount = 0
	if err := database.Model(&acceptanceDependencyRecord{}).
		Where("blocking_task_id = ? AND blocked_task_id = ?", "high-task", "low-task").
		Count(&endpointCount).Error; err != nil {
		t.Fatal(err)
	}
	if endpointCount != 1 {
		t.Fatalf("endpoint count after Lag=%d, want 1", endpointCount)
	}
	allocations := loadAcceptanceAllocations(t, database, "low-task", string(schedulingdomain.Execution))
	if len(allocations) != 1 || schedulingdomain.DateKey(allocations[0].AllocationDate) != "2026-08-04" || allocations[0].AllocatedMinutes != "300.000000" {
		t.Fatalf("lagged execution allocations=%#v", allocations)
	}
	var lowProject acceptanceProjectRecord
	if err := database.First(&lowProject, "id = ?", "low").Error; err != nil {
		t.Fatal(err)
	}
	if lowProject.ScheduleVersion != 2 {
		t.Fatalf(
			"schedule version=%d, want 2 confirmed projection changes; manual ownership-only mutation is version-neutral",
			lowProject.ScheduleVersion,
		)
	}
}

func requireOpenProjectImpact(t *testing.T, err error, expectedProjectID string) schedulingimpact.Error {
	t.Helper()
	var impact schedulingimpact.Error
	if !errors.As(err, &impact) {
		t.Fatalf("expected scheduling impact confirmation, got %v", err)
	}
	if impact.Kind != schedulingimpact.ConfirmationRequired || impact.Token == "" {
		t.Fatalf("scheduling impact = %#v", impact)
	}
	if len(impact.LockedProjects) != 0 || len(impact.OpenProjects) != 1 || impact.OpenProjects[0].ID != expectedProjectID {
		t.Fatalf("impacted projects = locked %#v, open %#v; want open %q", impact.LockedProjects, impact.OpenProjects, expectedProjectID)
	}
	return impact
}

func acceptanceImpactContext(t *testing.T, token string) context.Context {
	t.Helper()
	var captured context.Context
	handler := schedulingimpact.CaptureToken(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		captured = request.Context()
	}))
	request := httptest.NewRequest(http.MethodPost, "/", nil)
	request.Header.Set(schedulingimpact.ConfirmationTokenHeader, token)
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if captured == nil {
		t.Fatal("scheduling impact request context was not captured")
	}
	return captured
}

func loadAcceptanceTask(t *testing.T, database *gorm.DB, id string) acceptanceTaskRecord {
	t.Helper()
	var value acceptanceTaskRecord
	if err := database.First(&value, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	return value
}

func loadAcceptanceDependency(t *testing.T, database *gorm.DB, blockingTaskID, blockedTaskID string) acceptanceDependencyRecord {
	t.Helper()
	var value acceptanceDependencyRecord
	if err := database.Where("blocking_task_id = ? AND blocked_task_id = ?", blockingTaskID, blockedTaskID).First(&value).Error; err != nil {
		t.Fatal(err)
	}
	return value
}

func loadAcceptanceAllocations(t *testing.T, database *gorm.DB, taskID, timeline string) []acceptanceAllocationRecord {
	t.Helper()
	var values []acceptanceAllocationRecord
	if err := database.Where("task_id = ? AND timeline = ?", taskID, timeline).Order("allocation_date ASC").Find(&values).Error; err != nil {
		t.Fatal(err)
	}
	return values
}

func TestImpactProjectsPreservesRequestedLifecycleClassification_US62_AC15_AC20_AC21(t *testing.T) {
	state := &portfolioState{projects: map[string]projectModel{
		"open":   {ID: "open", Name: "Open", Status: "open", ScheduleVersion: 3},
		"locked": {ID: "locked", Name: "Locked", Status: "locked", ScheduleVersion: 7},
	}}

	locked := impactProjects(state, []string{"locked", "open"}, "locked")
	if len(locked) != 1 || locked[0].ID != "locked" || locked[0].Version != 7 {
		t.Fatalf("locked impact projects = %#v", locked)
	}
	open := impactProjects(state, []string{"locked", "open"}, "open")
	if len(open) != 1 || open[0].ID != "open" || open[0].Version != 3 {
		t.Fatalf("open impact projects = %#v", open)
	}
}
