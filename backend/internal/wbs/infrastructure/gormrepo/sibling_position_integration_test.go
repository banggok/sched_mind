package gormrepo

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
	"gorm.io/gorm"
)

func TestCreateSiblingInsertsAfterAuthoritativeAnchorAndKeepsGroupSubtree_DeltaD03_D07(t *testing.T) {
	repository, database := dependencyTestDB(t)
	now := time.Date(2026, 8, 6, 6, 0, 0, 0, time.UTC)
	if err := database.Create(&projectModel{ID: "project", Status: "open", AutomaticScheduling: true}).Error; err != nil {
		t.Fatal(err)
	}
	groupID := "group"
	nodes := []nodeModel{
		{ID: groupID, ProjectID: "project", ParentKey: "", Name: "Group", NameKey: "group", Position: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "later", ProjectID: "project", ParentKey: "", Name: "Later", NameKey: "later", Position: 2, CreatedAt: now, UpdatedAt: now},
		{ID: "child", ProjectID: "project", ParentID: &groupID, ParentKey: groupID, Name: "Child", NameKey: "child", Position: 1, CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&nodes).Error; err != nil {
		t.Fatal(err)
	}
	scheduleCalls := 0

	created, err := repository.CreateSibling(context.Background(), "new", "project", groupID, "New sibling", now, func(context.Context, string) error {
		scheduleCalls++
		return errors.New("name-only sibling create must not schedule")
	})
	if err != nil {
		t.Fatal(err)
	}
	if created == nil || created.ID != "new" || created.ParentID != nil || created.Position != 2 || scheduleCalls != 0 {
		t.Fatalf("created=%#v schedule calls=%d", created, scheduleCalls)
	}
	if got := siblingIDs(t, database, "project", ""); !reflect.DeepEqual(got, []string{"group", "new", "later"}) {
		t.Fatalf("root order=%v", got)
	}
	if got := siblingIDs(t, database, "project", groupID); !reflect.DeepEqual(got, []string{"child"}) {
		t.Fatalf("group children=%v", got)
	}
	var child nodeModel
	if err := database.First(&child, "id = ?", "child").Error; err != nil {
		t.Fatal(err)
	}
	if child.ParentID == nil || *child.ParentID != groupID || child.Position != 1 {
		t.Fatalf("child hierarchy changed: %#v", child)
	}
}

func TestCreateSiblingStaleAnchorRollsBackWithoutPositionChanges_DeltaD03_D09(t *testing.T) {
	repository, database := dependencyTestDB(t)
	now := time.Date(2026, 8, 6, 6, 5, 0, 0, time.UTC)
	if err := database.Create(&projectModel{ID: "project", Status: "open"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&[]nodeModel{
		{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	created, err := repository.CreateSibling(context.Background(), "new", "project", "missing", "New", now, func(context.Context, string) error { return nil })
	if created != nil || !errors.Is(err, domain.ErrCreateAnchorConflict) {
		t.Fatalf("created=%#v err=%v", created, err)
	}
	if got := siblingIDs(t, database, "project", ""); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("order changed=%v", got)
	}
	var count int64
	if err := database.Model(&nodeModel{}).Where("id = ?", "new").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("created row count=%d", count)
	}
}

func TestPlaceMovesNonAdjacentGroupAsOneSiblingAndPreservesDescendants_DeltaD05_D07(t *testing.T) {
	repository, database := dependencyTestDB(t)
	now := time.Date(2026, 8, 6, 6, 10, 0, 0, time.UTC)
	if err := database.Create(&projectModel{ID: "project", Status: "open", AutomaticScheduling: true}).Error; err != nil {
		t.Fatal(err)
	}
	groupID := "group"
	if err := database.Create(&[]nodeModel{
		{ID: groupID, ProjectID: "project", ParentKey: "", Name: "Group", NameKey: "group", Position: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "middle", ProjectID: "project", ParentKey: "", Name: "Middle", NameKey: "middle", Position: 2, CreatedAt: now, UpdatedAt: now},
		{ID: "last", ProjectID: "project", ParentKey: "", Name: "Last", NameKey: "last", Position: 3, CreatedAt: now, UpdatedAt: now},
		{ID: "child-a", ProjectID: "project", ParentID: &groupID, ParentKey: groupID, Name: "Child A", NameKey: "child a", Position: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "child-b", ProjectID: "project", ParentID: &groupID, ParentKey: groupID, Name: "Child B", NameKey: "child b", Position: 2, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	scheduleCalls := 0

	if err := repository.Place(context.Background(), "project", groupID, "last", domain.PlaceAfter, now.Add(time.Minute), func(_ context.Context, projectID string) error {
		scheduleCalls++
		if projectID != "project" {
			t.Fatalf("scheduled project=%q", projectID)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if scheduleCalls != 1 {
		t.Fatalf("schedule calls=%d", scheduleCalls)
	}
	if got := siblingIDs(t, database, "project", ""); !reflect.DeepEqual(got, []string{"middle", "last", "group"}) {
		t.Fatalf("root order=%v", got)
	}
	if got := siblingIDs(t, database, "project", groupID); !reflect.DeepEqual(got, []string{"child-a", "child-b"}) {
		t.Fatalf("child order=%v", got)
	}
}

func TestPlaceNoOpInvalidTargetAndSchedulerFailurePreserveConfirmedOrder_DeltaD05_D09(t *testing.T) {
	tests := []struct {
		name      string
		targetID  string
		placement domain.Placement
		schedule  func(context.Context, string) error
		wantErr   error
		wantCalls int
	}{
		{name: "existing position", targetID: "b", placement: domain.PlaceBefore, schedule: func(context.Context, string) error { return errors.New("must not schedule") }, wantCalls: 0},
		{name: "cross parent", targetID: "child", placement: domain.PlaceBefore, schedule: func(context.Context, string) error { return nil }, wantErr: domain.ErrReorderTargetInvalid, wantCalls: 0},
		{name: "scheduler rollback", targetID: "c", placement: domain.PlaceAfter, schedule: func(context.Context, string) error { return errors.New("schedule failed") }, wantErr: errors.New("schedule failed"), wantCalls: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, database := dependencyTestDB(t)
			now := time.Date(2026, 8, 6, 6, 15, 0, 0, time.UTC)
			if err := database.Create(&projectModel{ID: "project", Status: "open", AutomaticScheduling: true}).Error; err != nil {
				t.Fatal(err)
			}
			parentID := "parent"
			if err := database.Create(&[]nodeModel{
				{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1, CreatedAt: now, UpdatedAt: now},
				{ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2, CreatedAt: now, UpdatedAt: now},
				{ID: "c", ProjectID: "project", ParentKey: "", Name: "C", NameKey: "c", Position: 3, CreatedAt: now, UpdatedAt: now},
				{ID: parentID, ProjectID: "project", ParentKey: "", Name: "Parent", NameKey: "parent", Position: 4, CreatedAt: now, UpdatedAt: now},
				{ID: "child", ProjectID: "project", ParentID: &parentID, ParentKey: parentID, Name: "Child", NameKey: "child", Position: 1, CreatedAt: now, UpdatedAt: now},
			}).Error; err != nil {
				t.Fatal(err)
			}
			calls := 0
			err := repository.Place(context.Background(), "project", "a", test.targetID, test.placement, now.Add(time.Minute), func(ctx context.Context, projectID string) error {
				calls++
				return test.schedule(ctx, projectID)
			})
			if test.wantErr == nil {
				if err != nil {
					t.Fatal(err)
				}
			} else if test.name == "scheduler rollback" {
				if err == nil || err.Error() != test.wantErr.Error() {
					t.Fatalf("err=%v", err)
				}
			} else if !errors.Is(err, test.wantErr) {
				t.Fatalf("err=%v", err)
			}
			if calls != test.wantCalls {
				t.Fatalf("schedule calls=%d want=%d", calls, test.wantCalls)
			}
			if got := siblingIDs(t, database, "project", ""); !reflect.DeepEqual(got, []string{"a", "b", "c", "parent"}) {
				t.Fatalf("confirmed order changed=%v", got)
			}
		})
	}
}

func TestConcurrentSiblingCreateAndPlacementPreserveOneGapFreeOrder_DeltaD03_D05_D09(t *testing.T) {
	repository, database := dependencyTestDB(t)
	now := time.Date(2026, 8, 6, 6, 20, 0, 0, time.UTC)
	if err := database.Create(&projectModel{ID: "project", Status: "open"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&[]nodeModel{
		{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2, CreatedAt: now, UpdatedAt: now},
		{ID: "c", ProjectID: "project", ParentKey: "", Name: "C", NameKey: "c", Position: 3, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errorsByCommand := make(chan error, 2)
	var wait sync.WaitGroup
	wait.Add(2)
	go func() {
		defer wait.Done()
		<-start
		_, err := repository.CreateSibling(context.Background(), "new", "project", "a", "New", now.Add(time.Minute), func(context.Context, string) error { return nil })
		errorsByCommand <- err
	}()
	go func() {
		defer wait.Done()
		<-start
		errorsByCommand <- repository.Place(context.Background(), "project", "c", "a", domain.PlaceBefore, now.Add(2*time.Minute), func(context.Context, string) error { return nil })
	}()
	close(start)
	wait.Wait()
	close(errorsByCommand)
	for err := range errorsByCommand {
		if err != nil {
			t.Fatal(err)
		}
	}

	if got := siblingIDs(t, database, "project", ""); !reflect.DeepEqual(got, []string{"c", "a", "new", "b"}) {
		t.Fatalf("concurrent order=%v", got)
	}
}

func siblingIDs(t *testing.T, database *gorm.DB, projectID, parentKey string) []string {
	t.Helper()
	var siblings []nodeModel
	if err := database.Where("project_id = ? AND parent_key = ?", projectID, parentKey).Order("position ASC").Order("id ASC").Find(&siblings).Error; err != nil {
		t.Fatal(err)
	}
	ids := make([]string, len(siblings))
	for index := range siblings {
		ids[index] = siblings[index].ID
		if siblings[index].Position != index+1 {
			t.Fatalf("sibling %q position=%d want=%d", siblings[index].ID, siblings[index].Position, index+1)
		}
	}
	return ids
}

func TestCreateSiblingAfterCompletedTaskPreservesCompletedIdentityAndActualDates_DeltaD03(t *testing.T) {
	repository, database := dependencyTestDB(t)
	now := time.Date(2026, 8, 6, 7, 20, 0, 0, time.UTC)
	actualStart := now.Add(-48 * time.Hour)
	actualEnd := now.Add(-24 * time.Hour)
	effort := 480
	if err := database.Create(&projectModel{ID: "project", Status: "open", AutomaticScheduling: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&[]nodeModel{
		{ID: "completed", ProjectID: "project", ParentKey: "", Name: "Completed", NameKey: "completed", Position: 1, EffortMinutes: &effort, ActualStart: &actualStart, ActualEnd: &actualEnd, CreatedAt: now, UpdatedAt: now},
		{ID: "later", ProjectID: "project", ParentKey: "", Name: "Later", NameKey: "later", Position: 2, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	scheduleCalls := 0

	created, err := repository.CreateSibling(context.Background(), "new", "project", "completed", "New sibling", now.Add(time.Hour), func(context.Context, string) error {
		scheduleCalls++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if created == nil || created.ID != "new" || created.ParentID != nil || created.Position != 2 || scheduleCalls != 0 {
		t.Fatalf("created=%#v schedule calls=%d", created, scheduleCalls)
	}
	if got := siblingIDs(t, database, "project", ""); !reflect.DeepEqual(got, []string{"completed", "new", "later"}) {
		t.Fatalf("root order=%v", got)
	}
	var completed nodeModel
	if err := database.First(&completed, "id = ?", "completed").Error; err != nil {
		t.Fatal(err)
	}
	if completed.EffortMinutes == nil || *completed.EffortMinutes != effort || completed.ActualStart == nil || !completed.ActualStart.Equal(actualStart) || completed.ActualEnd == nil || !completed.ActualEnd.Equal(actualEnd) {
		t.Fatalf("completed anchor changed=%#v", completed)
	}
}

func TestCreateSiblingNameConflictRollsBackShiftedPositions_DeltaD03_D09(t *testing.T) {
	repository, database := dependencyTestDB(t)
	now := time.Date(2026, 8, 6, 7, 25, 0, 0, time.UTC)
	if err := database.Exec("CREATE UNIQUE INDEX wbs_test_sibling_name_unique ON wbs_nodes(project_id, parent_key, name_key)").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&projectModel{ID: "project", Status: "open"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&[]nodeModel{
		{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}

	created, err := repository.CreateSibling(context.Background(), "new", "project", "a", "B", now.Add(time.Hour), func(context.Context, string) error { return nil })
	if created != nil || err == nil {
		t.Fatalf("created=%#v err=%v", created, err)
	}
	if got := siblingIDs(t, database, "project", ""); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("confirmed order changed=%v", got)
	}
	var count int64
	if err := database.Model(&nodeModel{}).Where("id = ?", "new").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("created row count=%d", count)
	}
}

func TestPlaceMissingSourceIsRecoverableConflictWithoutMutation_DeltaD05_D09(t *testing.T) {
	repository, database := dependencyTestDB(t)
	now := time.Date(2026, 8, 6, 7, 30, 0, 0, time.UTC)
	if err := database.Create(&projectModel{ID: "project", Status: "open", AutomaticScheduling: true}).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Create(&[]nodeModel{
		{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1, CreatedAt: now, UpdatedAt: now},
		{ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2, CreatedAt: now, UpdatedAt: now},
	}).Error; err != nil {
		t.Fatal(err)
	}
	scheduleCalls := 0

	err := repository.Place(context.Background(), "project", "missing", "b", domain.PlaceBefore, now.Add(time.Minute), func(context.Context, string) error {
		scheduleCalls++
		return nil
	})
	if !errors.Is(err, domain.ErrReorderTargetInvalid) {
		t.Fatalf("err=%v", err)
	}
	if scheduleCalls != 0 {
		t.Fatalf("schedule calls=%d", scheduleCalls)
	}
	if got := siblingIDs(t, database, "project", ""); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("confirmed order changed=%v", got)
	}
}

func TestStructuralSiblingMutationsRespectLockedAndClosedProjectBoundaries_DeltaD02_D05_D08(t *testing.T) {
	tests := []struct {
		status  string
		wantErr error
	}{
		{status: "locked", wantErr: domain.ErrProjectLockedReadOnly},
		{status: "closed", wantErr: domain.ErrProjectClosedReadOnly},
	}
	for _, test := range tests {
		t.Run(test.status, func(t *testing.T) {
			repository, database := dependencyTestDB(t)
			now := time.Date(2026, 8, 6, 7, 35, 0, 0, time.UTC)
			if err := database.Create(&projectModel{ID: "project", Status: test.status, AutomaticScheduling: true}).Error; err != nil {
				t.Fatal(err)
			}
			if err := database.Create(&[]nodeModel{
				{ID: "a", ProjectID: "project", ParentKey: "", Name: "A", NameKey: "a", Position: 1, CreatedAt: now, UpdatedAt: now},
				{ID: "b", ProjectID: "project", ParentKey: "", Name: "B", NameKey: "b", Position: 2, CreatedAt: now, UpdatedAt: now},
			}).Error; err != nil {
				t.Fatal(err)
			}
			scheduleCalls := 0
			schedule := func(context.Context, string) error {
				scheduleCalls++
				return nil
			}

			created, createErr := repository.CreateSibling(context.Background(), "new", "project", "a", "New", now.Add(time.Minute), schedule)
			if created != nil || !errors.Is(createErr, test.wantErr) {
				t.Fatalf("created=%#v create err=%v", created, createErr)
			}
			placeErr := repository.Place(context.Background(), "project", "a", "b", domain.PlaceAfter, now.Add(2*time.Minute), schedule)
			if !errors.Is(placeErr, test.wantErr) {
				t.Fatalf("place err=%v", placeErr)
			}
			if scheduleCalls != 0 {
				t.Fatalf("schedule calls=%d", scheduleCalls)
			}
			if got := siblingIDs(t, database, "project", ""); !reflect.DeepEqual(got, []string{"a", "b"}) {
				t.Fatalf("confirmed order changed=%v", got)
			}
			var newCount int64
			if err := database.Model(&nodeModel{}).Where("id = ?", "new").Count(&newCount).Error; err != nil {
				t.Fatal(err)
			}
			if newCount != 0 {
				t.Fatalf("created row count=%d", newCount)
			}
		})
	}
}
