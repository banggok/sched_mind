package gormrepo

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dependencydomain "github.com/banggok/sched_mind/backend/internal/dependencies/domain"
	dependencyrepo "github.com/banggok/sched_mind/backend/internal/dependencies/infrastructure/gormrepo"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type reopenProjectRecord struct {
	ID                       string `gorm:"primaryKey"`
	Name                     string
	NameKey                  string
	Status                   string
	AutomaticScheduling      bool
	LockedExecutionSnapshot  *string
	LockedCommitmentSnapshot *string
}

func (reopenProjectRecord) TableName() string { return "projects" }

type reopenDependencyRecord struct {
	ID             string `gorm:"primaryKey"`
	BlockingTaskID string
	BlockedTaskID  string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (reopenDependencyRecord) TableName() string { return "task_dependencies" }

func reopenTestDB(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "reopen.db") + "?_busy_timeout=5000&_journal_mode=WAL"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(10)
	if err := db.AutoMigrate(&reopenProjectRecord{}, &nodeModel{}, &reopenDependencyRecord{}, &actualAllocationModel{}); err != nil {
		t.Fatal(err)
	}
	return New(db), db
}

func seedReopenGraph(t *testing.T, db *gorm.DB, status string) nodeModel {
	t.Helper()
	executionSnapshot, commitmentSnapshot := "execution-baseline", "commitment-baseline"
	project := reopenProjectRecord{ID: "project", Name: "Alpha", NameKey: "alpha", Status: status, AutomaticScheduling: true, LockedExecutionSnapshot: &executionSnapshot, LockedCommitmentSnapshot: &commitmentSnapshot}
	if err := db.Create(&project).Error; err != nil {
		t.Fatal(err)
	}
	parent, role, assignee, effort := "group", "role", "member", 390
	executionStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	executionEnd := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	commitmentStart := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	commitmentEnd := time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC)
	actualStart := time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC)
	actualEnd := time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC)
	created := time.Date(2026, 7, 1, 8, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 7, 2, 8, 0, 0, 0, time.UTC)
	nodes := []nodeModel{
		{ID: "group", ProjectID: "project", ParentKey: "", Name: "Group", NameKey: "group", Position: 1, CreatedAt: created, UpdatedAt: updated},
		{ID: "incoming", ProjectID: "project", ParentKey: "group", ParentID: &parent, Name: "Incoming", NameKey: "incoming", Position: 1, CreatedAt: created, UpdatedAt: updated},
		{ID: "task", ProjectID: "project", ParentKey: "group", ParentID: &parent, Name: "Build API", NameKey: "build api", Position: 2, RoleID: &role, AssigneeID: &assignee, EffortMinutes: &effort, ExecutionStart: &executionStart, ExecutionEnd: &executionEnd, CommitmentStart: &commitmentStart, CommitmentEnd: &commitmentEnd, ActualStart: &actualStart, ActualEnd: &actualEnd, CreatedAt: created, UpdatedAt: updated},
		{ID: "outgoing", ProjectID: "project", ParentKey: "group", ParentID: &parent, Name: "Outgoing", NameKey: "outgoing", Position: 3, CreatedAt: created, UpdatedAt: updated},
		{ID: "candidate-source", ProjectID: "project", ParentKey: "group", ParentID: &parent, Name: "Candidate Source", NameKey: "candidate source", Position: 4, CreatedAt: created, UpdatedAt: updated},
	}
	if err := db.Create(&nodes).Error; err != nil {
		t.Fatal(err)
	}
	links := []reopenDependencyRecord{
		{ID: "incoming-link", BlockingTaskID: "incoming", BlockedTaskID: "task", CreatedAt: created, UpdatedAt: updated},
		{ID: "outgoing-link", BlockingTaskID: "task", BlockedTaskID: "outgoing", CreatedAt: created, UpdatedAt: updated},
	}
	if err := db.Create(&links).Error; err != nil {
		t.Fatal(err)
	}
	return nodes[2]
}

func loadNodeModel(t *testing.T, db *gorm.DB, id string) nodeModel {
	t.Helper()
	var value nodeModel
	if err := db.First(&value, "id = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	return value
}

func loadLinks(t *testing.T, db *gorm.DB) []dependencyLinkModel {
	t.Helper()
	var values []dependencyLinkModel
	if err := db.Order("id ASC").Find(&values).Error; err != nil {
		t.Fatal(err)
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	return values
}

func withoutReopenManagedFields(value nodeModel) nodeModel {
	value.ActualStart = nil
	value.ActualEnd = nil
	value.UpdatedAt = time.Time{}
	return value
}

func TestRepositoryReopenOpenProjectIsAtomicAndPreservesTaskAndDependencies_AC1_AC2_AC4_AC5_AC7(t *testing.T) {
	repo, db := reopenTestDB(t)
	before := seedReopenGraph(t, db, "open")
	var beforeProject reopenProjectRecord
	if err := db.First(&beforeProject, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	beforeLinks := loadLinks(t, db)
	now := time.Date(2026, 7, 29, 8, 0, 0, 0, time.UTC)
	forecastCalls := 0
	confirmed, err := repo.Reopen(context.Background(), "project", "task", now, func(_ context.Context, projectIDs []string) error {
		forecastCalls++
		if len(projectIDs) != 1 || projectIDs[0] != "project" {
			t.Fatalf("invalidation projects=%#v", projectIDs)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	after := loadNodeModel(t, db, "task")
	if after.ActualStart != nil || after.ActualEnd != nil || confirmed == nil || confirmed.Executable.ActualStart != nil || confirmed.Executable.ActualEnd != nil {
		t.Fatalf("actual dates persisted=(%v,%v) confirmed=%#v", after.ActualStart, after.ActualEnd, confirmed)
	}
	if forecastCalls != 1 {
		t.Fatalf("forecast calls=%d", forecastCalls)
	}
	if !reflect.DeepEqual(withoutReopenManagedFields(before), withoutReopenManagedFields(after)) {
		t.Fatalf("non-reopen fields changed\nbefore=%#v\nafter=%#v", before, after)
	}
	if after.UpdatedAt != now {
		t.Fatalf("updated at=%v", after.UpdatedAt)
	}
	afterLinks := loadLinks(t, db)
	if !reflect.DeepEqual(beforeLinks, afterLinks) {
		t.Fatalf("dependency endpoints changed\nbefore=%#v\nafter=%#v", beforeLinks, afterLinks)
	}
	var afterProject reopenProjectRecord
	if err := db.First(&afterProject, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeProject, afterProject) || afterProject.Status != "open" {
		t.Fatalf("open project changed\nbefore=%#v\nafter=%#v", beforeProject, afterProject)
	}
}

func TestRepositoryReopenAutomaticSchedulingOffPreservesManualTimelinesAndCallsForecast_AC7(t *testing.T) {
	repo, db := reopenTestDB(t)
	before := seedReopenGraph(t, db, "open")
	if err := db.Model(&reopenProjectRecord{}).Where("id = ?", "project").Update("automatic_scheduling", false).Error; err != nil {
		t.Fatal(err)
	}
	forecastCalls := 0
	if _, err := repo.Reopen(context.Background(), "project", "task", time.Now().UTC(), func(context.Context, []string) error {
		forecastCalls++
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	after := loadNodeModel(t, db, "task")
	if forecastCalls != 1 {
		t.Fatalf("forecast calls=%d", forecastCalls)
	}
	if !reflect.DeepEqual(before.ExecutionStart, after.ExecutionStart) || !reflect.DeepEqual(before.ExecutionEnd, after.ExecutionEnd) || !reflect.DeepEqual(before.CommitmentStart, after.CommitmentStart) || !reflect.DeepEqual(before.CommitmentEnd, after.CommitmentEnd) {
		t.Fatalf("manual timelines changed\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestRepositoryReopenLockedIsRejectedAndPreservesStatusBaselinesAndActualDates_US62_AC16(t *testing.T) {
	repo, db := reopenTestDB(t)
	beforeTask := seedReopenGraph(t, db, "locked")
	var beforeProject reopenProjectRecord
	if err := db.First(&beforeProject, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	calls := 0
	_, err := repo.Reopen(context.Background(), "project", "task", time.Now().UTC(), func(context.Context, []string) error {
		calls++
		return nil
	})
	if !errors.Is(err, domain.ErrProjectLockedReadOnly) || calls != 0 {
		t.Fatalf("err=%v invalidation calls=%d", err, calls)
	}
	var afterProject reopenProjectRecord
	if err := db.First(&afterProject, "id = ?", "project").Error; err != nil {
		t.Fatal(err)
	}
	afterTask := loadNodeModel(t, db, "task")
	if !reflect.DeepEqual(beforeProject, afterProject) || !reflect.DeepEqual(beforeTask, afterTask) {
		t.Fatalf("locked rejection changed state\nproject before=%#v after=%#v\ntask before=%#v after=%#v", beforeProject, afterProject, beforeTask, afterTask)
	}
}

func TestRepositoryReopenRejectsClosedGroupingAndUnfinishedWithoutForecast_AC2_AC9_AC13(t *testing.T) {
	cases := []struct {
		name    string
		status  string
		id      string
		prepare func(*testing.T, *gorm.DB)
		want    error
	}{
		{name: "closed", status: "closed", id: "task", want: domain.ErrProjectClosedReadOnly},
		{name: "group", status: "open", id: "group", want: domain.ErrExecutableOnly},
		{name: "unfinished", status: "open", id: "task", prepare: func(t *testing.T, db *gorm.DB) {
			if err := db.Model(&nodeModel{}).Where("id = ?", "task").Update("actual_end", nil).Error; err != nil {
				t.Fatal(err)
			}
		}, want: domain.ErrTaskNotCompleted},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, db := reopenTestDB(t)
			seedReopenGraph(t, db, tc.status)
			if tc.prepare != nil {
				tc.prepare(t, db)
			}
			beforeTarget := loadNodeModel(t, db, tc.id)
			beforeLinks := loadLinks(t, db)
			calls := 0
			_, err := repo.Reopen(context.Background(), "project", tc.id, time.Now().UTC(), func(context.Context, []string) error { calls++; return nil })
			if !errors.Is(err, tc.want) || calls != 0 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
			afterTarget := loadNodeModel(t, db, tc.id)
			afterLinks := loadLinks(t, db)
			if !reflect.DeepEqual(beforeTarget, afterTarget) {
				t.Fatalf("rejected Reopen changed target\nbefore=%#v\nafter=%#v", beforeTarget, afterTarget)
			}
			if !reflect.DeepEqual(beforeLinks, afterLinks) {
				t.Fatalf("rejected Reopen changed dependency endpoints\nbefore=%#v\nafter=%#v", beforeLinks, afterLinks)
			}
		})
	}
}

func TestRepositoryReopenRefreshesDependencyProjectionAndCurrentCompletionRules_AC10(t *testing.T) {
	repo, db := reopenTestDB(t)
	seedReopenGraph(t, db, "open")
	dependencies := dependencyrepo.New(db)
	ctx := context.Background()
	dependencyMutationAt := time.Date(2026, 7, 29, 9, 0, 0, 0, time.UTC)

	beforeDetail, err := dependencies.List(ctx, "outgoing")
	if err != nil {
		t.Fatal(err)
	}
	if len(beforeDetail.BlockedBy) != 1 || beforeDetail.BlockedBy[0].Dependency.ID != "outgoing-link" || beforeDetail.BlockedBy[0].Task.ID != "task" || beforeDetail.BlockedBy[0].Task.ActualEnd == nil {
		t.Fatalf("completed dependency projection=%#v", beforeDetail.BlockedBy)
	}
	beforeCandidates, err := dependencies.Candidates(ctx, "candidate-source", dependencydomain.Blocks, "Build API", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if containsDependencyCandidate(beforeCandidates.Items, "task") {
		t.Fatal("completed Task was eligible as a blocked candidate before Reopen")
	}
	if err := dependencies.Delete(ctx, "incoming-link", dependencyMutationAt, func(context.Context, []string) error { return nil }); !errors.Is(err, dependencydomain.ErrCompletedHistory) {
		t.Fatalf("completed dependency history error=%v", err)
	}

	if _, err := repo.Reopen(ctx, "project", "task", time.Now().UTC(), func(context.Context, []string) error { return nil }); err != nil {
		t.Fatal(err)
	}

	afterDetail, err := dependencies.List(ctx, "outgoing")
	if err != nil {
		t.Fatal(err)
	}
	if len(afterDetail.BlockedBy) != 1 || afterDetail.BlockedBy[0].Dependency.ID != "outgoing-link" || afterDetail.BlockedBy[0].Task.ID != "task" || afterDetail.BlockedBy[0].Task.ActualEnd != nil {
		t.Fatalf("reopened dependency projection=%#v", afterDetail.BlockedBy)
	}
	afterCandidates, err := dependencies.Candidates(ctx, "candidate-source", dependencydomain.Blocks, "Build API", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if !containsDependencyCandidate(afterCandidates.Items, "task") {
		t.Fatalf("reopened Task missing from blocked candidates: %#v", afterCandidates.Items)
	}
	if err := dependencies.Delete(ctx, "incoming-link", dependencyMutationAt.Add(time.Minute), func(context.Context, []string) error { return nil }); err != nil {
		t.Fatalf("historical restriction did not use current Actual End: %v", err)
	}
}

func containsDependencyCandidate(values []dependencydomain.Task, id string) bool {
	for _, value := range values {
		if value.ID == id {
			return true
		}
	}
	return false
}

func TestRepositoryReopenForecastFailureRollsBackPersistedState_AC11(t *testing.T) {
	repo, db := reopenTestDB(t)
	before := seedReopenGraph(t, db, "open")
	forecastFailure := errors.New("forecast coordination failed")
	_, err := repo.Reopen(context.Background(), "project", "task", time.Now().UTC(), func(context.Context, []string) error { return forecastFailure })
	if !errors.Is(err, forecastFailure) {
		t.Fatalf("err=%v", err)
	}
	after := loadNodeModel(t, db, "task")
	if after.ActualEnd == nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("rollback did not restore state\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestRepositoryReopenPersistenceFailureHasNoPartialMutation_AC11(t *testing.T) {
	repo, db := reopenTestDB(t)
	before := seedReopenGraph(t, db, "open")
	failure := errors.New("forced update failure")
	callbackName := "test:fail-reopen-update"
	if err := db.Callback().Update().Before("gorm:update").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement.Table == "wbs_nodes" {
			tx.AddError(failure)
		}
	}); err != nil {
		t.Fatal(err)
	}
	forecastCalls := 0
	_, err := repo.Reopen(context.Background(), "project", "task", time.Now().UTC(), func(context.Context, []string) error { forecastCalls++; return nil })
	if !errors.Is(err, failure) || forecastCalls != 0 {
		t.Fatalf("err=%v forecast=%d", err, forecastCalls)
	}
	if err := db.Callback().Update().Remove(callbackName); err != nil {
		t.Fatal(err)
	}
	after := loadNodeModel(t, db, "task")
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("partial state after persistence failure\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestRepositoryReopenConcurrentCommandsProduceOneSuccessOneConflict_AC12(t *testing.T) {
	repo, db := reopenTestDB(t)
	seedReopenGraph(t, db, "open")
	enteredForecast := make(chan struct{})
	releaseForecast := make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseForecast) }) }
	secondObservedCompleted := make(chan struct{})
	type concurrentRequestKey struct{}
	queryCallback := "test:observe-second-reopen-read"
	if err := db.Callback().Query().After("gorm:query").Register(queryCallback, func(tx *gorm.DB) {
		if tx.Statement.Context.Value(concurrentRequestKey{}) != "second" || tx.Statement.Table != "wbs_nodes" {
			return
		}
		if node, ok := tx.Statement.Dest.(*nodeModel); ok && node.ActualEnd != nil {
			select {
			case <-secondObservedCompleted:
			default:
				close(secondObservedCompleted)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Callback().Query().Remove(queryCallback); err != nil {
			t.Errorf("remove query callback: %v", err)
		}
	})
	var forecastCalls atomic.Int32
	forecast := func(context.Context, []string) error {
		if forecastCalls.Add(1) == 1 {
			close(enteredForecast)
			<-releaseForecast
		}
		return nil
	}
	results := make(chan error, 2)
	go func() {
		_, err := repo.Reopen(context.Background(), "project", "task", time.Now().UTC(), forecast)
		results <- err
	}()
	select {
	case <-enteredForecast:
	case <-time.After(5 * time.Second):
		t.Fatal("first Reopen did not reach Forecast coordination")
	}
	go func() {
		ctx := context.WithValue(context.Background(), concurrentRequestKey{}, "second")
		_, err := repo.Reopen(ctx, "project", "task", time.Now().UTC().Add(time.Second), forecast)
		results <- err
	}()
	select {
	case <-secondObservedCompleted:
	case <-time.After(5 * time.Second):
		release()
		t.Fatal("second Reopen did not read the completed Task concurrently")
	}
	release()
	var err1, err2 error
	select {
	case err1 = <-results:
	case <-time.After(5 * time.Second):
		t.Fatal("first concurrent Reopen did not finish")
	}
	select {
	case err2 = <-results:
	case <-time.After(5 * time.Second):
		t.Fatal("second concurrent Reopen did not finish")
	}
	successes, conflicts := 0, 0
	for _, err := range []error{err1, err2} {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, domain.ErrTaskReopenConflict):
			conflicts++
		default:
			t.Fatalf("unexpected concurrent error: %v", err)
		}
	}
	if successes != 1 || conflicts != 1 || forecastCalls.Load() != 1 {
		t.Fatalf("success=%d conflict=%d forecast=%d errors=%v/%v", successes, conflicts, forecastCalls.Load(), err1, err2)
	}
	if value := loadNodeModel(t, db, "task"); value.ActualStart != nil || value.ActualEnd != nil {
		t.Fatalf("actual dates remained: start=%v end=%v", value.ActualStart, value.ActualEnd)
	}
}

func TestRepositoryRepeatedReopenAfterConfirmedUnfinishedReturnsNotCompleted_AC2_AC13(t *testing.T) {
	repo, db := reopenTestDB(t)
	seedReopenGraph(t, db, "open")
	var calls atomic.Int32
	forecast := func(context.Context, []string) error { calls.Add(1); return nil }
	if _, err := repo.Reopen(context.Background(), "project", "task", time.Now().UTC(), forecast); err != nil {
		t.Fatal(err)
	}
	_, err := repo.Reopen(context.Background(), "project", "task", time.Now().UTC().Add(time.Second), forecast)
	if !errors.Is(err, domain.ErrTaskNotCompleted) || calls.Load() != 1 {
		t.Fatalf("err=%v forecast=%d", err, calls.Load())
	}
}

func TestRepositoryReopenIdentityPredicateIncludesProjectID_AC5(t *testing.T) {
	repo, db := reopenTestDB(t)
	seedReopenGraph(t, db, "open")
	if err := db.Create(&reopenProjectRecord{ID: "other", Status: "open"}).Error; err != nil {
		t.Fatal(err)
	}
	_, err := repo.Reopen(context.Background(), "other", "task", time.Now().UTC(), func(context.Context, []string) error { return nil })
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	if loadNodeModel(t, db, "task").ActualEnd == nil {
		t.Fatal("cross-project request mutated task")
	}
}

func TestReopenTestHelpersUseExactDependencyEndpointIdentity(t *testing.T) {
	_, db := reopenTestDB(t)
	seedReopenGraph(t, db, "open")
	links := loadLinks(t, db)
	got := fmt.Sprintf("%s:%s>%s,%s:%s>%s", links[0].ID, links[0].BlockingTaskID, links[0].BlockedTaskID, links[1].ID, links[1].BlockingTaskID, links[1].BlockedTaskID)
	if got != "incoming-link:incoming>task,outgoing-link:task>outgoing" {
		t.Fatalf("fixtures do not distinguish endpoint preservation: %s", got)
	}
}
