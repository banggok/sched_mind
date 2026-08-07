package gormrepo

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/projects/domain"
	"github.com/banggok/sched_mind/backend/internal/shared/listing"
	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testRepository(t *testing.T) *Repository {
	t.Helper()
	database, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{TranslateError: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&projectModel{}, &lifecycleTaskTestModel{}); err != nil {
		t.Fatal(err)
	}
	return New(database)
}

type lifecycleTaskTestModel struct {
	ID                          string `gorm:"primaryKey"`
	ProjectID                   string
	ParentID                    *string
	ExecutionStart              *time.Time
	ExecutionEnd                *time.Time
	CommitmentStart             *time.Time
	CommitmentEnd               *time.Time
	ExecutionUnscheduledReason  *string
	CommitmentUnscheduledReason *string
	ActualStart                 *time.Time
	ActualEnd                   *time.Time
}

func (lifecycleTaskTestModel) TableName() string { return "wbs_nodes" }

func TestRepositoryCreateListSearchAndPriority(t *testing.T) {
	repository := testRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 7, 26, 10, 0, 0, 0, time.UTC)
	alpha, err := repository.CreateNext(ctx, "a", "Alpha", true, nil, 20, now)
	if err != nil || alpha == nil {
		t.Fatalf("alpha: %#v %v", alpha, err)
	}
	beta, err := repository.CreateNext(ctx, "b", "Beta", true, nil, 20, now)
	if err != nil || beta == nil || beta.Priority != 2 {
		t.Fatalf("beta: %#v %v", beta, err)
	}
	if _, err := repository.CreateNext(ctx, "duplicate", "ALPHA", true, nil, 20, now); !errors.Is(err, domain.ErrNameExists) {
		t.Fatalf("duplicate: %v", err)
	}
	result, err := repository.List(ctx, listing.Query{Search: "al", Page: 1, PageSize: 5})
	if err != nil || result.Total != 1 || result.Items[0].ID != "a" {
		t.Fatalf("list: %#v %v", result, err)
	}
	called := false
	moved, err := repository.MovePriority(ctx, "b", domain.PriorityUp, now.Add(time.Hour), func(context.Context) error { called = true; return nil })
	if err != nil || moved == nil || moved.Priority != 1 || !called {
		t.Fatalf("move: %#v %v called=%v", moved, err, called)
	}
	if _, err := repository.MovePriority(ctx, "b", domain.PriorityUp, now, func(context.Context) error { return nil }); !errors.Is(err, domain.ErrPriorityMoveNotAllowed) {
		t.Fatalf("boundary: %v", err)
	}
}

func TestRepositoryLifecycleDeleteAndRollback(t *testing.T) {
	repository := testRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	value, err := repository.CreateNext(ctx, "a", "Alpha", true, nil, 20, now)
	if err != nil || value == nil {
		t.Fatal(err)
	}
	if _, err := repository.ChangeStatus(ctx, "a", domain.StatusLocked, now.Add(time.Hour), func(context.Context) error { t.Fatal("scheduler called for rejected lock"); return nil }); !errors.Is(err, domain.ErrCannotLockWithoutTasks) {
		t.Fatalf("lock zero tasks: %v", err)
	}
	if _, err := repository.ChangeStatus(ctx, "a", domain.StatusClosed, now, func(context.Context) error { t.Fatal("scheduler called for rejected close"); return nil }); !errors.Is(err, domain.ErrCannotCloseWithoutTasks) {
		t.Fatalf("close zero: %v", err)
	}
	_, _ = repository.CreateNext(ctx, "b", "Beta", true, nil, 20, now)
	before, _ := repository.Find(ctx, "b")
	_, err = repository.MovePriority(ctx, "b", domain.PriorityUp, now, func(context.Context) error { return errors.New("scheduler unavailable") })
	if err == nil {
		t.Fatal("expected scheduler failure")
	}
	after, _ := repository.Find(ctx, "b")
	if before == nil || after == nil || before.Priority != after.Priority {
		t.Fatalf("rollback: %#v %#v", before, after)
	}
	if err := repository.DeleteChildless(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Find(ctx, "a"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("deleted find: %v", err)
	}
}

func TestRepositoryStatusLifecycleSnapshotsSchedulingAndRollback_AC9_AC15_AC16_AC18_AC23(t *testing.T) {
	repository := testRepository(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)
	anchor := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	created, err := repository.CreateNext(ctx, "project", "Alpha", true, &anchor, 20, now)
	if err != nil || created == nil {
		t.Fatalf("create: %#v %v", created, err)
	}
	executionStart := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	executionEnd := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	commitmentEnd := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)
	leaf := lifecycleTaskTestModel{
		ID: "leaf", ProjectID: "project", ExecutionStart: &executionStart, ExecutionEnd: &executionEnd,
		CommitmentStart: &executionStart, CommitmentEnd: &commitmentEnd,
	}
	if err := repository.database.Create(&leaf).Error; err != nil {
		t.Fatal(err)
	}

	schedulerStatuses := make([]string, 0, 2)
	schedule := func(scheduleCtx context.Context) error {
		transaction := sharedpersistence.Transaction(scheduleCtx, repository.database)
		var status string
		if err := transaction.Model(&projectModel{}).Select("status").Where("id = ?", "project").Scan(&status).Error; err != nil {
			return err
		}
		schedulerStatuses = append(schedulerStatuses, status)
		return nil
	}
	locked, err := repository.ChangeStatus(ctx, "project", domain.StatusLocked, now.Add(2*time.Hour), func(context.Context) error {
		t.Fatal("scheduler must not run while locking a Project")
		return nil
	})
	if err != nil || locked == nil || locked.Status != domain.StatusLocked || locked.LockedExecutionSnapshot == nil || locked.LockedCommitmentSnapshot == nil {
		t.Fatalf("lock: %#v %v", locked, err)
	}
	var executionSnapshot []timelineSnapshotEntry
	if err := json.Unmarshal([]byte(*locked.LockedExecutionSnapshot), &executionSnapshot); err != nil {
		t.Fatal(err)
	}
	if len(executionSnapshot) != 1 || executionSnapshot[0].TaskID != "leaf" || executionSnapshot[0].Start == nil || !executionSnapshot[0].Start.Equal(executionStart) || executionSnapshot[0].End == nil || !executionSnapshot[0].End.Equal(executionEnd) {
		t.Fatalf("execution snapshot: %#v", executionSnapshot)
	}

	if _, err := repository.ChangeStatus(ctx, "project", domain.StatusClosed, now.Add(3*time.Hour), func(context.Context) error { t.Fatal("scheduler called for rejected incomplete close"); return nil }); !errors.Is(err, domain.ErrCannotCloseWithActiveTasks) {
		t.Fatalf("incomplete close: %v", err)
	}
	completedAt := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	if err := repository.database.Model(&lifecycleTaskTestModel{}).Where("id = ?", "leaf").Updates(map[string]any{"actual_start": completedAt, "actual_end": completedAt}).Error; err != nil {
		t.Fatal(err)
	}
	_, err = repository.ChangeStatus(ctx, "project", domain.StatusClosed, now.Add(4*time.Hour), func(context.Context) error {
		return errors.New("scheduler unavailable")
	})
	if err == nil {
		t.Fatal("expected close rollback")
	}
	afterFailedClose, _ := repository.Find(ctx, "project")
	if afterFailedClose == nil || afterFailedClose.Status != domain.StatusLocked || afterFailedClose.ClosedAt != nil {
		t.Fatalf("failed close persisted partial state: %#v", afterFailedClose)
	}

	closed, err := repository.ChangeStatus(ctx, "project", domain.StatusClosed, now.Add(5*time.Hour), schedule)
	if err != nil || closed == nil || closed.Status != domain.StatusClosed || closed.ClosedAt == nil {
		t.Fatalf("close: %#v %v", closed, err)
	}
	_, err = repository.ChangeStatus(ctx, "project", domain.StatusOpen, now.Add(6*time.Hour), func(context.Context) error {
		return errors.New("scheduler unavailable")
	})
	if err == nil {
		t.Fatal("expected reopen rollback")
	}
	afterFailedReopen, _ := repository.Find(ctx, "project")
	if afterFailedReopen == nil || afterFailedReopen.Status != domain.StatusClosed || afterFailedReopen.ClosedAt == nil {
		t.Fatalf("failed reopen persisted partial state: %#v", afterFailedReopen)
	}

	reopened, err := repository.ChangeStatus(ctx, "project", domain.StatusOpen, now.Add(7*time.Hour), schedule)
	if err != nil || reopened == nil || reopened.Status != domain.StatusOpen || reopened.ClosedAt != nil || reopened.LockedExecutionSnapshot != nil || reopened.LockedCommitmentSnapshot != nil {
		t.Fatalf("reopen: %#v %v", reopened, err)
	}
	if len(schedulerStatuses) != 2 || schedulerStatuses[0] != "closed" || schedulerStatuses[1] != "open" {
		t.Fatalf("scheduler transaction statuses: %#v", schedulerStatuses)
	}
}

func TestRepositorySettingsPersistenceAndSchedulerRollback(t *testing.T) {
	repository := testRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	created, err := repository.CreateNext(ctx, "a", "Alpha", true, nil, 20, now)
	if err != nil || created == nil || !created.AutomaticScheduling || created.ProjectBuffer != 20 {
		t.Fatalf("defaults: %#v %v", created, err)
	}
	anchor := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	disableScheduleCalls := 0
	updated, err := repository.UpdateSettings(ctx, "a", false, &anchor, 35, now.Add(time.Hour), func(context.Context, string) error {
		disableScheduleCalls++
		return nil
	}, func(context.Context, string, string) error {
		t.Fatal("legacy unscheduled marker must not own effective Group scheduling")
		return nil
	})
	if err != nil || updated == nil || updated.AutomaticScheduling || updated.ProjectBuffer != 35 || disableScheduleCalls != 1 {
		t.Fatalf("disable: %#v scheduleCalls=%d err=%v", updated, disableScheduleCalls, err)
	}
	if updated.SchedulingStartDate == nil || !updated.SchedulingStartDate.Equal(anchor) {
		t.Fatalf("persist anchor: %#v", updated)
	}
	_, err = repository.UpdateSettings(ctx, "a", true, &anchor, 40, now.Add(2*time.Hour), func(context.Context, string) error { return errors.New("scheduler unavailable") }, func(context.Context, string, string) error { t.Fatal("unexpected unscheduled marker"); return nil })
	if err == nil {
		t.Fatal("expected scheduler failure")
	}
	stored, _ := repository.Find(ctx, "a")
	if stored == nil || stored.AutomaticScheduling || stored.ProjectBuffer != 35 || stored.SchedulingStartDate == nil || !stored.SchedulingStartDate.Equal(anchor) {
		t.Fatalf("rollback: %#v", stored)
	}
	_, err = repository.UpdateDetails(ctx, "a", "Renamed", true, &anchor, 40, now.Add(3*time.Hour), func(context.Context, string) error { return errors.New("scheduler unavailable") }, func(context.Context, string, string) error { t.Fatal("unexpected unscheduled marker"); return nil })
	if err == nil {
		t.Fatal("expected combined update failure")
	}
	stored, _ = repository.Find(ctx, "a")
	if stored == nil || stored.Name != "Alpha" || stored.AutomaticScheduling || stored.ProjectBuffer != 35 {
		t.Fatalf("combined rollback: %#v", stored)
	}
}

func TestRepositoryDelegatesMissingProjectAnchorToEffectiveScheduler_US44_AC11_AC17_AC18(t *testing.T) {
	repository := testRepository(t)
	ctx := context.Background()
	now := time.Now().UTC()
	created, err := repository.CreateNext(ctx, "a", "Alpha", false, nil, 20, now)
	if err != nil || created == nil {
		t.Fatalf("create: %#v %v", created, err)
	}
	scheduleCalls := 0
	updated, err := repository.UpdateSettings(ctx, "a", true, nil, 20, now.Add(time.Hour), func(context.Context, string) error {
		scheduleCalls++
		return nil
	}, func(context.Context, string, string) error {
		t.Fatal("legacy Project-level unscheduled marker must not bypass effective Group resolution")
		return nil
	})
	if err != nil || updated == nil || !updated.AutomaticScheduling || updated.SchedulingStartDate != nil || scheduleCalls != 1 {
		t.Fatalf("enable without Project anchor: %#v scheduleCalls=%d err=%v", updated, scheduleCalls, err)
	}
}

func TestRepositoryRenameLockedProjectUpdatesOnlyIdentityFields_US31_AC11_US62_AC16A(t *testing.T) {
	repository := testRepository(t)
	ctx := context.Background()
	createdAt := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(2 * time.Hour)
	anchor := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	executionSnapshot := `[{"taskId":"task-1","end":"2026-08-04"}]`
	commitmentSnapshot := `[{"taskId":"task-1","end":"2026-08-05"}]`
	created, err := repository.CreateNext(ctx, "project", "Alpha", false, &anchor, 35, createdAt)
	if err != nil || created == nil {
		t.Fatalf("create: %#v %v", created, err)
	}
	if err := repository.database.Model(&projectModel{}).Where("id = ?", created.ID).Updates(map[string]interface{}{
		"status":                     string(domain.StatusLocked),
		"schedule_version":           int64(11),
		"locked_execution_snapshot":  executionSnapshot,
		"locked_commitment_snapshot": commitmentSnapshot,
	}).Error; err != nil {
		t.Fatal(err)
	}
	before, err := repository.Find(ctx, created.ID)
	if err != nil || before == nil {
		t.Fatalf("before: %#v %v", before, err)
	}

	renamed, err := repository.Rename(ctx, created.ID, " Renamed ", updatedAt)
	if err != nil || renamed == nil {
		t.Fatalf("rename: %#v %v", renamed, err)
	}
	after, err := repository.Find(ctx, created.ID)
	if err != nil || after == nil {
		t.Fatalf("after: %#v %v", after, err)
	}
	if after.Name != "Renamed" || !after.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("identity fields: %#v", after)
	}
	if after.ID != before.ID || after.Status != before.Status || after.Priority != before.Priority ||
		after.ScheduleVersion != before.ScheduleVersion || after.AutomaticScheduling != before.AutomaticScheduling ||
		after.ProjectBuffer != before.ProjectBuffer || after.SchedulingStartDate == nil || before.SchedulingStartDate == nil ||
		!after.SchedulingStartDate.Equal(*before.SchedulingStartDate) || after.LockedExecutionSnapshot == nil ||
		before.LockedExecutionSnapshot == nil || *after.LockedExecutionSnapshot != *before.LockedExecutionSnapshot ||
		after.LockedCommitmentSnapshot == nil || before.LockedCommitmentSnapshot == nil ||
		*after.LockedCommitmentSnapshot != *before.LockedCommitmentSnapshot || !after.CreatedAt.Equal(before.CreatedAt) {
		t.Fatalf("protected fields changed: before=%#v after=%#v", before, after)
	}
}

func TestRequiredReopenPlanUsesSimulationFixedPointInsteadOfConnectivity_US62_D07_D08_AC25_AC26(t *testing.T) {
	repository := testRepository(t)
	now := time.Date(2026, 8, 1, 8, 0, 0, 0, time.UTC)
	projects := []projectModel{
		{ID: "a", Name: "Project A", NameKey: "project a", Status: "locked", Priority: 1, AutomaticScheduling: true, AutoCalculateDate: true, ScheduleVersion: 7, CreatedAt: now, UpdatedAt: now},
		{ID: "b", Name: "Project B", NameKey: "project b", Status: "locked", Priority: 2, AutomaticScheduling: true, AutoCalculateDate: true, ScheduleVersion: 11, CreatedAt: now, UpdatedAt: now},
		{ID: "c", Name: "Project C", NameKey: "project c", Status: "open", Priority: 3, AutomaticScheduling: true, AutoCalculateDate: true, ScheduleVersion: 13, CreatedAt: now, UpdatedAt: now},
		{ID: "d", Name: "Project D", NameKey: "project d", Status: "open", Priority: 4, AutomaticScheduling: true, AutoCalculateDate: true, ScheduleVersion: 17, CreatedAt: now, UpdatedAt: now},
	}
	if err := repository.database.Create(&projects).Error; err != nil {
		t.Fatal(err)
	}

	calls := 0
	schedule := func(ctx context.Context) error {
		calls++
		database := sharedpersistence.Transaction(ctx, repository.database)
		var statuses []projectModel
		if err := database.Order("priority ASC").Find(&statuses).Error; err != nil {
			return err
		}
		byID := make(map[string]string, len(statuses))
		for _, project := range statuses {
			byID[project.ID] = project.Status
		}
		if byID["b"] != "open" {
			return errors.New("root B was not staged open during preview")
		}
		if calls == 1 {
			if byID["a"] != "locked" {
				return errors.New("A must remain locked during first preview")
			}
			return schedulingimpact.Error{
				Kind:           schedulingimpact.LockedProjectImpact,
				LockedProjects: []schedulingimpact.Project{{ID: "a", Name: "Project A", Status: "locked", Version: 7}},
				OpenProjects:   []schedulingimpact.Project{{ID: "c", Name: "Project C", Status: "open", Version: 13}},
			}
		}
		if byID["a"] != "open" {
			return errors.New("A must be staged open after joining the fixed-point closure")
		}
		return schedulingimpact.Error{
			Kind:         schedulingimpact.ConfirmationRequired,
			OpenProjects: []schedulingimpact.Project{{ID: "d", Name: "Project D", Status: "open", Version: 17}},
		}
	}

	plan, err := requiredReopenPlan(context.Background(), repository.database, "b", schedule)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("preview calls = %d, want 2 fixed-point iterations", calls)
	}
	if len(plan.Locked) != 2 || plan.Locked[0].ID != "a" || plan.Locked[1].ID != "b" {
		t.Fatalf("locked closure = %#v, want A/B", plan.Locked)
	}
	if len(plan.Open) != 1 || plan.Open[0].ID != "d" {
		t.Fatalf("open timeline impacts = %#v, want only final-impact D", plan.Open)
	}
	if plan.Token == "" {
		t.Fatal("reopen plan token is empty")
	}
	for _, project := range projects {
		var persisted projectModel
		if err := repository.database.First(&persisted, "id = ?", project.ID).Error; err != nil {
			t.Fatal(err)
		}
		if persisted.Status != project.Status {
			t.Fatalf("preview leaked status for %s: %q -> %q", project.ID, project.Status, persisted.Status)
		}
	}
}
