package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type scheduleProjectRecord struct {
	ID                  string `gorm:"primaryKey"`
	Name                string
	Status              string
	Priority            int
	AutomaticScheduling bool
	SchedulingStartDate *time.Time
	ProjectBuffer       int
	StartDate           *time.Time
	EndDate             *time.Time
	ScheduleVersion     int64
	UpdatedAt           time.Time
}

func (scheduleProjectRecord) TableName() string { return "projects" }

func schedulerRepository(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	database, err := gorm.Open(
		sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_busy_timeout=5000"),
		&gorm.Config{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(
		&scheduleProjectRecord{},
		&taskModel{},
		&memberModel{},
		&capacityOverrideModel{},
		&holidayDateModel{},
		&dependencyModel{},
		&allocationModel{},
	); err != nil {
		t.Fatal(err)
	}
	sequence := 0
	repository := NewWithDependencies(
		database,
		func() time.Time { return mustDate("2026-08-31") },
		func() (string, error) {
			sequence++
			return fmt.Sprintf("automatic-%d", sequence), nil
		},
	)
	return repository, database
}

func TestRecalculatePortfolioResolvesCapacityPrecedenceAndIndependentTimelines_AC5_AC6_AC7_AC8_AC9_AC10_AC33(t *testing.T) {
	repository, database := schedulerRepository(t)
	seedProject(t, database, projectModel{
		ID: "project", Status: "open", Priority: 1, AutomaticScheduling: true,
		SchedulingStartDate: datePointer(mustDate("2026-08-07")), ProjectBuffer: 20,
	})
	seedMember(t, database, "member", "8", "30")
	seedTask(t, database, taskModel{
		ID: "task", ProjectID: "project", ParentKey: "", Position: 1, Name: "Task",
		AssigneeID: textPointer("member"), EffortMinutes: intPointer(840),
	})
	if err := database.Create(&capacityOverrideModel{
		ID: "override-friday", TeamMemberID: "member",
		StartDate: mustDate("2026-08-07"), EndDate: mustDate("2026-08-07"), Capacity: "4",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&capacityOverrideModel{
		ID: "override-holiday", TeamMemberID: "member",
		StartDate: mustDate("2026-08-10"), EndDate: mustDate("2026-08-10"), Capacity: "12",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&holidayDateModel{PublicHolidayID: "holiday", Date: mustDate("2026-08-10")}).Error; err != nil {
		t.Fatal(err)
	}

	if err := repository.RecalculatePortfolio(context.Background(), []string{"project"}); err != nil {
		t.Fatal(err)
	}

	task := loadScheduledTask(t, database, "task")
	assertDate(t, "execution start", task.ExecutionStart, "2026-08-07")
	assertDate(t, "execution end", task.ExecutionEnd, "2026-08-12")
	assertDate(t, "commitment start", task.CommitmentStart, "2026-08-07")
	assertDate(t, "commitment end", task.CommitmentEnd, "2026-08-13")
	if !task.CommitmentEnd.After(*task.ExecutionEnd) {
		t.Fatalf("Commitment end %v must be independently later than Execution end %v", task.CommitmentEnd, task.ExecutionEnd)
	}

	rows := loadAllocations(t, database, "task", "execution")
	if len(rows) != 3 {
		t.Fatalf("Execution allocation rows = %d, want 3", len(rows))
	}
	if got := rows[0].AllocatedMinutes; got != "180.000000" {
		t.Fatalf("Friday override allocation = %s, want rounded 180.000000", got)
	}
	for _, row := range rows {
		if key := schedulingdomain.DateKey(row.AllocationDate); key == "2026-08-08" || key == "2026-08-09" || key == "2026-08-10" {
			t.Fatalf("zero-capacity date %s received allocation", key)
		}
	}
}

func TestDependencyAndLagReadinessRules_AC11_AC12_AC13_AC14_AC15(t *testing.T) {
	t.Run("same assignee uses predecessor end-day remaining capacity", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
		seedMember(t, database, "member", "5", "0")
		seedTask(t, database, schedulableTask("a", "project", 1, "member", 480, 0))
		seedTask(t, database, schedulableTask("b", "project", 2, "member", 480, 0))
		seedDependency(t, database, dependencyModel{ID: "a-b", BlockingTaskID: "a", BlockedTaskID: "b", ManualOwned: true})

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		a := loadScheduledTask(t, database, "a")
		b := loadScheduledTask(t, database, "b")
		assertDate(t, "A end", a.ExecutionEnd, "2026-08-04")
		assertDate(t, "B start", b.ExecutionStart, "2026-08-04")
		assertDate(t, "B end", b.ExecutionEnd, "2026-08-06")
	})

	t.Run("different assignee starts on next positive-capacity date", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
		seedMember(t, database, "member-a", "5", "0")
		seedMember(t, database, "member-b", "5", "0")
		seedTask(t, database, schedulableTask("a", "project", 1, "member-a", 480, 0))
		seedTask(t, database, schedulableTask("b", "project", 2, "member-b", 300, 0))
		seedDependency(t, database, dependencyModel{ID: "a-b", BlockingTaskID: "a", BlockedTaskID: "b", ManualOwned: true})

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		assertDate(t, "B start", loadScheduledTask(t, database, "b").ExecutionStart, "2026-08-05")
	})

	t.Run("lag is a calendar offset and skips weekend capacity", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-06", 0))
		seedMember(t, database, "member", "5", "0")
		seedTask(t, database, schedulableTask("a", "project", 1, "member", 480, 0))
		seedTask(t, database, schedulableTask("b", "project", 2, "member", 300, 1))
		seedDependency(t, database, dependencyModel{ID: "a-b", BlockingTaskID: "a", BlockedTaskID: "b", ManualOwned: true})

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		assertDate(t, "A end", loadScheduledTask(t, database, "a").ExecutionEnd, "2026-08-07")
		assertDate(t, "B start", loadScheduledTask(t, database, "b").ExecutionStart, "2026-08-10")
	})

	t.Run("lag without dependency offsets project anchor", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-07", 0))
		seedMember(t, database, "member", "8", "0")
		seedTask(t, database, schedulableTask("task", "project", 1, "member", 60, 1))

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		assertDate(t, "Task start", loadScheduledTask(t, database, "task").ExecutionStart, "2026-08-10")
	})
}

func TestNonPreemptiveFitAndPriorityDisplacement_AC16_AC18_AC19_AC20_AC21_AC22(t *testing.T) {
	t.Run("lower-priority task that fits uses the gap and identifies the preceding blocker", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedGapFixture(t, database, 240)
		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		candidate := loadScheduledTask(t, database, "candidate")
		assertDate(t, "candidate start", candidate.ExecutionStart, "2026-08-04")
		assertDate(t, "candidate end", candidate.ExecutionEnd, "2026-08-04")
		assertAutomaticBlocker(t, database, "fixed-a", "candidate")
	})

	t.Run("lower-priority task that does not fit moves after the next reservation", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedGapFixture(t, database, 480)
		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		candidate := loadScheduledTask(t, database, "candidate")
		assertDate(t, "candidate start", candidate.ExecutionStart, "2026-08-06")
		assertAutomaticBlocker(t, database, "fixed-b", "candidate")
	})

	t.Run("newly-ready higher-priority task displaces the whole lower-priority task", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("high", 1, "2026-08-03", 0))
		seedProject(t, database, automaticProject("medium", 2, "2026-08-03", 0))
		seedProject(t, database, automaticProject("low", 3, "2026-08-03", 0))
		seedMember(t, database, "shared", "5", "0")
		seedMember(t, database, "blocker-member", "5", "0")
		seedTask(t, database, schedulableTask("candidate", "high", 1, "shared", 480, 0))
		seedTask(t, database, schedulableTask("lower", "medium", 1, "shared", 600, 0))
		seedTask(t, database, schedulableTask("blocker", "low", 1, "blocker-member", 300, 0))
		seedDependency(t, database, dependencyModel{ID: "blocker-candidate", BlockingTaskID: "blocker", BlockedTaskID: "candidate", ManualOwned: true})

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		candidate := loadScheduledTask(t, database, "candidate")
		lower := loadScheduledTask(t, database, "lower")
		assertDate(t, "candidate start", candidate.ExecutionStart, "2026-08-04")
		if lower.ExecutionStart == nil || candidate.ExecutionEnd == nil || lower.ExecutionStart.Before(*candidate.ExecutionEnd) {
			t.Fatalf("lower-priority task started at %v before displaced candidate ended at %v", lower.ExecutionStart, candidate.ExecutionEnd)
		}
		for _, row := range loadAllocations(t, database, "lower", "execution") {
			if row.AllocationDate.Before(*candidate.ExecutionEnd) {
				t.Fatalf("lower-priority task retained preempted allocation on %s", schedulingdomain.DateKey(row.AllocationDate))
			}
		}
		assertAutomaticBlocker(t, database, "candidate", "lower")
	})
}

func TestAutomaticOwnershipReconciliationPreservesManualGraph_AC22_AC23_AC24_AC25_AC26(t *testing.T) {
	t.Run("manual endpoint becomes shared and appears once", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
		seedMember(t, database, "member", "5", "0")
		seedTask(t, database, schedulableTask("a", "project", 1, "member", 300, 0))
		seedTask(t, database, schedulableTask("b", "project", 2, "member", 300, 0))
		seedDependency(t, database, dependencyModel{ID: "a-b", BlockingTaskID: "a", BlockedTaskID: "b", ManualOwned: true})

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		var rows []dependencyModel
		if err := database.Where("blocking_task_id = ? AND blocked_task_id = ?", "a", "b").Find(&rows).Error; err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || !rows[0].ManualOwned || !rows[0].AutomaticOwned {
			t.Fatalf("endpoint ownership = %#v, want one shared relation", rows)
		}
	})

	t.Run("stale automatic ownership is removed without removing manual ownership", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
		seedMember(t, database, "member-a", "8", "0")
		seedMember(t, database, "member-b", "8", "0")
		seedTask(t, database, schedulableTask("a", "project", 1, "member-a", 60, 0))
		seedTask(t, database, schedulableTask("b", "project", 2, "member-b", 60, 0))
		seedDependency(t, database, dependencyModel{ID: "a-b", BlockingTaskID: "a", BlockedTaskID: "b", ManualOwned: true, AutomaticOwned: true})

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		var relation dependencyModel
		if err := database.First(&relation, "id = ?", "a-b").Error; err != nil {
			t.Fatal(err)
		}
		if !relation.ManualOwned || relation.AutomaticOwned {
			t.Fatalf("ownership = manual:%v automatic:%v, want manual-only", relation.ManualOwned, relation.AutomaticOwned)
		}
	})

	t.Run("clearing assignee removes automatic-only ownership and leaves safe unscheduled state", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
		seedMember(t, database, "member", "8", "0")
		seedTask(t, database, schedulableTask("a", "project", 1, "member", 60, 0))
		missingAssignee := schedulableTask("b", "project", 2, "member", 60, 0)
		missingAssignee.AssigneeID = nil
		seedTask(t, database, missingAssignee)
		seedDependency(t, database, dependencyModel{ID: "a-b", BlockingTaskID: "a", BlockedTaskID: "b", AutomaticOwned: true})

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		var count int64
		if err := database.Model(&dependencyModel{}).Where("id = ?", "a-b").Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("automatic-only relation count = %d, want 0", count)
		}
		b := loadScheduledTask(t, database, "b")
		if b.ExecutionStart != nil || b.ExecutionUnscheduledReason == nil || *b.ExecutionUnscheduledReason != reasonMissingAssignee {
			t.Fatalf("assignee-clear projection = %#v", b)
		}
	})
}

func TestEligibilityFixedReservationsAndClosedExclusion_AC4_AC30_AC31_AC32_AC34(t *testing.T) {
	t.Run("missing anchor is persisted as explicit unscheduled state", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		project := automaticProject("project", 1, "2026-08-03", 0)
		project.SchedulingStartDate = nil
		seedProject(t, database, project)
		seedMember(t, database, "member", "8", "0")
		seedTask(t, database, schedulableTask("task", "project", 1, "member", 60, 0))

		if err := repository.RecalculatePortfolio(context.Background(), []string{"project"}); err != nil {
			t.Fatal(err)
		}
		task := loadScheduledTask(t, database, "task")
		if task.ExecutionStart != nil || task.ExecutionUnscheduledReason == nil || *task.ExecutionUnscheduledReason != reasonMissingAnchor {
			t.Fatalf("missing-anchor projection = %#v", task)
		}
	})

	t.Run("zero capacity produces no fabricated dates for either timeline", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-03", 100))
		seedMember(t, database, "member", "0", "0")
		seedTask(t, database, schedulableTask("task", "project", 1, "member", 60, 0))

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		task := loadScheduledTask(t, database, "task")
		if task.ExecutionStart != nil || task.CommitmentStart != nil {
			t.Fatalf("zero-capacity dates = execution:%v commitment:%v", task.ExecutionStart, task.CommitmentStart)
		}
		if task.ExecutionUnscheduledReason == nil || *task.ExecutionUnscheduledReason != reasonZeroCapacity {
			t.Fatalf("execution reason = %v", task.ExecutionUnscheduledReason)
		}
		if task.CommitmentUnscheduledReason == nil || *task.CommitmentUnscheduledReason != reasonZeroCapacity {
			t.Fatalf("commitment reason = %v", task.CommitmentUnscheduledReason)
		}
	})

	t.Run("completed blocker uses Actual End and keeps completed dates", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
		seedMember(t, database, "member-a", "5", "0")
		seedMember(t, database, "member-b", "5", "0")
		completed := schedulableTask("completed", "project", 1, "member-a", 300, 0)
		completed.ExecutionStart = datePointer(mustDate("2026-08-07"))
		completed.ExecutionEnd = datePointer(mustDate("2026-08-07"))
		completed.CommitmentStart = datePointer(mustDate("2026-08-07"))
		completed.CommitmentEnd = datePointer(mustDate("2026-08-07"))
		completed.ActualStart = datePointer(mustDate("2026-08-07"))
		completed.ActualEnd = datePointer(mustDate("2026-08-07"))
		seedTask(t, database, completed)
		seedTask(t, database, schedulableTask("successor", "project", 2, "member-b", 300, 0))
		seedDependency(t, database, dependencyModel{ID: "completed-successor", BlockingTaskID: "completed", BlockedTaskID: "successor", ManualOwned: true})

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		storedCompleted := loadScheduledTask(t, database, "completed")
		assertDate(t, "completed execution end", storedCompleted.ExecutionEnd, "2026-08-07")
		assertDate(t, "successor start", loadScheduledTask(t, database, "successor").ExecutionStart, "2026-08-10")
	})

	t.Run("completed historical effort may exceed current capacity without blocking recalculation", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
		seedMember(t, database, "member", "5", "0")

		completed := schedulableTask("completed", "project", 1, "member", 600, 0)
		completed.ExecutionStart = datePointer(mustDate("2026-08-03"))
		completed.ExecutionEnd = datePointer(mustDate("2026-08-03"))
		completed.CommitmentStart = datePointer(mustDate("2026-08-03"))
		completed.CommitmentEnd = datePointer(mustDate("2026-08-03"))
		completed.ActualStart = datePointer(mustDate("2026-08-03"))
		completed.ActualEnd = datePointer(mustDate("2026-08-03"))
		seedTask(t, database, completed)
		seedTask(t, database, schedulableTask("new-task", "project", 2, "member", 300, 0))

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}

		storedCompleted := loadScheduledTask(t, database, "completed")
		assertDate(t, "completed execution start", storedCompleted.ExecutionStart, "2026-08-03")
		assertDate(t, "completed execution end", storedCompleted.ExecutionEnd, "2026-08-03")
		assertDate(t, "new task starts after completed Actual End", loadScheduledTask(t, database, "new-task").ExecutionStart, "2026-08-04")
	})

	t.Run("locked timeline still rejects effort that exceeds fixed capacity", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		project := automaticProject("locked", 1, "2026-08-03", 0)
		project.Status = "locked"
		seedProject(t, database, project)
		seedMember(t, database, "member", "5", "0")

		locked := schedulableTask("locked-task", "locked", 1, "member", 600, 0)
		locked.ExecutionStart = datePointer(mustDate("2026-08-03"))
		locked.ExecutionEnd = datePointer(mustDate("2026-08-03"))
		locked.CommitmentStart = datePointer(mustDate("2026-08-03"))
		locked.CommitmentEnd = datePointer(mustDate("2026-08-03"))
		seedTask(t, database, locked)

		err := repository.RecalculatePortfolio(context.Background(), nil)
		if !errors.Is(err, schedulingdomain.ErrDataIntegrity) {
			t.Fatalf("error = %v, want ErrDataIntegrity", err)
		}
	})

	t.Run("locked dates reserve capacity while closed project is excluded", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		lockedProject := automaticProject("locked", 1, "2026-08-03", 0)
		lockedProject.Status = "locked"
		seedProject(t, database, lockedProject)
		closedProject := automaticProject("closed", 2, "2026-08-03", 0)
		closedProject.Status = "closed"
		seedProject(t, database, closedProject)
		seedProject(t, database, automaticProject("open", 3, "2026-08-03", 0))
		seedMember(t, database, "member", "5", "0")

		locked := schedulableTask("locked-task", "locked", 1, "member", 300, 0)
		locked.ExecutionStart = datePointer(mustDate("2026-08-03"))
		locked.ExecutionEnd = datePointer(mustDate("2026-08-03"))
		locked.CommitmentStart = datePointer(mustDate("2026-08-03"))
		locked.CommitmentEnd = datePointer(mustDate("2026-08-03"))
		seedTask(t, database, locked)
		closed := schedulableTask("closed-task", "closed", 1, "member", 300, 0)
		closed.ExecutionStart = datePointer(mustDate("2026-08-04"))
		closed.ExecutionEnd = datePointer(mustDate("2026-08-04"))
		closed.CommitmentStart = datePointer(mustDate("2026-08-04"))
		closed.CommitmentEnd = datePointer(mustDate("2026-08-04"))
		seedTask(t, database, closed)
		seedTask(t, database, schedulableTask("open-task", "open", 1, "member", 300, 0))

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}
		assertDate(t, "locked date", loadScheduledTask(t, database, "locked-task").ExecutionStart, "2026-08-03")
		assertDate(t, "open task starts after locked reservation", loadScheduledTask(t, database, "open-task").ExecutionStart, "2026-08-04")
		assertDate(t, "closed date remains unchanged", loadScheduledTask(t, database, "closed-task").ExecutionStart, "2026-08-04")
	})

	t.Run("persisted fixed allocation survives a later capacity reduction", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		lockedProject := automaticProject("locked", 1, "2026-08-03", 0)
		lockedProject.Status = "locked"
		seedProject(t, database, lockedProject)
		seedProject(t, database, automaticProject("open", 2, "2026-08-03", 0))
		seedMember(t, database, "member", "4", "0")

		locked := schedulableTask("locked-task", "locked", 1, "member", 480, 0)
		locked.ExecutionStart = datePointer(mustDate("2026-08-03"))
		locked.ExecutionEnd = datePointer(mustDate("2026-08-03"))
		locked.CommitmentStart = datePointer(mustDate("2026-08-03"))
		locked.CommitmentEnd = datePointer(mustDate("2026-08-03"))
		seedTask(t, database, locked)
		for _, timeline := range []string{"execution", "commitment"} {
			if err := database.Create(&allocationModel{
				TaskID: "locked-task", AssigneeID: "member", Timeline: timeline,
				AllocationDate: mustDate("2026-08-03"), AllocatedMinutes: "480.000000",
				RemainingCapacityMinutes: "0.000000", Sequence: 1,
			}).Error; err != nil {
				t.Fatal(err)
			}
		}
		seedTask(t, database, schedulableTask("open-task", "open", 1, "member", 240, 0))

		if err := repository.RecalculatePortfolio(context.Background(), nil); err != nil {
			t.Fatal(err)
		}

		assertDate(t, "locked date", loadScheduledTask(t, database, "locked-task").ExecutionStart, "2026-08-03")
		assertDate(t, "new task starts after over-capacity frozen reservation", loadScheduledTask(t, database, "open-task").ExecutionStart, "2026-08-04")
		rows := loadAllocations(t, database, "locked-task", "execution")
		if len(rows) != 1 || rows[0].AllocatedMinutes != "480.000000" {
			t.Fatalf("persisted fixed allocation = %#v, want unchanged 480 minutes", rows)
		}
	})
}

func TestInvalidOrderingRollsBackWithoutInventingTieBreaker_AC17_AC35(t *testing.T) {
	t.Run("duplicate active project priority", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("a", 1, "2026-08-03", 0))
		seedProject(t, database, automaticProject("b", 1, "2026-08-03", 0))
		seedMember(t, database, "member", "8", "0")
		seedTask(t, database, schedulableTask("task-a", "a", 1, "member", 60, 0))
		seedTask(t, database, schedulableTask("task-b", "b", 1, "member", 60, 0))

		err := repository.RecalculatePortfolio(context.Background(), nil)
		if !errors.Is(err, schedulingdomain.ErrDataIntegrity) {
			t.Fatalf("error = %v, want ErrDataIntegrity", err)
		}
		if loadScheduledTask(t, database, "task-a").ExecutionStart != nil {
			t.Fatal("task schedule changed despite invalid project priority")
		}
	})

	t.Run("duplicate sibling WBS position", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
		seedMember(t, database, "member", "8", "0")
		seedTask(t, database, schedulableTask("a", "project", 1, "member", 60, 0))
		seedTask(t, database, schedulableTask("b", "project", 1, "member", 60, 0))

		err := repository.RecalculatePortfolio(context.Background(), nil)
		if !errors.Is(err, schedulingdomain.ErrDataIntegrity) {
			t.Fatalf("error = %v, want ErrDataIntegrity", err)
		}
	})
}

func TestPersistenceFailureRollsBackDatesDependenciesAndAllocations_AC35(t *testing.T) {
	repository, database := schedulerRepository(t)
	seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
	seedMember(t, database, "member", "8", "0")
	seedTask(t, database, schedulableTask("a", "project", 1, "member", 60, 0))
	seedTask(t, database, schedulableTask("b", "project", 2, "member", 60, 0))
	if err := database.Exec(`
		CREATE TRIGGER fail_schedule_version
		BEFORE UPDATE OF schedule_version ON projects
		BEGIN
			SELECT RAISE(ABORT, 'injected schedule persistence failure');
		END;
	`).Error; err != nil {
		t.Fatal(err)
	}

	if err := repository.RecalculatePortfolio(context.Background(), []string{"project"}); err == nil {
		t.Fatal("expected injected persistence failure")
	}
	for _, taskID := range []string{"a", "b"} {
		task := loadScheduledTask(t, database, taskID)
		if task.ExecutionStart != nil || task.CommitmentStart != nil {
			t.Fatalf("task %s retained partial dates after rollback", taskID)
		}
	}
	var allocations, dependencies int64
	if err := database.Model(&allocationModel{}).Count(&allocations).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Model(&dependencyModel{}).Count(&dependencies).Error; err != nil {
		t.Fatal(err)
	}
	if allocations != 0 || dependencies != 0 {
		t.Fatalf("partial rollback state: allocations=%d dependencies=%d", allocations, dependencies)
	}
}

func TestConcurrentRecalculationSerializesAndPreservesOneAllocationPerTaskDate_AC36(t *testing.T) {
	repository, database := schedulerRepository(t)
	seedProject(t, database, automaticProject("project", 1, "2026-08-03", 0))
	seedMember(t, database, "member", "8", "0")
	seedTask(t, database, schedulableTask("a", "project", 1, "member", 240, 0))
	seedTask(t, database, schedulableTask("b", "project", 2, "member", 240, 0))

	start := make(chan struct{})
	errorsByRun := make([]error, 2)
	var wait sync.WaitGroup
	for index := range errorsByRun {
		wait.Add(1)
		go func(run int) {
			defer wait.Done()
			<-start
			errorsByRun[run] = repository.RecalculatePortfolio(context.Background(), []string{"project"})
		}(index)
	}
	close(start)
	wait.Wait()
	for index, err := range errorsByRun {
		if err != nil {
			t.Fatalf("concurrent run %d error = %v", index, err)
		}
	}

	var project projectModel
	if err := database.First(&project, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	if project.ScheduleVersion != 1 {
		t.Fatalf("schedule version = %d, want one confirmed dirty projection and one serialized no-op", project.ScheduleVersion)
	}
	var rows []allocationModel
	if err := database.Order("timeline ASC").Order("allocation_date ASC").Order("sequence ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		key := row.TaskID + "|" + row.Timeline + "|" + schedulingdomain.DateKey(row.AllocationDate)
		if seen[key] {
			t.Fatalf("duplicate allocation projection %s", key)
		}
		seen[key] = true
	}
}

func seedGapFixture(t *testing.T, database *gorm.DB, candidateEffort int) {
	t.Helper()
	manual := automaticProject("manual", 1, "2026-08-03", 0)
	manual.AutomaticScheduling = false
	seedProject(t, database, manual)
	seedProject(t, database, automaticProject("candidate-project", 2, "2026-08-03", 0))
	seedMember(t, database, "member", "5", "0")

	fixedA := schedulableTask("fixed-a", "manual", 1, "member", 300, 0)
	fixedA.ExecutionStart = datePointer(mustDate("2026-08-03"))
	fixedA.ExecutionEnd = datePointer(mustDate("2026-08-03"))
	fixedA.CommitmentStart = datePointer(mustDate("2026-08-03"))
	fixedA.CommitmentEnd = datePointer(mustDate("2026-08-03"))
	seedTask(t, database, fixedA)

	fixedB := schedulableTask("fixed-b", "manual", 2, "member", 300, 0)
	fixedB.ExecutionStart = datePointer(mustDate("2026-08-05"))
	fixedB.ExecutionEnd = datePointer(mustDate("2026-08-05"))
	fixedB.CommitmentStart = datePointer(mustDate("2026-08-05"))
	fixedB.CommitmentEnd = datePointer(mustDate("2026-08-05"))
	seedTask(t, database, fixedB)
	seedTask(t, database, schedulableTask("candidate", "candidate-project", 1, "member", candidateEffort, 0))
}

func automaticProject(id string, priority int, anchor string, buffer int) projectModel {
	return projectModel{
		ID: id, Status: "open", Priority: priority, AutomaticScheduling: true,
		SchedulingStartDate: datePointer(mustDate(anchor)), ProjectBuffer: buffer,
	}
}

func schedulableTask(id, projectID string, position int, memberID string, effortMinutes, lagDays int) taskModel {
	return taskModel{
		ID: id, ProjectID: projectID, ParentKey: "", Name: id, Position: position,
		AssigneeID: textPointer(memberID), EffortMinutes: intPointer(effortMinutes), LagDays: lagDays,
		CreatedAt: mustDate("2026-08-01"), UpdatedAt: mustDate("2026-08-01"),
	}
}

func seedProject(t *testing.T, database *gorm.DB, value projectModel) {
	t.Helper()
	if err := database.Create(&value).Error; err != nil {
		t.Fatal(err)
	}
}

func seedMember(t *testing.T, database *gorm.DB, id, dailyCapacity, memberBuffer string) {
	t.Helper()
	if err := database.Create(&memberModel{ID: id, DailyCapacity: dailyCapacity, BufferPercentage: memberBuffer}).Error; err != nil {
		t.Fatal(err)
	}
}

func seedTask(t *testing.T, database *gorm.DB, value taskModel) {
	t.Helper()
	if err := database.Create(&value).Error; err != nil {
		t.Fatal(err)
	}
}

func seedDependency(t *testing.T, database *gorm.DB, value dependencyModel) {
	t.Helper()
	value.CreatedAt = mustDate("2026-08-01")
	value.UpdatedAt = mustDate("2026-08-01")
	if err := database.Create(&value).Error; err != nil {
		t.Fatal(err)
	}
}

func loadScheduledTask(t *testing.T, database *gorm.DB, id string) taskModel {
	t.Helper()
	var task taskModel
	if err := database.First(&task, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	return task
}

func loadAllocations(t *testing.T, database *gorm.DB, taskID, timeline string) []allocationModel {
	t.Helper()
	var rows []allocationModel
	if err := database.Where("task_id = ? AND timeline = ?", taskID, timeline).Order("allocation_date ASC").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return rows
}

func assertAutomaticBlocker(t *testing.T, database *gorm.DB, blockingTaskID, blockedTaskID string) {
	t.Helper()
	var relation dependencyModel
	if err := database.Where("blocking_task_id = ? AND blocked_task_id = ?", blockingTaskID, blockedTaskID).First(&relation).Error; err != nil {
		t.Fatal(err)
	}
	if !relation.AutomaticOwned {
		t.Fatalf("relation %s -> %s is not automatic", blockingTaskID, blockedTaskID)
	}
	var count int64
	if err := database.Model(&dependencyModel{}).Where("blocked_task_id = ? AND automatic_owned = ?", blockedTaskID, true).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("automatic blockers for %s = %d, want 1", blockedTaskID, count)
	}
}

func assertDate(t *testing.T, label string, value *time.Time, expected string) {
	t.Helper()
	if value == nil || schedulingdomain.DateKey(*value) != expected {
		t.Fatalf("%s = %v, want %s", label, value, expected)
	}
}

func mustDate(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func textPointer(value string) *string { return &value }
func intPointer(value int) *int        { return &value }
