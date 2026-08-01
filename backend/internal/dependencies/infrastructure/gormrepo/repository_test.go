package gormrepo

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/dependencies/domain"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func testRepository(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&projectModel{}, &taskModel{}, &dependencyModel{}); err != nil {
		t.Fatal(err)
	}
	return New(db), db
}

func TestConcurrentDuplicateCreatesOneRelation(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)
	start := make(chan struct{})
	errorsByRequest := make([]error, 2)
	var wait sync.WaitGroup
	for i, id := range []string{"first", "second"} {
		wait.Add(1)
		go func(index int, dependencyID string) {
			defer wait.Done()
			<-start
			_, errorsByRequest[index] = repo.Create(context.Background(), testDependency(dependencyID, "a", "b"), func(context.Context, []string) error { return nil })
		}(i, id)
	}
	close(start)
	wait.Wait()
	successes, conflicts := 0, 0
	for _, err := range errorsByRequest {
		if err == nil {
			successes++
		} else if errors.Is(err, domain.ErrAlreadyExists) {
			conflicts++
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("errors=%v, successes=%d conflicts=%d", errorsByRequest, successes, conflicts)
	}
	var count int64
	if err := db.Model(&dependencyModel{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("count=%d", count)
	}
}

func TestConcurrentOppositeEdgesRemainAcyclic(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)
	start := make(chan struct{})
	errorsByRequest := make([]error, 2)
	var wait sync.WaitGroup
	values := []domain.Dependency{testDependency("ab", "a", "b"), testDependency("ba", "b", "a")}
	for i, value := range values {
		wait.Add(1)
		go func(index int, candidate domain.Dependency) {
			defer wait.Done()
			<-start
			_, errorsByRequest[index] = repo.Create(context.Background(), candidate, func(context.Context, []string) error { return nil })
		}(i, value)
	}
	close(start)
	wait.Wait()
	successes, cycles := 0, 0
	for _, err := range errorsByRequest {
		if err == nil {
			successes++
		} else if errors.Is(err, domain.ErrCycle) {
			cycles++
		}
	}
	if successes != 1 || cycles != 1 {
		t.Fatalf("errors=%v, successes=%d cycles=%d", errorsByRequest, successes, cycles)
	}
}

func seedProjectAndTasks(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Create(&projectModel{ID: "project", Name: "Alpha", NameKey: "alpha", Status: "open", AutomaticScheduling: true}).Error; err != nil {
		t.Fatal(err)
	}
	tasks := []taskModel{{ID: "a", ProjectID: "project", Name: "API", NameKey: "api", ParentKey: "", Position: 1}, {ID: "b", ProjectID: "project", Name: "Build", NameKey: "build", ParentKey: "", Position: 2}, {ID: "c", ProjectID: "project", Name: "Client", NameKey: "client", ParentKey: "", Position: 3}}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
}

func testDependency(id, from, to string) domain.Dependency {
	now := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	return domain.Dependency{ID: id, BlockingTaskID: from, BlockedTaskID: to, ManualOwned: true, CreatedAt: now, UpdatedAt: now}
}

func TestCreateListAndDuplicateConstraint(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)
	calls := 0
	created, err := repo.Create(context.Background(), testDependency("ab", "a", "b"), func(context.Context, []string) error { calls++; return nil })
	if err != nil || created == nil || calls != 1 {
		t.Fatalf("Create()=(%#v,%v), calls=%d", created, err, calls)
	}
	detail, err := repo.List(context.Background(), "b")
	if err != nil || len(detail.BlockedBy) != 1 || detail.BlockedBy[0].Task.Name != "API" {
		t.Fatalf("List()=(%#v,%v)", detail, err)
	}
	value, err := repo.Create(context.Background(), testDependency("duplicate", "a", "b"), func(context.Context, []string) error { return nil })
	if value != nil || !errors.Is(err, domain.ErrAlreadyExists) {
		t.Fatalf("duplicate=(%#v,%v)", value, err)
	}
}

func TestListProjectsOneRelationFromBothTaskDirections(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)

	created, err := repo.Create(
		context.Background(),
		testDependency("ab", "a", "b"),
		func(context.Context, []string) error { return nil },
	)
	if err != nil {
		t.Fatalf("Create(a -> b) error=%v", err)
	}
	if created == nil {
		t.Fatal("Create(a -> b) returned nil dependency")
	}

	blockingTaskDetail, err := repo.List(context.Background(), "a")
	if err != nil {
		t.Fatalf("List(a) error=%v", err)
	}
	if len(blockingTaskDetail.BlockedBy) != 0 {
		t.Fatalf("List(a).BlockedBy=%d, want 0", len(blockingTaskDetail.BlockedBy))
	}
	if len(blockingTaskDetail.Blocks) != 1 {
		t.Fatalf("List(a).Blocks=%d, want 1", len(blockingTaskDetail.Blocks))
	}

	blockedTaskDetail, err := repo.List(context.Background(), "b")
	if err != nil {
		t.Fatalf("List(b) error=%v", err)
	}
	if len(blockedTaskDetail.Blocks) != 0 {
		t.Fatalf("List(b).Blocks=%d, want 0", len(blockedTaskDetail.Blocks))
	}
	if len(blockedTaskDetail.BlockedBy) != 1 {
		t.Fatalf("List(b).BlockedBy=%d, want 1", len(blockedTaskDetail.BlockedBy))
	}

	blocksItem := blockingTaskDetail.Blocks[0]
	blockedByItem := blockedTaskDetail.BlockedBy[0]
	if blocksItem.Dependency.ID != created.ID || blockedByItem.Dependency.ID != created.ID {
		t.Fatalf(
			"relation IDs=(Blocks:%q BlockedBy:%q), want %q",
			blocksItem.Dependency.ID,
			blockedByItem.Dependency.ID,
			created.ID,
		)
	}
	if blocksItem.Task.ID != "b" {
		t.Fatalf("List(a).Blocks[0].Task.ID=%q, want b", blocksItem.Task.ID)
	}
	if blockedByItem.Task.ID != "a" {
		t.Fatalf("List(b).BlockedBy[0].Task.ID=%q, want a", blockedByItem.Task.ID)
	}
}

func TestCreateSupportsMultipleBlockers(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)
	callback := func(context.Context, []string) error { return nil }

	for _, value := range []domain.Dependency{
		testDependency("ac", "a", "c"),
		testDependency("bc", "b", "c"),
	} {
		if _, err := repo.Create(context.Background(), value, callback); err != nil {
			t.Fatalf("Create(%s -> %s) error=%v", value.BlockingTaskID, value.BlockedTaskID, err)
		}
	}

	detail, err := repo.List(context.Background(), "c")
	if err != nil {
		t.Fatalf("List(c) error=%v", err)
	}
	if len(detail.BlockedBy) != 2 {
		t.Fatalf("BlockedBy=%d, want 2", len(detail.BlockedBy))
	}

	seen := map[string]bool{}
	for _, item := range detail.BlockedBy {
		seen[item.Task.ID] = true
	}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("BlockedBy tasks=%v, want a and b", seen)
	}
}

func TestCreateSupportsMultipleBlockedTasks(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)
	callback := func(context.Context, []string) error { return nil }

	for _, value := range []domain.Dependency{
		testDependency("ab", "a", "b"),
		testDependency("ac", "a", "c"),
	} {
		if _, err := repo.Create(context.Background(), value, callback); err != nil {
			t.Fatalf("Create(%s -> %s) error=%v", value.BlockingTaskID, value.BlockedTaskID, err)
		}
	}

	detail, err := repo.List(context.Background(), "a")
	if err != nil {
		t.Fatalf("List(a) error=%v", err)
	}
	if len(detail.Blocks) != 2 {
		t.Fatalf("Blocks=%d, want 2", len(detail.Blocks))
	}

	seen := map[string]bool{}
	for _, item := range detail.Blocks {
		seen[item.Task.ID] = true
	}
	if !seen["b"] || !seen["c"] {
		t.Fatalf("Blocks tasks=%v, want b and c", seen)
	}
}

func TestCreateRejectsIndirectCrossProjectCycleWithFullPath(t *testing.T) {
	repo, db := testRepository(t)

	projects := []projectModel{
		{ID: "alpha", Name: "Alpha", NameKey: "alpha", Status: "open", AutomaticScheduling: true},
		{ID: "beta", Name: "Beta", NameKey: "beta", Status: "open", AutomaticScheduling: true},
		{ID: "gamma", Name: "Gamma", NameKey: "gamma", Status: "open", AutomaticScheduling: true},
	}
	if err := db.Create(&projects).Error; err != nil {
		t.Fatal(err)
	}

	tasks := []taskModel{
		{ID: "a", ProjectID: "alpha", Name: "Task A", NameKey: "task a", ParentKey: "", Position: 1},
		{ID: "b", ProjectID: "beta", Name: "Task B", NameKey: "task b", ParentKey: "", Position: 1},
		{ID: "c", ProjectID: "gamma", Name: "Task C", NameKey: "task c", ParentKey: "", Position: 1},
	}
	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}

	callback := func(context.Context, []string) error { return nil }
	for _, value := range []domain.Dependency{
		testDependency("ab", "a", "b"),
		testDependency("bc", "b", "c"),
	} {
		if _, err := repo.Create(context.Background(), value, callback); err != nil {
			t.Fatalf("seed Create(%s -> %s) error=%v", value.BlockingTaskID, value.BlockedTaskID, err)
		}
	}

	created, err := repo.Create(
		context.Background(),
		testDependency("ca", "c", "a"),
		callback,
	)
	if created != nil {
		t.Fatalf("Create(c -> a) result=%#v, want nil", created)
	}

	var cycle *domain.CycleError
	if !errors.As(err, &cycle) {
		t.Fatalf("Create(c -> a) error=%v, want CycleError", err)
	}

	wantTaskIDs := []string{"a", "b", "c", "a"}
	wantProjects := []string{"Alpha", "Beta", "Gamma", "Alpha"}
	if len(cycle.Path) != len(wantTaskIDs) {
		t.Fatalf("cycle path length=%d, want %d: %#v", len(cycle.Path), len(wantTaskIDs), cycle.Path)
	}
	for i := range wantTaskIDs {
		if cycle.Path[i].TaskID != wantTaskIDs[i] {
			t.Fatalf("cycle.Path[%d].TaskID=%q, want %q; path=%#v", i, cycle.Path[i].TaskID, wantTaskIDs[i], cycle.Path)
		}
		if cycle.Path[i].ProjectName != wantProjects[i] {
			t.Fatalf("cycle.Path[%d].ProjectName=%q, want %q; path=%#v", i, cycle.Path[i].ProjectName, wantProjects[i], cycle.Path)
		}
	}

	var dependencies []dependencyModel
	if err := db.Order("id ASC").Find(&dependencies).Error; err != nil {
		t.Fatal(err)
	}
	if len(dependencies) != 2 {
		t.Fatalf("dependency count=%d, want 2", len(dependencies))
	}
	if dependencies[0].ID != "ab" || dependencies[1].ID != "bc" {
		t.Fatalf("dependencies=%#v, want only existing ab and bc relations", dependencies)
	}
}

func TestCandidatesExcludeCompletedBlockedDirection(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)
	actual := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	if err := db.Model(&taskModel{}).Where("id = ?", "b").Updates(map[string]any{"actual_start": actual, "actual_end": actual}).Error; err != nil {
		t.Fatal(err)
	}
	page, err := repo.Candidates(context.Background(), "a", domain.Blocks, "", 1, 5)
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range page.Items {
		if candidate.ID == "b" {
			t.Fatal("completed task returned as blocked candidate")
		}
	}
}

func TestDeleteRejectsCompletedBlockedHistory(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)

	if err := db.Create(&dependencyModel{
		ID:             "ab",
		BlockingTaskID: "a",
		BlockedTaskID:  "b",
		ManualOwned:    true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}).Error; err != nil {
		t.Fatal(err)
	}

	actual := time.Now()

	if err := db.Model(&taskModel{}).
		Where("id = ?", "b").
		Updates(map[string]any{"actual_start": actual, "actual_end": actual}).Error; err != nil {
		t.Fatal(err)
	}

	if err := repo.Delete(
		context.Background(),
		"ab",
		time.Now(),
		func(context.Context, []string) error { return nil },
	); !errors.Is(err, domain.ErrCompletedHistory) {
		t.Fatalf("Delete() error=%v", err)
	}

	var count int64
	if err := db.Model(&dependencyModel{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 1 {
		t.Fatalf("dependency count=%d, want 1", count)
	}

	var dependency dependencyModel
	if err := db.First(&dependency, "id = ?", "ab").Error; err != nil {
		t.Fatalf("dependency removed unexpectedly: %v", err)
	}

	if dependency.BlockingTaskID != "a" {
		t.Fatalf(
			"BlockingTaskID=%q, want a",
			dependency.BlockingTaskID,
		)
	}

	if dependency.BlockedTaskID != "b" {
		t.Fatalf(
			"BlockedTaskID=%q, want b",
			dependency.BlockedTaskID,
		)
	}
}

func seedCrossProjectTasks(t *testing.T, db *gorm.DB) {
	t.Helper()

	projects := []projectModel{
		{
			ID:                  "alpha",
			Name:                "Alpha",
			NameKey:             "alpha",
			Status:              "open",
			AutomaticScheduling: true,
		},
		{
			ID:                  "beta",
			Name:                "Beta",
			NameKey:             "beta",
			Status:              "open",
			AutomaticScheduling: true,
		},
	}

	if err := db.Create(&projects).Error; err != nil {
		t.Fatal(err)
	}

	tasks := []taskModel{
		{
			ID:        "task-a",
			ProjectID: "alpha",
			Name:      "Task A",
			NameKey:   "task a",
			ParentKey: "",
			Position:  1,
		},
		{
			ID:        "task-b",
			ProjectID: "beta",
			Name:      "Task B",
			NameKey:   "task b",
			ParentKey: "",
			Position:  1,
		},
	}

	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
}

func TestCreateAcceptsCrossProjectDependencyAndInvalidatesBothProjects(
	t *testing.T,
) {
	repo, db := testRepository(t)
	seedCrossProjectTasks(t, db)

	var affectedProjects []string

	created, err := repo.Create(
		context.Background(),
		testDependency("alpha-beta", "task-a", "task-b"),
		func(_ context.Context, projectIDs []string) error {
			affectedProjects = append([]string(nil), projectIDs...)
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Create() error=%v", err)
	}

	if created == nil {
		t.Fatal("Create() result=nil")
	}

	if created.BlockingTaskID != "task-a" {
		t.Fatalf(
			"BlockingTaskID=%q, want task-a",
			created.BlockingTaskID,
		)
	}

	if created.BlockedTaskID != "task-b" {
		t.Fatalf(
			"BlockedTaskID=%q, want task-b",
			created.BlockedTaskID,
		)
	}

	if len(affectedProjects) != 2 {
		t.Fatalf(
			"affectedProjects=%v, want [alpha beta]",
			affectedProjects,
		)
	}

	seen := map[string]bool{}
	for _, projectID := range affectedProjects {
		seen[projectID] = true
	}

	if !seen["alpha"] || !seen["beta"] {
		t.Fatalf(
			"affectedProjects=%v, want both alpha and beta",
			affectedProjects,
		)
	}

	alphaDetail, err := repo.List(context.Background(), "task-a")
	if err != nil {
		t.Fatalf("List(task-a) error=%v", err)
	}

	if len(alphaDetail.Blocks) != 1 {
		t.Fatalf(
			"task-a Blocks=%d, want 1",
			len(alphaDetail.Blocks),
		)
	}

	if alphaDetail.Blocks[0].Task.ID != "task-b" {
		t.Fatalf(
			"task-a Blocks[0].Task.ID=%q, want task-b",
			alphaDetail.Blocks[0].Task.ID,
		)
	}

	if alphaDetail.Blocks[0].Task.ProjectID != "beta" {
		t.Fatalf(
			"task-a Blocks[0].Task.ProjectID=%q, want beta",
			alphaDetail.Blocks[0].Task.ProjectID,
		)
	}

	betaDetail, err := repo.List(context.Background(), "task-b")
	if err != nil {
		t.Fatalf("List(task-b) error=%v", err)
	}

	if len(betaDetail.BlockedBy) != 1 {
		t.Fatalf(
			"task-b BlockedBy=%d, want 1",
			len(betaDetail.BlockedBy),
		)
	}

	if betaDetail.BlockedBy[0].Task.ID != "task-a" {
		t.Fatalf(
			"task-b BlockedBy[0].Task.ID=%q, want task-a",
			betaDetail.BlockedBy[0].Task.ID,
		)
	}

	if betaDetail.BlockedBy[0].Task.ProjectID != "alpha" {
		t.Fatalf(
			"task-b BlockedBy[0].Task.ProjectID=%q, want alpha",
			betaDetail.BlockedBy[0].Task.ProjectID,
		)
	}
}

func TestCandidatesSearchesTaskNameBySubstring(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)

	if err := db.Model(&taskModel{}).
		Where("id = ?", "c").
		Updates(map[string]any{
			"name":     "Sub Task Karton",
			"name_key": "sub task karton",
		}).Error; err != nil {
		t.Fatal(err)
	}

	page, err := repo.Candidates(
		context.Background(),
		"a",
		domain.Blocks,
		"task",
		1,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("items=%d, want 1", len(page.Items))
	}

	if page.Items[0].ID != "c" {
		t.Fatalf("id=%q, want c", page.Items[0].ID)
	}

	if page.Items[0].Name != "Sub Task Karton" {
		t.Fatalf("name=%q", page.Items[0].Name)
	}
}

func TestCandidatesReturnFullHierarchyPath(t *testing.T) {
	repo, db := testRepository(t)

	if err := db.Create(&projectModel{
		ID:                  "ntb",
		Name:                "NTB",
		NameKey:             "ntb",
		Status:              "open",
		AutomaticScheduling: true,
	}).Error; err != nil {
		t.Fatal(err)
	}

	parentID := "group-task-1"

	nodes := []taskModel{
		{
			ID:        parentID,
			ProjectID: "ntb",
			ParentKey: "",
			Name:      "Task 1",
			NameKey:   "task 1",
			Position:  1,
		},
		{
			ID:        "task-2-under-task-1",
			ProjectID: "ntb",
			ParentID:  &parentID,
			ParentKey: parentID,
			Name:      "Task 2",
			NameKey:   "task 2",
			Position:  1,
		},
		{
			ID:        "current",
			ProjectID: "ntb",
			ParentKey: "",
			Name:      "Current Task",
			NameKey:   "current task",
			Position:  2,
		},
	}

	if err := db.Create(&nodes).Error; err != nil {
		t.Fatal(err)
	}

	page, err := repo.Candidates(
		context.Background(),
		"current",
		domain.Blocks,
		"task 2",
		1,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("items=%d, want 1", len(page.Items))
	}

	if page.Items[0].HierarchyPath != "NTB > Task 1 > Task 2" {
		t.Fatalf(
			"hierarchyPath=%q, want %q",
			page.Items[0].HierarchyPath,
			"NTB > Task 1 > Task 2",
		)
	}
}

func TestCandidatesDisambiguateDuplicateTaskNamesWithHierarchyPath(
	t *testing.T,
) {
	repo, db := testRepository(t)

	if err := db.Create(&projectModel{
		ID:                  "ntb",
		Name:                "NTB",
		NameKey:             "ntb",
		Status:              "open",
		AutomaticScheduling: true,
	}).Error; err != nil {
		t.Fatal(err)
	}

	group1 := "group-1"
	group2 := "group-2"

	nodes := []taskModel{
		{
			ID:        group1,
			ProjectID: "ntb",
			ParentKey: "",
			Name:      "Task 1",
			NameKey:   "task 1",
			Position:  1,
		},
		{
			ID:        "task-2-under-1",
			ProjectID: "ntb",
			ParentID:  &group1,
			ParentKey: group1,
			Name:      "Task 2",
			NameKey:   "task 2",
			Position:  1,
		},
		{
			ID:        group2,
			ProjectID: "ntb",
			ParentKey: "",
			Name:      "Task 2",
			NameKey:   "task 2",
			Position:  2,
		},
		{
			ID:        "task-2-under-2",
			ProjectID: "ntb",
			ParentID:  &group2,
			ParentKey: group2,
			Name:      "Task 2",
			NameKey:   "task 2",
			Position:  1,
		},
		{
			ID:        "current",
			ProjectID: "ntb",
			ParentKey: "",
			Name:      "Current",
			NameKey:   "current",
			Position:  3,
		},
	}

	if err := db.Create(&nodes).Error; err != nil {
		t.Fatal(err)
	}

	page, err := repo.Candidates(
		context.Background(),
		"current",
		domain.Blocks,
		"task 2",
		1,
		5,
	)
	if err != nil {
		t.Fatal(err)
	}

	paths := make(map[string]bool)
	for _, item := range page.Items {
		paths[item.HierarchyPath] = true
	}

	if !paths["NTB > Task 1 > Task 2"] {
		t.Fatal("missing NTB > Task 1 > Task 2")
	}

	if !paths["NTB > Task 2 > Task 2"] {
		t.Fatal("missing NTB > Task 2 > Task 2")
	}
}

func TestCreateRejectsSummaryTaskAsEitherEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		blocking string
		blocked  string
	}{
		{
			name:     "summary task as blocker",
			blocking: "a",
			blocked:  "b",
		},
		{
			name:     "summary task as blocked task",
			blocking: "b",
			blocked:  "a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, db := testRepository(t)
			seedProjectAndTasks(t, db)

			parentID := "a"
			if err := db.Create(&taskModel{
				ID:        "a-child",
				ProjectID: "project",
				ParentID:  &parentID,
				ParentKey: parentID,
				Name:      "API Child",
				NameKey:   "api child",
				Position:  1,
			}).Error; err != nil {
				t.Fatal(err)
			}

			created, err := repo.Create(
				context.Background(),
				testDependency("summary-relation", tt.blocking, tt.blocked),
				func(context.Context, []string) error { return nil },
			)
			if created != nil {
				t.Fatalf("Create() result=%#v, want nil", created)
			}
			if !errors.Is(err, domain.ErrExecutableNeeded) {
				t.Fatalf("Create() error=%v, want ErrExecutableNeeded", err)
			}

			var count int64
			if err := db.Model(&dependencyModel{}).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("dependency count=%d, want 0", count)
			}
		})
	}
}

func TestCreateRejectsDirectCycleWithExistingPathOrder(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)
	callback := func(context.Context, []string) error { return nil }

	if _, err := repo.Create(
		context.Background(),
		testDependency("ab", "a", "b"),
		callback,
	); err != nil {
		t.Fatalf("Create(a -> b) error=%v", err)
	}

	created, err := repo.Create(
		context.Background(),
		testDependency("ba", "b", "a"),
		callback,
	)
	if created != nil {
		t.Fatalf("Create(b -> a) result=%#v, want nil", created)
	}

	var cycle *domain.CycleError
	if !errors.As(err, &cycle) {
		t.Fatalf("Create(b -> a) error=%v, want CycleError", err)
	}

	want := []string{"a", "b", "a"}
	if len(cycle.Path) != len(want) {
		t.Fatalf("cycle path length=%d, want %d: %#v", len(cycle.Path), len(want), cycle.Path)
	}
	for i, taskID := range want {
		if cycle.Path[i].TaskID != taskID {
			t.Fatalf("cycle.Path[%d].TaskID=%q, want %q; path=%#v", i, cycle.Path[i].TaskID, taskID, cycle.Path)
		}
	}

	var count int64
	if err := db.Model(&dependencyModel{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("dependency count=%d, want 1", count)
	}
}

func TestCreateAllowsCompletedTaskAsBlocker(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)
	actual := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)
	if err := db.Model(&taskModel{}).Where("id = ?", "a").Updates(map[string]any{"actual_start": actual, "actual_end": actual}).Error; err != nil {
		t.Fatal(err)
	}
	created, err := repo.Create(
		context.Background(),
		testDependency("completed-blocker", "a", "b"),
		func(context.Context, []string) error { return nil },
	)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created == nil {
		t.Fatal("Create() returned nil")
	}
	detail, err := repo.List(context.Background(), "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Blocks) != 1 {
		t.Fatalf("Blocks=%d, want 1", len(detail.Blocks))
	}
	if detail.Blocks[0].Task.ID != "b" {
		t.Fatalf("blocked task=%q, want b", detail.Blocks[0].Task.ID)
	}
	var actualEnd *time.Time
	if err := db.Model(&taskModel{}).Select("actual_end").Where("id = ?", "a").Scan(&actualEnd).Error; err != nil {
		t.Fatal(err)
	}
	if actualEnd == nil || !actualEnd.Equal(actual) {
		t.Fatal("actual_end changed")
	}
}

func TestCreateRejectsCompletedTaskAsBlockedTask(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)

	actual := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)

	if err := db.Model(&taskModel{}).
		Where("id = ?", "b").
		Updates(map[string]any{"actual_start": actual, "actual_end": actual}).Error; err != nil {
		t.Fatal(err)
	}

	created, err := repo.Create(
		context.Background(),
		testDependency("completed-blocked", "a", "b"),
		func(context.Context, []string) error { return nil },
	)

	if created != nil {
		t.Fatalf("Create() result=%#v, want nil", created)
	}

	if !errors.Is(err, domain.ErrCompletedBlocked) {
		t.Fatalf("Create() error=%v, want ErrCompletedBlocked", err)
	}

	var count int64
	if err := db.Model(&dependencyModel{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("dependency count=%d, want 0", count)
	}

	var actualEnd *time.Time
	if err := db.Model(&taskModel{}).
		Select("actual_end").
		Where("id = ?", "b").
		Scan(&actualEnd).Error; err != nil {
		t.Fatal(err)
	}

	if actualEnd == nil {
		t.Fatal("actual_end unexpectedly cleared")
	}

	if !actualEnd.Equal(actual) {
		t.Fatalf("actual_end changed: got %v want %v", actualEnd, actual)
	}
}

func TestCandidatesExcludeClosedProjectTasks(t *testing.T) {
	repo, db := testRepository(t)
	seedProjectAndTasks(t, db)

	closedProject := projectModel{
		ID:                  "closed-project",
		Name:                "Closed Project",
		NameKey:             "closed project",
		Status:              "closed",
		AutomaticScheduling: true,
	}

	if err := db.Create(&closedProject).Error; err != nil {
		t.Fatal(err)
	}

	tasks := []taskModel{
		{
			ID:        "closed-task",
			ProjectID: closedProject.ID,
			Name:      "Closed Task",
			NameKey:   "closed task",
			ParentKey: "",
			Position:  1,
		},
		{
			ID:        "open-task",
			ProjectID: "project",
			Name:      "Open Task",
			NameKey:   "open task",
			ParentKey: "",
			Position:  4,
		},
	}

	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}

	result, err := repo.Candidates(
		context.Background(),
		"a",
		domain.Blocks,
		"",
		1,
		20,
	)
	if err != nil {
		t.Fatalf("Candidates() error=%v", err)
	}

	openTaskFound := false

	for _, candidate := range result.Items {
		switch candidate.ID {
		case "closed-task":
			t.Fatal("closed project task returned as dependency candidate")
		case "open-task":
			openTaskFound = true
		}
	}

	if !openTaskFound {
		t.Fatal("open project task was not returned as dependency candidate")
	}
}

func TestCandidatesIncludeTasksFromAllActiveProjects(t *testing.T) {
	repo, db := testRepository(t)

	projects := []projectModel{
		{
			ID:                  "alpha",
			Name:                "Alpha",
			NameKey:             "alpha",
			Status:              "open",
			AutomaticScheduling: true,
		},
		{
			ID:                  "beta",
			Name:                "Beta",
			NameKey:             "beta",
			Status:              "open",
			AutomaticScheduling: true,
		},
		{
			ID:                  "gamma",
			Name:                "Gamma",
			NameKey:             "gamma",
			Status:              "open",
			AutomaticScheduling: true,
		},
	}

	if err := db.Create(&projects).Error; err != nil {
		t.Fatal(err)
	}

	actualEnd := time.Date(2026, 7, 27, 0, 0, 0, 0, time.UTC)

	tasks := []taskModel{
		{
			ID:        "current",
			ProjectID: "alpha",
			Name:      "Current Task",
			NameKey:   "current task",
			ParentKey: "",
			Position:  1,
		},
		{
			ID:          "beta-task",
			ProjectID:   "beta",
			Name:        "Beta Blocker",
			NameKey:     "beta blocker",
			ParentKey:   "",
			Position:    1,
			ActualStart: &actualEnd,
			ActualEnd:   &actualEnd,
		},
		{
			ID:        "gamma-task",
			ProjectID: "gamma",
			Name:      "Gamma Blocker",
			NameKey:   "gamma blocker",
			ParentKey: "",
			Position:  1,
		},
	}

	if err := db.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}

	page, err := repo.Candidates(
		context.Background(),
		"current",
		domain.BlockedBy,
		"",
		1,
		20,
	)
	if err != nil {
		t.Fatalf("Candidates() error=%v", err)
	}

	itemsByID := make(map[string]domain.Task, len(page.Items))
	for _, item := range page.Items {
		itemsByID[item.ID] = item
	}

	beta, exists := itemsByID["beta-task"]
	if !exists {
		t.Fatal("completed Task from active Beta Project was not returned")
	}

	if beta.ProjectID != "beta" {
		t.Fatalf("beta ProjectID=%q, want beta", beta.ProjectID)
	}

	if beta.HierarchyPath != "Beta > Beta Blocker" {
		t.Fatalf(
			"beta HierarchyPath=%q, want %q",
			beta.HierarchyPath,
			"Beta > Beta Blocker",
		)
	}

	if beta.ActualEnd == nil || !beta.ActualEnd.Equal(actualEnd) {
		t.Fatalf("beta ActualEnd=%v, want %v", beta.ActualEnd, actualEnd)
	}

	gamma, exists := itemsByID["gamma-task"]
	if !exists {
		t.Fatal("Task from active Gamma Project was not returned")
	}

	if gamma.ProjectID != "gamma" {
		t.Fatalf("gamma ProjectID=%q, want gamma", gamma.ProjectID)
	}

	if gamma.HierarchyPath != "Gamma > Gamma Blocker" {
		t.Fatalf(
			"gamma HierarchyPath=%q, want %q",
			gamma.HierarchyPath,
			"Gamma > Gamma Blocker",
		)
	}
}

func TestOwnershipCommandsPreserveOneEndpointAndRollbackWithScheduler_AC23_AC24_AC25_AC35(t *testing.T) {
	t.Run("manual create upgrades automatic endpoint and rolls back on scheduling failure", func(t *testing.T) {
		repo, db := testRepository(t)
		seedProjectAndTasks(t, db)
		now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
		if err := db.Create(&dependencyModel{
			ID: "automatic", BlockingTaskID: "a", BlockedTaskID: "b",
			AutomaticOwned: true, CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			t.Fatal(err)
		}

		schedulerFailure := errors.New("scheduler failed")
		_, err := repo.Create(
			context.Background(),
			testDependency("ignored-new-id", "a", "b"),
			func(context.Context, []string) error { return schedulerFailure },
		)
		if !errors.Is(err, schedulerFailure) {
			t.Fatalf("Create() error=%v, want scheduler failure", err)
		}
		var afterFailure dependencyModel
		if err := db.First(&afterFailure, "id = ?", "automatic").Error; err != nil {
			t.Fatal(err)
		}
		if afterFailure.ManualOwned || !afterFailure.AutomaticOwned {
			t.Fatalf("ownership after rollback = manual:%v automatic:%v", afterFailure.ManualOwned, afterFailure.AutomaticOwned)
		}

		created, err := repo.Create(
			context.Background(),
			testDependency("ignored-new-id", "a", "b"),
			func(context.Context, []string) error { return nil },
		)
		if err != nil {
			t.Fatal(err)
		}
		if created == nil || created.ID != "automatic" || created.Source() != domain.SourceShared {
			t.Fatalf("shared result=%#v", created)
		}
		var rows []dependencyModel
		if err := db.Where("blocking_task_id = ? AND blocked_task_id = ?", "a", "b").Find(&rows).Error; err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || !rows[0].ManualOwned || !rows[0].AutomaticOwned {
			t.Fatalf("endpoint rows=%#v, want one shared row", rows)
		}
	})

	t.Run("shared delete removes manual ownership only and automatic-only remains protected", func(t *testing.T) {
		repo, db := testRepository(t)
		seedProjectAndTasks(t, db)
		now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
		if err := db.Create(&dependencyModel{
			ID: "shared", BlockingTaskID: "a", BlockedTaskID: "b",
			ManualOwned: true, AutomaticOwned: true, CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			t.Fatal(err)
		}

		schedulerFailure := errors.New("scheduler failed")
		if err := repo.Delete(context.Background(), "shared", now.Add(time.Hour), func(context.Context, []string) error {
			return schedulerFailure
		}); !errors.Is(err, schedulerFailure) {
			t.Fatalf("Delete() error=%v, want scheduler failure", err)
		}
		var afterFailure dependencyModel
		if err := db.First(&afterFailure, "id = ?", "shared").Error; err != nil {
			t.Fatal(err)
		}
		if !afterFailure.ManualOwned || !afterFailure.AutomaticOwned {
			t.Fatalf("rollback ownership = %#v", afterFailure)
		}

		if err := repo.Delete(context.Background(), "shared", now.Add(2*time.Hour), func(context.Context, []string) error { return nil }); err != nil {
			t.Fatal(err)
		}
		var automaticOnly dependencyModel
		if err := db.First(&automaticOnly, "id = ?", "shared").Error; err != nil {
			t.Fatal(err)
		}
		if automaticOnly.ManualOwned || !automaticOnly.AutomaticOwned {
			t.Fatalf("ownership after manual removal = %#v", automaticOnly)
		}
		if err := repo.Delete(context.Background(), "shared", now.Add(3*time.Hour), func(context.Context, []string) error { return nil }); !errors.Is(err, domain.ErrAutomaticOnlyReadOnly) {
			t.Fatalf("automatic-only Delete() error=%v", err)
		}
		var count int64
		if err := db.Model(&dependencyModel{}).Where("id = ?", "shared").Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("automatic endpoint count=%d, want 1", count)
		}
	})

	t.Run("keep as manual preserves automatic ownership and rolls back with scheduler", func(t *testing.T) {
		repo, db := testRepository(t)
		seedProjectAndTasks(t, db)
		now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
		if err := db.Create(&dependencyModel{
			ID: "automatic", BlockingTaskID: "a", BlockedTaskID: "b",
			AutomaticOwned: true, CreatedAt: now, UpdatedAt: now,
		}).Error; err != nil {
			t.Fatal(err)
		}

		schedulerFailure := errors.New("scheduler failed")
		_, err := repo.KeepAsManual(context.Background(), "automatic", now.Add(time.Hour), func(context.Context, []string) error {
			return schedulerFailure
		})
		if !errors.Is(err, schedulerFailure) {
			t.Fatalf("KeepAsManual() error=%v, want scheduler failure", err)
		}
		var afterFailure dependencyModel
		if err := db.First(&afterFailure, "id = ?", "automatic").Error; err != nil {
			t.Fatal(err)
		}
		if afterFailure.ManualOwned || !afterFailure.AutomaticOwned {
			t.Fatalf("rollback ownership = %#v", afterFailure)
		}

		kept, err := repo.KeepAsManual(context.Background(), "automatic", now.Add(2*time.Hour), func(context.Context, []string) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		if kept == nil || kept.Source() != domain.SourceShared {
			t.Fatalf("kept dependency=%#v", kept)
		}
		var stored dependencyModel
		if err := db.First(&stored, "id = ?", "automatic").Error; err != nil {
			t.Fatal(err)
		}
		if !stored.ManualOwned || !stored.AutomaticOwned {
			t.Fatalf("stored ownership=%#v", stored)
		}
	})
}
