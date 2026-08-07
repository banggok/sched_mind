package gormrepo

import (
	"context"
	"errors"
	"testing"
	"time"

	sharedpersistence "github.com/banggok/sched_mind/backend/internal/shared/persistence"
	"github.com/banggok/sched_mind/backend/internal/shared/schedulingimpact"
	"github.com/banggok/sched_mind/backend/internal/wbs/application"
	"github.com/banggok/sched_mind/backend/internal/wbs/domain"
	"gorm.io/gorm"
)

type groupSchedulingProjectColumns struct {
	ID                  string `gorm:"primaryKey"`
	SchedulingStartDate *time.Time
	ProjectBuffer       int
	Priority            int
	ScheduleVersion     int64
	UpdatedAt           time.Time
}

func (groupSchedulingProjectColumns) TableName() string { return "projects" }

func TestUpdateGroupSchedulingPersistsRepresentationWithoutSchedulingWhenEffectiveValuesDoNotChange_US44_AC1_AC5_AC8(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	anchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &anchor)
	seedGroupWithTask(t, database, nodeModel{}, nodeModel{})

	scheduleCalls := 0
	automatic := true
	confirmed, err := repository.UpdateGroupScheduling(
		context.Background(),
		"project",
		"group",
		application.GroupSchedulingInput{
			Source:              "override",
			AutomaticScheduling: &automatic,
			SchedulingStartDate: &anchor,
		},
		time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
		func(context.Context, string) error {
			scheduleCalls++
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if scheduleCalls != 0 {
		t.Fatalf("schedule calls=%d, want 0 for representation-only change", scheduleCalls)
	}
	if confirmed.Scheduling.Source != "override" || confirmed.Scheduling.AutomaticScheduling == nil || !*confirmed.Scheduling.AutomaticScheduling {
		t.Fatalf("confirmed scheduling=%+v, want persisted ON override", confirmed.Scheduling)
	}
	if confirmed.Scheduling.SchedulingStartDate == nil || !confirmed.Scheduling.SchedulingStartDate.Equal(anchor) {
		t.Fatalf("confirmed start=%v, want %v", confirmed.Scheduling.SchedulingStartDate, anchor)
	}

	stored := loadNodeModel(t, database, "group")
	if stored.GroupSchedulingSource != "override" || stored.GroupAutomaticScheduling == nil || !*stored.GroupAutomaticScheduling {
		t.Fatalf("stored Group scheduling=%#v", stored)
	}
}

func TestGroupRenameAdvancesSchedulingVersionSoUnifiedSaveDetectsStaleDraft_US44_AC37(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	anchorDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &anchorDate)
	seedGroupWithTask(t, database, nodeModel{}, nodeModel{})

	renamed, err := repository.Rename(
		context.Background(),
		"project",
		"group",
		"Platform Renamed",
		time.Date(2026, 8, 7, 11, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Scheduling.Version != 1 {
		t.Fatalf("version=%d, want 1 after Group rename", renamed.Scheduling.Version)
	}

	staleName := "Stale Draft"
	automatic := false
	_, err = repository.UpdateGroupScheduling(
		context.Background(),
		"project",
		"group",
		application.GroupSchedulingInput{
			ExpectedVersion:     0,
			Name:                &staleName,
			Source:              "override",
			AutomaticScheduling: &automatic,
		},
		time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
		func(context.Context, string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupSchedulingStale) {
		t.Fatalf("stale unified Save err=%v, want GROUP_SCHEDULING_STALE", err)
	}
	stored := loadNodeModel(t, database, "group")
	if stored.Name != "Platform Renamed" || stored.GroupSchedulingVersion != 1 || stored.GroupSchedulingSource != "inherit" {
		t.Fatalf("stale unified Save overwrote newer Group rename: %#v", stored)
	}
}

func TestUpdateGroupSchedulingRollsBackNameAndSchedulingTogetherWhenSchedulingFails_US44_AC35_AC37(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	anchorDate := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &anchorDate)
	seedGroupWithTask(t, database, nodeModel{}, nodeModel{})

	updatedName := "Platform Stream"
	automatic := false
	scheduleErr := errors.New("schedule failed")
	_, err := repository.UpdateGroupScheduling(
		context.Background(),
		"project",
		"group",
		application.GroupSchedulingInput{
			ExpectedVersion:     0,
			Name:                &updatedName,
			Source:              "override",
			AutomaticScheduling: &automatic,
		},
		time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
		func(context.Context, string) error { return scheduleErr },
	)
	if !errors.Is(err, scheduleErr) {
		t.Fatalf("err=%v, want schedule failure", err)
	}

	stored := loadNodeModel(t, database, "group")
	if stored.Name != "Platform" || stored.GroupSchedulingSource != "inherit" || stored.GroupSchedulingVersion != 0 {
		t.Fatalf("failed unified Save partially persisted Group state: %#v", stored)
	}
	if stored.GroupAutomaticScheduling != nil || stored.GroupSchedulingStartDate != nil {
		t.Fatalf("failed unified Save leaked Group overrides: %#v", stored)
	}
}

func TestGroupSchedulingRejectsStaleConfigAndLifecycleVersionsAtomically_US44_AC37(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	anchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &anchor)
	start := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	seedGroupWithTask(t, database, nodeModel{}, nodeModel{ExecutionStart: &start, ExecutionEnd: &end, CommitmentStart: &start, CommitmentEnd: &end})

	automatic := false
	updated, err := repository.UpdateGroupScheduling(
		context.Background(), "project", "group",
		application.GroupSchedulingInput{ExpectedVersion: 0, Source: "override", AutomaticScheduling: &automatic},
		time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC),
		func(context.Context, string) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Scheduling.Version != 1 {
		t.Fatalf("version=%d, want 1 after config update", updated.Scheduling.Version)
	}
	staleAutomatic := true
	_, err = repository.UpdateGroupScheduling(
		context.Background(), "project", "group",
		application.GroupSchedulingInput{ExpectedVersion: 0, Source: "override", AutomaticScheduling: &staleAutomatic},
		time.Date(2026, 8, 7, 11, 0, 0, 0, time.UTC),
		func(context.Context, string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupSchedulingStale) {
		t.Fatalf("stale config err=%v, want GROUP_SCHEDULING_STALE", err)
	}
	stored := loadNodeModel(t, database, "group")
	if stored.GroupSchedulingVersion != 1 || stored.GroupAutomaticScheduling == nil || *stored.GroupAutomaticScheduling {
		t.Fatalf("stale config overwrote confirmed state: %#v", stored)
	}

	locked, err := repository.ChangeGroupStatus(
		context.Background(), "project", "group", "locked", 1,
		time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
		func(context.Context, string) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if locked.Scheduling.Version != 2 {
		t.Fatalf("version=%d, want 2 after Lock", locked.Scheduling.Version)
	}
	_, err = repository.ChangeGroupStatus(
		context.Background(), "project", "group", "open", 1,
		time.Date(2026, 8, 7, 13, 0, 0, 0, time.UTC),
		func(context.Context, string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupSchedulingStale) {
		t.Fatalf("stale lifecycle err=%v, want GROUP_SCHEDULING_STALE", err)
	}
	stored = loadNodeModel(t, database, "group")
	if stored.GroupSchedulingVersion != 2 || stored.GroupLocalStatus != "locked" {
		t.Fatalf("stale lifecycle changed confirmed state: %#v", stored)
	}
}

func TestGroupReopenPreviewPropagatesScheduleMutationSerialization_US44_AC18_AC25(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	anchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &anchor)
	seedGroupWithTask(t, database, nodeModel{
		GroupLocalStatus:               "locked",
		GroupLockedAutomaticScheduling: groupSchedulingBool(true),
		GroupLockedSchedulingStartDate: &anchor,
	}, nodeModel{})

	scheduleCalls := 0
	reopened, err := repository.ChangeGroupStatus(
		context.Background(),
		"project",
		"group",
		"open",
		0,
		time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
		func(ctx context.Context, _ string) error {
			if !sharedpersistence.ScheduleMutationSerialized(ctx) {
				return errors.New("Group reopen scheduler callback lost schedule-mutation serialization marker")
			}
			scheduleCalls++
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if scheduleCalls != 2 {
		t.Fatalf("schedule calls=%d, want preview + confirmed recalculation", scheduleCalls)
	}
	if reopened == nil || reopened.Scheduling.LocalStatus != "open" {
		t.Fatalf("reopened Group=%#v, want local open", reopened)
	}
}

func TestGroupLockFreezesEffectiveConfigAndActualDateKeepsProtectedBaseline_US44_AC13_AC14_AC17_AC20(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	anchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &anchor)
	start := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 8, 12, 0, 0, 0, 0, time.UTC)
	seedGroupWithTask(t, database, nodeModel{}, nodeModel{
		ExecutionStart:  &start,
		ExecutionEnd:    &end,
		CommitmentStart: &start,
		CommitmentEnd:   &end,
	})

	scheduleCalls := 0
	locked, err := repository.ChangeGroupStatus(
		context.Background(),
		"project",
		"group",
		"locked",
		0,
		time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC),
		func(context.Context, string) error {
			scheduleCalls++
			return nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if scheduleCalls != 0 {
		t.Fatalf("Group Lock scheduled %d times, want 0", scheduleCalls)
	}
	if locked.Scheduling.LocalStatus != "locked" || locked.Scheduling.EffectiveLifecycle != "locked" {
		t.Fatalf("locked scheduling=%+v", locked.Scheduling)
	}

	newAnchor := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	if err := database.Model(&groupSchedulingProjectColumns{}).Where("id = ?", "project").Updates(map[string]any{
		"scheduling_start_date": newAnchor,
		"automatic_scheduling":  false,
	}).Error; err != nil {
		t.Fatal(err)
	}
	confirmed, err := repository.Find(context.Background(), "project", "task")
	if err != nil {
		t.Fatal(err)
	}
	if !confirmed.Scheduling.EffectiveAutomatic || confirmed.Scheduling.EffectiveStartDate == nil || !confirmed.Scheduling.EffectiveStartDate.Equal(anchor) {
		t.Fatalf("locked Task effective config=%+v, want frozen ON/%v", confirmed.Scheduling, anchor)
	}

	actualStart := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	actualEnd := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	if _, err := repository.Complete(
		context.Background(),
		"project",
		"task",
		actualStart,
		actualEnd,
		time.Date(2026, 8, 13, 12, 0, 0, 0, time.UTC),
		func(context.Context, []string) error { return nil },
	); err != nil {
		t.Fatal(err)
	}
	after := loadNodeModel(t, database, "task")
	if after.ActualStart == nil || after.ActualEnd == nil {
		t.Fatalf("actual dates were not persisted: %#v", after)
	}
	if after.ExecutionStart == nil || !after.ExecutionStart.Equal(start) || after.ExecutionEnd == nil || !after.ExecutionEnd.Equal(end) || after.CommitmentStart == nil || !after.CommitmentStart.Equal(start) || after.CommitmentEnd == nil || !after.CommitmentEnd.Equal(end) {
		t.Fatalf("protected baseline changed after Actual Date: %#v", after)
	}
}

func TestGroupLockRejectsUnscheduledDescendantAtomically_US44_AC13(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	seedGroupSchedulingProject(t, database, true, nil)
	seedGroupWithTask(t, database, nodeModel{}, nodeModel{})

	_, err := repository.ChangeGroupStatus(
		context.Background(),
		"project",
		"group",
		"locked",
		0,
		time.Now().UTC(),
		func(context.Context, string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupCannotLockUnscheduled) {
		t.Fatalf("err=%v, want GROUP_CANNOT_LOCK_WITH_UNSCHEDULED_TASKS", err)
	}
	stored := loadNodeModel(t, database, "group")
	if stored.GroupLocalStatus != "open" || stored.GroupLockedAutomaticScheduling != nil || stored.GroupLockedSchedulingStartDate != nil {
		t.Fatalf("failed Lock changed Group state: %#v", stored)
	}
}

func TestDeletingFinalChildOfCustomGroupRequiresResetAndPreservesHierarchy_US44_AC31(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	seedGroupSchedulingProject(t, database, true, nil)
	automatic := true
	seedGroupWithTask(t, database, nodeModel{
		GroupSchedulingSource:    "override",
		GroupAutomaticScheduling: &automatic,
	}, nodeModel{})

	err := repository.Delete(
		context.Background(),
		"project",
		"task",
		time.Now().UTC(),
		func(context.Context, string) error { return nil },
		func(context.Context, []string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupOverrideMustReset) {
		t.Fatalf("err=%v, want GROUP_OVERRIDE_MUST_BE_RESET_BEFORE_TASK_CONVERSION", err)
	}
	if _, err := repository.Find(context.Background(), "project", "group"); err != nil {
		t.Fatalf("custom Group disappeared after rejected delete: %v", err)
	}
	if _, err := repository.Find(context.Background(), "project", "task"); err != nil {
		t.Fatalf("final child disappeared after rejected delete: %v", err)
	}
}

func groupSchedulingTestDB(t *testing.T) (*Repository, *gorm.DB) {
	t.Helper()
	repository, database := reopenTestDB(t)
	// Reopen tests intentionally use a minimal Project fixture. Group-scheduling
	// tests exercise production projectModel reads/writes (including Priority),
	// so extend that shared SQLite schema with the complete production model.
	if err := database.AutoMigrate(&projectModel{}); err != nil {
		t.Fatal(err)
	}
	return repository, database
}

func seedGroupSchedulingProject(t *testing.T, database *gorm.DB, automatic bool, anchor *time.Time) {
	t.Helper()
	if err := database.Create(&reopenProjectRecord{
		ID:                  "project",
		Name:                "Alpha",
		NameKey:             "alpha",
		Status:              "open",
		AutomaticScheduling: automatic,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if anchor != nil {
		if err := database.Model(&groupSchedulingProjectColumns{}).Where("id = ?", "project").Update("scheduling_start_date", *anchor).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func seedGroupWithTask(t *testing.T, database *gorm.DB, groupOverrides nodeModel, taskOverrides nodeModel) {
	t.Helper()
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	parentID := "group"
	group := nodeModel{
		ID:                    "group",
		ProjectID:             "project",
		ParentKey:             "",
		Name:                  "Platform",
		NameKey:               "platform",
		Position:              1,
		GroupSchedulingSource: "inherit",
		GroupLocalStatus:      "open",
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	applyNodeOverrides(&group, groupOverrides)
	task := nodeModel{
		ID:                    "task",
		ProjectID:             "project",
		ParentID:              &parentID,
		ParentKey:             parentID,
		Name:                  "Task",
		NameKey:               "task",
		Position:              1,
		GroupSchedulingSource: "inherit",
		GroupLocalStatus:      "open",
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	applyNodeOverrides(&task, taskOverrides)
	if err := database.Create(&[]nodeModel{group, task}).Error; err != nil {
		t.Fatal(err)
	}
}

func groupSchedulingBool(value bool) *bool { return &value }

func applyNodeOverrides(target *nodeModel, overrides nodeModel) {
	if overrides.GroupSchedulingSource != "" {
		target.GroupSchedulingSource = overrides.GroupSchedulingSource
	}
	if overrides.GroupAutomaticScheduling != nil {
		target.GroupAutomaticScheduling = overrides.GroupAutomaticScheduling
	}
	if overrides.GroupSchedulingStartDate != nil {
		target.GroupSchedulingStartDate = overrides.GroupSchedulingStartDate
	}
	if overrides.GroupLocalStatus != "" {
		target.GroupLocalStatus = overrides.GroupLocalStatus
	}
	if overrides.GroupLockedAutomaticScheduling != nil {
		target.GroupLockedAutomaticScheduling = overrides.GroupLockedAutomaticScheduling
	}
	if overrides.GroupLockedSchedulingStartDate != nil {
		target.GroupLockedSchedulingStartDate = overrides.GroupLockedSchedulingStartDate
	}
	if overrides.GroupSchedulingVersion != 0 {
		target.GroupSchedulingVersion = overrides.GroupSchedulingVersion
	}
	if overrides.ExecutionStart != nil {
		target.ExecutionStart = overrides.ExecutionStart
	}
	if overrides.ExecutionEnd != nil {
		target.ExecutionEnd = overrides.ExecutionEnd
	}
	if overrides.CommitmentStart != nil {
		target.CommitmentStart = overrides.CommitmentStart
	}
	if overrides.CommitmentEnd != nil {
		target.CommitmentEnd = overrides.CommitmentEnd
	}
}

func TestExecutableToGroupConversionCreatesInheritedBoundaryAndPreservesEffectiveConfig_US44_AC1(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	anchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &anchor)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	if err := database.Create(&nodeModel{
		ID:                    "parent-task",
		ProjectID:             "project",
		ParentKey:             "",
		Name:                  "Parent Task",
		NameKey:               "parent task",
		Position:              1,
		GroupSchedulingSource: "inherit",
		GroupLocalStatus:      "open",
		CreatedAt:             now,
		UpdatedAt:             now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	child, err := repository.Create(
		context.Background(), "child-task", "project", groupSchedulingString("parent-task"), "Child Task", false,
		now.Add(time.Hour), func(context.Context, string) error { return nil }, func(context.Context, []string) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	parent, err := repository.Find(context.Background(), "project", "parent-task")
	if err != nil {
		t.Fatal(err)
	}
	if !parent.HasChildren || parent.Scheduling.Source != "inherit" || parent.Scheduling.LocalStatus != "open" {
		t.Fatalf("converted Group scheduling=%+v hasChildren=%v, want inherit/open Group", parent.Scheduling, parent.HasChildren)
	}
	if child.ParentID == nil || *child.ParentID != parent.ID || child.Scheduling.EffectiveAutomatic != parent.Scheduling.EffectiveAutomatic {
		t.Fatalf("child=%+v parent=%+v, want child to keep equivalent inherited effective mode", child.Scheduling, parent.Scheduling)
	}
	if child.Scheduling.EffectiveStartDate == nil || !child.Scheduling.EffectiveStartDate.Equal(anchor) {
		t.Fatalf("child effective start=%v, want inherited project anchor %v", child.Scheduling.EffectiveStartDate, anchor)
	}
}

func TestAddSiblingBesideCustomGroupDoesNotCopyOverride_US44_AC1(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	anchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &anchor)
	manual := false
	seedGroupWithTask(t, database, nodeModel{GroupSchedulingSource: "override", GroupAutomaticScheduling: &manual}, nodeModel{})

	sibling, err := repository.CreateSibling(
		context.Background(), "sibling", "project", "group", "Sibling", time.Date(2026, 8, 7, 10, 0, 0, 0, time.UTC),
		func(context.Context, string) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if sibling.Scheduling.Source != "inherit" || !sibling.Scheduling.EffectiveAutomatic {
		t.Fatalf("sibling scheduling=%+v, want inherited Project ON and no copied Group override", sibling.Scheduling)
	}
	if sibling.Scheduling.EffectiveStartDate == nil || !sibling.Scheduling.EffectiveStartDate.Equal(anchor) {
		t.Fatalf("sibling start=%v, want Project anchor %v", sibling.Scheduling.EffectiveStartDate, anchor)
	}
}

func groupSchedulingString(value string) *string { return &value }

func TestLockedGroupRejectsRenameAndPreservesName_US44_AC14(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	seedGroupSchedulingProject(t, database, true, nil)
	seedGroupWithTask(t, database, nodeModel{GroupLocalStatus: "locked", GroupLockedAutomaticScheduling: groupSchedulingBool(true)}, nodeModel{})

	_, err := repository.Rename(
		context.Background(),
		"project",
		"task",
		"Renamed Task",
		time.Now().UTC(),
	)
	if !errors.Is(err, domain.ErrGroupLockedReadOnly) {
		t.Fatalf("err=%v, want GROUP_LOCKED_READ_ONLY", err)
	}
	stored := loadNodeModel(t, database, "task")
	if stored.Name != "Task" {
		t.Fatalf("locked rename changed name=%q, want Task", stored.Name)
	}
}

func TestReorderCannotShiftLockedSibling_US44_AC14_AC30(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	seedGroupSchedulingProject(t, database, true, nil)
	seedOpenRootAndLockedSibling(t, database)

	err := repository.Reorder(
		context.Background(),
		"project",
		"open-root",
		domain.MoveDown,
		time.Now().UTC(),
		func(context.Context, string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupLockedReadOnly) {
		t.Fatalf("err=%v, want GROUP_LOCKED_READ_ONLY", err)
	}
	if got := loadNodeModel(t, database, "open-root").Position; got != 1 {
		t.Fatalf("open root position=%d, want 1 after rejected reorder", got)
	}
	if got := loadNodeModel(t, database, "locked-group").Position; got != 2 {
		t.Fatalf("locked Group position=%d, want 2 after rejected reorder", got)
	}
}

func TestPlaceCannotReorderAroundLockedSibling_US44_AC14_AC30(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	seedGroupSchedulingProject(t, database, true, nil)
	seedOpenRootAndLockedSibling(t, database)

	err := repository.Place(
		context.Background(),
		"project",
		"open-root",
		"locked-group",
		domain.PlaceAfter,
		time.Now().UTC(),
		func(context.Context, string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupLockedReadOnly) {
		t.Fatalf("err=%v, want GROUP_LOCKED_READ_ONLY", err)
	}
	if got := loadNodeModel(t, database, "open-root").Position; got != 1 {
		t.Fatalf("open root position=%d, want 1 after rejected place", got)
	}
	if got := loadNodeModel(t, database, "locked-group").Position; got != 2 {
		t.Fatalf("locked Group position=%d, want 2 after rejected place", got)
	}
}

func TestCreateSiblingCannotShiftLockedSibling_US44_AC14_AC30(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	seedGroupSchedulingProject(t, database, true, nil)
	seedOpenRootAndLockedSibling(t, database)

	_, err := repository.CreateSibling(
		context.Background(),
		"new-root",
		"project",
		"open-root",
		"Inserted Root",
		time.Now().UTC(),
		func(context.Context, string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupLockedReadOnly) {
		t.Fatalf("err=%v, want GROUP_LOCKED_READ_ONLY", err)
	}
	if _, err := repository.Find(context.Background(), "project", "new-root"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("new root exists after rejected insert, err=%v", err)
	}
	if got := loadNodeModel(t, database, "locked-group").Position; got != 2 {
		t.Fatalf("locked Group position=%d, want 2 after rejected insert", got)
	}
}

func seedOpenRootAndLockedSibling(t *testing.T, database *gorm.DB) {
	t.Helper()
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	lockedParent := "locked-group"
	values := []nodeModel{
		{
			ID:                    "open-root",
			ProjectID:             "project",
			ParentKey:             "",
			Name:                  "Open Root",
			NameKey:               "open root",
			Position:              1,
			GroupSchedulingSource: "inherit",
			GroupLocalStatus:      "open",
			CreatedAt:             now,
			UpdatedAt:             now,
		},
		{
			ID:                             lockedParent,
			ProjectID:                      "project",
			ParentKey:                      "",
			Name:                           "Locked Group",
			NameKey:                        "locked group",
			Position:                       2,
			GroupSchedulingSource:          "inherit",
			GroupLocalStatus:               "locked",
			GroupLockedAutomaticScheduling: groupSchedulingBool(true),
			CreatedAt:                      now,
			UpdatedAt:                      now,
		},
		{
			ID:                    "locked-task",
			ProjectID:             "project",
			ParentID:              &lockedParent,
			ParentKey:             lockedParent,
			Name:                  "Locked Task",
			NameKey:               "locked task",
			Position:              1,
			GroupSchedulingSource: "inherit",
			GroupLocalStatus:      "open",
			CreatedAt:             now,
			UpdatedAt:             now,
		},
	}
	if err := database.Create(&values).Error; err != nil {
		t.Fatal(err)
	}
}

func TestDeleteCannotCompactLockedSiblingPosition_US44_AC14_AC30(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	seedGroupSchedulingProject(t, database, true, nil)
	seedOpenRootAndLockedSibling(t, database)

	err := repository.Delete(
		context.Background(),
		"project",
		"open-root",
		time.Now().UTC(),
		func(context.Context, string) error { return nil },
		func(context.Context, []string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupLockedReadOnly) {
		t.Fatalf("err=%v, want GROUP_LOCKED_READ_ONLY", err)
	}
	if _, err := repository.Find(context.Background(), "project", "open-root"); err != nil {
		t.Fatalf("open root disappeared after rejected delete: %v", err)
	}
	if got := loadNodeModel(t, database, "locked-group").Position; got != 2 {
		t.Fatalf("locked Group position=%d, want 2 after rejected delete", got)
	}
}

func TestMoveCannotCompactLockedSiblingPosition_US44_AC14_AC30(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	seedGroupSchedulingProject(t, database, true, nil)
	seedOpenRootAndLockedSibling(t, database)

	err := repository.Move(
		context.Background(),
		"project",
		"open-root",
		"conversion",
		nil,
		false,
		time.Now().UTC(),
		func(context.Context, string) error { return nil },
		func(context.Context, []string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupLockedReadOnly) {
		t.Fatalf("err=%v, want GROUP_LOCKED_READ_ONLY", err)
	}
	if got := loadNodeModel(t, database, "open-root").Position; got != 1 {
		t.Fatalf("open root position=%d, want 1 after rejected move", got)
	}
	if got := loadNodeModel(t, database, "locked-group").Position; got != 2 {
		t.Fatalf("locked Group position=%d, want 2 after rejected move", got)
	}
}

func TestMoveRejectsEffectiveLockedDestinationAndPreservesSource_US44_AC30(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	seedGroupSchedulingProject(t, database, true, nil)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	sourceParent := "source-group"
	lockedParent := "locked-target"
	values := []nodeModel{
		{ID: sourceParent, ProjectID: "project", ParentKey: "", Name: "Source", NameKey: "source", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: "moving-task", ProjectID: "project", ParentID: &sourceParent, ParentKey: sourceParent, Name: "Moving", NameKey: "moving", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: "source-remains", ProjectID: "project", ParentID: &sourceParent, ParentKey: sourceParent, Name: "Source Remains", NameKey: "source remains", Position: 2, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: lockedParent, ProjectID: "project", ParentKey: "", Name: "Locked Target", NameKey: "locked target", Position: 2, GroupSchedulingSource: "inherit", GroupLocalStatus: "locked", GroupLockedAutomaticScheduling: groupSchedulingBool(true), CreatedAt: now, UpdatedAt: now},
		{ID: "locked-child", ProjectID: "project", ParentID: &lockedParent, ParentKey: lockedParent, Name: "Locked Child", NameKey: "locked child", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&values).Error; err != nil {
		t.Fatal(err)
	}

	err := repository.Move(
		context.Background(), "project", "moving-task", "conversion", &lockedParent, false, now.Add(time.Hour),
		func(context.Context, string) error { return nil },
		func(context.Context, []string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupLockedReadOnly) {
		t.Fatalf("err=%v, want GROUP_LOCKED_READ_ONLY", err)
	}
	stored := loadNodeModel(t, database, "moving-task")
	if stored.ParentID == nil || *stored.ParentID != sourceParent || stored.ParentKey != sourceParent || stored.Position != 1 {
		t.Fatalf("moving Task changed after rejected destination: %#v", stored)
	}
}

func TestPlaceCannotShiftIntermediateLockedSibling_US44_AC14_AC30(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	seedGroupSchedulingProject(t, database, true, nil)
	seedOpenRootAndLockedSibling(t, database)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	if err := database.Create(&nodeModel{
		ID:                    "open-after",
		ProjectID:             "project",
		ParentKey:             "",
		Name:                  "Open After",
		NameKey:               "open after",
		Position:              3,
		GroupSchedulingSource: "inherit",
		GroupLocalStatus:      "open",
		CreatedAt:             now,
		UpdatedAt:             now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	err := repository.Place(
		context.Background(),
		"project",
		"open-root",
		"open-after",
		domain.PlaceAfter,
		time.Now().UTC(),
		func(context.Context, string) error { return nil },
	)
	if !errors.Is(err, domain.ErrGroupLockedReadOnly) {
		t.Fatalf("err=%v, want GROUP_LOCKED_READ_ONLY", err)
	}
	if got := loadNodeModel(t, database, "open-root").Position; got != 1 {
		t.Fatalf("open root position=%d, want 1 after rejected place", got)
	}
	if got := loadNodeModel(t, database, "locked-group").Position; got != 2 {
		t.Fatalf("locked Group position=%d, want 2 after rejected place", got)
	}
	if got := loadNodeModel(t, database, "open-after").Position; got != 3 {
		t.Fatalf("open-after position=%d, want 3 after rejected place", got)
	}
}

func TestRequiredGroupReopenPlanExpandsFixedPointAndRollsBackPreview_US44_AC18_AC25_AC26_AC37(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	lockedAuto := true
	projects := []projectModel{
		{ID: "project", Name: "Alpha", Status: "open", AutomaticScheduling: true, ScheduleVersion: 3, UpdatedAt: now},
		{ID: "project-b", Name: "Beta", Status: "locked", AutomaticScheduling: true, ScheduleVersion: 7, UpdatedAt: now},
	}
	if err := database.Create(&projects).Error; err != nil {
		t.Fatal(err)
	}
	groupBID := "group-b"
	values := []nodeModel{
		{ID: "group", ProjectID: "project", ParentKey: "", Name: "Root Group", NameKey: "root group", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "locked", GroupLockedAutomaticScheduling: &lockedAuto, CreatedAt: now, UpdatedAt: now},
		{ID: "task", ProjectID: "project", ParentID: stringPointer("group"), ParentKey: "group", Name: "Root Task", NameKey: "root task", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: groupBID, ProjectID: "project-b", ParentKey: "", Name: "Beta Group", NameKey: "beta group", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "locked", GroupLockedAutomaticScheduling: &lockedAuto, CreatedAt: now.Add(time.Minute), UpdatedAt: now.Add(time.Minute)},
		{ID: "task-b", ProjectID: "project-b", ParentID: &groupBID, ParentKey: groupBID, Name: "Beta Task", NameKey: "beta task", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&values).Error; err != nil {
		t.Fatal(err)
	}

	calls := 0
	schedule := func(ctx context.Context, _ string) error {
		calls++
		tx := sharedpersistence.Transaction(ctx, repository.db)
		var root nodeModel
		if err := tx.First(&root, "id = ?", "group").Error; err != nil {
			return err
		}
		if root.GroupLocalStatus != "open" {
			return errors.New("root Group was not staged open")
		}
		var betaProject projectModel
		if err := tx.First(&betaProject, "id = ?", "project-b").Error; err != nil {
			return err
		}
		var betaGroup nodeModel
		if err := tx.First(&betaGroup, "id = ?", groupBID).Error; err != nil {
			return err
		}
		if calls == 1 {
			if betaProject.Status != "locked" || betaGroup.GroupLocalStatus != "locked" {
				return errors.New("Beta scopes must remain locked during first preview")
			}
			return schedulingimpact.Error{
				Kind:         schedulingimpact.LockedScopeImpact,
				LockedGroups: []schedulingimpact.Group{{ID: groupBID, ProjectID: "project-b", Name: "Beta Group", Path: "Beta / Beta Group", Status: "locked", Version: betaGroup.GroupSchedulingVersion}},
			}
		}
		if betaProject.Status != "open" || betaGroup.GroupLocalStatus != "open" {
			return errors.New("Beta Project and Group were not staged open after closure expansion")
		}
		return nil
	}

	plan, err := requiredGroupReopenPlan(context.Background(), database, "project", "group", schedule)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 2 {
		t.Fatalf("preview calls=%d, want 2 fixed-point iterations", calls)
	}
	if len(plan.LockedProjects) != 1 || plan.LockedProjects[0].ID != "project-b" {
		t.Fatalf("locked Projects=%#v, want project-b", plan.LockedProjects)
	}
	if len(plan.LockedGroups) != 2 || plan.LockedGroups[0].ID != "group" || plan.LockedGroups[1].ID != "group-b" {
		t.Fatalf("locked Groups=%#v, want root and group-b", plan.LockedGroups)
	}
	if plan.Token == "" {
		t.Fatal("Group reopen plan token is empty")
	}
	if got := loadNodeModel(t, database, "group").GroupLocalStatus; got != "locked" {
		t.Fatalf("preview leaked root status=%q, want locked", got)
	}
	if got := loadNodeModel(t, database, "group-b").GroupLocalStatus; got != "locked" {
		t.Fatalf("preview leaked Beta Group status=%q, want locked", got)
	}
	var betaAfter projectModel
	if err := database.First(&betaAfter, "id = ?", "project-b").Error; err != nil {
		t.Fatal(err)
	}
	if betaAfter.Status != "locked" {
		t.Fatalf("preview leaked Beta Project status=%q, want locked", betaAfter.Status)
	}
}

func TestBulkGroupReopenRejectsStaleTokenWithoutLifecycleMutation_US44_AC25_AC37(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, nil)
	lockedAuto := true
	seedGroupWithTask(t, database, nodeModel{GroupLocalStatus: "locked", GroupLockedAutomaticScheduling: groupSchedulingBool(true)}, nodeModel{})
	if err := database.Model(&nodeModel{}).Where("id = ?", "group").Updates(map[string]any{
		"group_locked_automatic_scheduling": lockedAuto,
		"updated_at":                        now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	_, err := requiredGroupReopenPlan(
		context.Background(),
		database,
		"project",
		"group",
		func(context.Context, string) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repository.BulkReopenGroup(
		context.Background(),
		"project",
		"group",
		"stale-token",
		0,
		now.Add(2*time.Minute),
		func(context.Context, string) error { return nil },
	)
	var stale schedulingimpact.Error
	if !errors.As(err, &stale) || stale.Kind != schedulingimpact.StaleImpact {
		t.Fatalf("err=%v impact=%#v, want SCHEDULING_IMPACT_STALE", err, stale)
	}
	stored := loadNodeModel(t, database, "group")
	if stored.GroupLocalStatus != "locked" || stored.GroupLockedAutomaticScheduling == nil {
		t.Fatalf("stale Reopen All mutated Group lifecycle: %#v", stored)
	}
}

func TestBulkGroupReopenRejectsTokenAfterConcurrentHierarchyChange_US44_AC37(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, nil)
	seedGroupWithTask(t, database, nodeModel{GroupLocalStatus: "locked", GroupLockedAutomaticScheduling: groupSchedulingBool(true)}, nodeModel{})

	plan, err := requiredGroupReopenPlan(
		context.Background(),
		database,
		"project",
		"group",
		func(context.Context, string) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Token == "" {
		t.Fatal("preview token is empty")
	}
	if err := database.Model(&nodeModel{}).Where("id = ?", "task").Update("position", 2).Error; err != nil {
		t.Fatal(err)
	}

	_, err = repository.BulkReopenGroup(
		context.Background(),
		"project",
		"group",
		plan.Token,
		0,
		now.Add(time.Minute),
		func(context.Context, string) error { return nil },
	)
	var stale schedulingimpact.Error
	if !errors.As(err, &stale) || stale.Kind != schedulingimpact.StaleImpact {
		t.Fatalf("err=%v impact=%#v, want stale impact after hierarchy change", err, stale)
	}
	stored := loadNodeModel(t, database, "group")
	if stored.GroupLocalStatus != "locked" {
		t.Fatalf("stale hierarchy confirmation reopened Group: %#v", stored)
	}
}

func TestMoveTaskAdoptsDestinationEffectiveScheduling_US44_AC27(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	anchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &anchor)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	automaticOff := false
	sourceID, targetID := "source", "manual-group"
	values := []nodeModel{
		{ID: sourceID, ProjectID: "project", ParentKey: "", Name: "Source", NameKey: "source", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: "moving-task", ProjectID: "project", ParentID: &sourceID, ParentKey: sourceID, Name: "Moving Task", NameKey: "moving task", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: targetID, ProjectID: "project", ParentKey: "", Name: "Manual Group", NameKey: "manual group", Position: 2, GroupSchedulingSource: "override", GroupAutomaticScheduling: &automaticOff, GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: "target-child", ProjectID: "project", ParentID: &targetID, ParentKey: targetID, Name: "Target Child", NameKey: "target child", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&values).Error; err != nil {
		t.Fatal(err)
	}

	if err := repository.Move(
		context.Background(),
		"project",
		"moving-task",
		"conversion",
		&targetID,
		false,
		now.Add(time.Minute),
		func(context.Context, string) error { return nil },
		func(context.Context, []string) error { return nil },
	); err != nil {
		t.Fatal(err)
	}

	moved, err := repository.Find(context.Background(), "project", "moving-task")
	if err != nil {
		t.Fatal(err)
	}
	if moved.ParentID == nil || *moved.ParentID != targetID {
		t.Fatalf("moved Task parent=%v, want %s", moved.ParentID, targetID)
	}
	if moved.Scheduling.EffectiveAutomatic {
		t.Fatalf("moved Task scheduling=%+v, want destination effective Automatic OFF", moved.Scheduling)
	}
}

func TestMoveInheritedGroupAdoptsDestinationEffectiveScheduling_US44_AC28(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	anchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &anchor)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	automaticOff := false
	movingID, targetID := "moving-group", "manual-group"
	values := []nodeModel{
		{ID: movingID, ProjectID: "project", ParentKey: "", Name: "Moving Group", NameKey: "moving group", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: "moving-child", ProjectID: "project", ParentID: &movingID, ParentKey: movingID, Name: "Moving Child", NameKey: "moving child", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: targetID, ProjectID: "project", ParentKey: "", Name: "Manual Group", NameKey: "manual group", Position: 2, GroupSchedulingSource: "override", GroupAutomaticScheduling: &automaticOff, GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: "target-child", ProjectID: "project", ParentID: &targetID, ParentKey: targetID, Name: "Target Child", NameKey: "target child", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&values).Error; err != nil {
		t.Fatal(err)
	}

	if err := repository.Move(
		context.Background(),
		"project",
		movingID,
		"conversion",
		&targetID,
		false,
		now.Add(time.Minute),
		func(context.Context, string) error { return nil },
		func(context.Context, []string) error { return nil },
	); err != nil {
		t.Fatal(err)
	}

	moved, err := repository.Find(context.Background(), "project", movingID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.Scheduling.Source != "inherit" || moved.Scheduling.EffectiveAutomatic {
		t.Fatalf("moved inherited Group scheduling=%+v, want inherited destination Automatic OFF", moved.Scheduling)
	}
	child, err := repository.Find(context.Background(), "project", "moving-child")
	if err != nil {
		t.Fatal(err)
	}
	if child.Scheduling.EffectiveAutomatic {
		t.Fatalf("moved Group descendant scheduling=%+v, want inherited destination Automatic OFF", child.Scheduling)
	}
}

func TestMoveCustomGroupRetainsOwnSchedulingOverride_US44_AC29(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	projectAnchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	customAnchor := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, true, &projectAnchor)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	automaticOn, automaticOff := true, false
	movingID, targetID := "custom-group", "manual-group"
	values := []nodeModel{
		{ID: movingID, ProjectID: "project", ParentKey: "", Name: "Custom Group", NameKey: "custom group", Position: 1, GroupSchedulingSource: "override", GroupAutomaticScheduling: &automaticOn, GroupSchedulingStartDate: &customAnchor, GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: "moving-child", ProjectID: "project", ParentID: &movingID, ParentKey: movingID, Name: "Moving Child", NameKey: "moving child", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: targetID, ProjectID: "project", ParentKey: "", Name: "Manual Group", NameKey: "manual group", Position: 2, GroupSchedulingSource: "override", GroupAutomaticScheduling: &automaticOff, GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
		{ID: "target-child", ProjectID: "project", ParentID: &targetID, ParentKey: targetID, Name: "Target Child", NameKey: "target child", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&values).Error; err != nil {
		t.Fatal(err)
	}

	if err := repository.Move(
		context.Background(),
		"project",
		movingID,
		"conversion",
		&targetID,
		false,
		now.Add(time.Minute),
		func(context.Context, string) error { return nil },
		func(context.Context, []string) error { return nil },
	); err != nil {
		t.Fatal(err)
	}

	moved, err := repository.Find(context.Background(), "project", movingID)
	if err != nil {
		t.Fatal(err)
	}
	if moved.Scheduling.Source != "override" || !moved.Scheduling.EffectiveAutomatic || moved.Scheduling.EffectiveStartDate == nil || !moved.Scheduling.EffectiveStartDate.Equal(customAnchor) {
		t.Fatalf("moved custom Group scheduling=%+v, want retained ON/%v override", moved.Scheduling, customAnchor)
	}
}

func TestParentGroupReopenReresolvesInheritanceAndPreservesNestedLocalLock_US44_AC18_AC19(t *testing.T) {
	repository, database := groupSchedulingTestDB(t)
	newAnchor := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	oldAnchor := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	seedGroupSchedulingProject(t, database, false, &newAnchor)
	now := time.Date(2026, 8, 7, 9, 0, 0, 0, time.UTC)
	lockedAuto := true
	rootID, childID := "root-group", "child-group"
	values := []nodeModel{
		{ID: rootID, ProjectID: "project", ParentKey: "", Name: "Root Group", NameKey: "root group", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "locked", GroupLockedAutomaticScheduling: &lockedAuto, GroupLockedSchedulingStartDate: &oldAnchor, CreatedAt: now, UpdatedAt: now},
		{ID: childID, ProjectID: "project", ParentID: &rootID, ParentKey: rootID, Name: "Child Group", NameKey: "child group", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "locked", GroupLockedAutomaticScheduling: &lockedAuto, GroupLockedSchedulingStartDate: &oldAnchor, CreatedAt: now, UpdatedAt: now},
		{ID: "task", ProjectID: "project", ParentID: &childID, ParentKey: childID, Name: "Task", NameKey: "task", Position: 1, GroupSchedulingSource: "inherit", GroupLocalStatus: "open", CreatedAt: now, UpdatedAt: now},
	}
	if err := database.Create(&values).Error; err != nil {
		t.Fatal(err)
	}

	reopened, err := repository.ChangeGroupStatus(
		context.Background(),
		"project",
		rootID,
		"open",
		0,
		now.Add(time.Minute),
		func(context.Context, string) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Scheduling.LocalStatus != "open" || reopened.Scheduling.EffectiveAutomatic || reopened.Scheduling.EffectiveStartDate == nil || !reopened.Scheduling.EffectiveStartDate.Equal(newAnchor) {
		t.Fatalf("reopened parent scheduling=%+v, want current inherited OFF/%v", reopened.Scheduling, newAnchor)
	}
	child, err := repository.Find(context.Background(), "project", childID)
	if err != nil {
		t.Fatal(err)
	}
	if child.Scheduling.LocalStatus != "locked" || child.Scheduling.EffectiveLifecycle != "locked" || !child.Scheduling.EffectiveAutomatic || child.Scheduling.EffectiveStartDate == nil || !child.Scheduling.EffectiveStartDate.Equal(oldAnchor) {
		t.Fatalf("nested local lock scheduling=%+v, want preserved frozen ON/%v", child.Scheduling, oldAnchor)
	}
}

func stringPointer(value string) *string { return &value }
