package groupscheduling

import (
	"testing"
	"time"
)

func TestResolverUsesNearestOverrideAndFieldLevelStartDateInheritance_US44_AC1ToAC6(t *testing.T) {
	projectStart := resolverDate("2026-08-03")
	parentStart := resolverDate("2026-08-10")
	parentID := "parent"
	childID := "child"
	parentAutomatic := true
	childAutomatic := false

	resolver, err := New(Project{
		ID:                  "project",
		Name:                "Project Alpha",
		Status:              "open",
		AutomaticScheduling: false,
		SchedulingStartDate: &projectStart,
	}, []Node{
		{
			ID:                          parentID,
			Name:                        "Platform",
			Source:                      SourceOverride,
			AutomaticSchedulingOverride: &parentAutomatic,
			SchedulingStartDateOverride: &parentStart,
			LocalStatus:                 StatusOpen,
		},
		{
			ID:                          childID,
			ParentID:                    &parentID,
			Name:                        "API",
			Source:                      SourceOverride,
			AutomaticSchedulingOverride: &childAutomatic,
			// Null is intentional: the child keeps its own Automatic Scheduling
			// while inheriting the nearest parent Scheduling Start Date.
			LocalStatus: StatusOpen,
		},
		{
			ID:          "task",
			ParentID:    &childID,
			Name:        "Task",
			Source:      SourceInherit,
			LocalStatus: StatusOpen,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	effective, err := resolver.EffectiveFor("task")
	if err != nil {
		t.Fatal(err)
	}
	if effective.AutomaticScheduling {
		t.Fatal("Task Automatic Scheduling=true, want child override OFF")
	}
	if effective.AutomaticOwnerID != childID || effective.AutomaticOwnerName != "API" {
		t.Fatalf("automatic owner=%q/%q, want child Group", effective.AutomaticOwnerID, effective.AutomaticOwnerName)
	}
	if effective.SchedulingStartDate == nil || !effective.SchedulingStartDate.Equal(parentStart) {
		t.Fatalf("effective start=%v, want parent start %v", effective.SchedulingStartDate, parentStart)
	}
	if effective.StartDateOwnerID != parentID || effective.StartDateOwnerName != "Platform" {
		t.Fatalf("start owner=%q/%q, want parent Group", effective.StartDateOwnerID, effective.StartDateOwnerName)
	}
	inherited, err := resolver.InheritedFor(childID)
	if err != nil {
		t.Fatal(err)
	}
	if !inherited.AutomaticScheduling || inherited.AutomaticOwnerID != parentID {
		t.Fatalf("inherited automatic=%v owner=%q, want parent Group ON", inherited.AutomaticScheduling, inherited.AutomaticOwnerID)
	}
	if inherited.SchedulingStartDate == nil || !inherited.SchedulingStartDate.Equal(parentStart) {
		t.Fatalf("inherited start=%v, want parent start %v", inherited.SchedulingStartDate, parentStart)
	}
}

func TestResolverLockedGroupFreezesConfigWhileProjectAndParentCanChange_US44_AC17_AC18(t *testing.T) {
	projectStart := resolverDate("2026-08-03")
	parentStart := resolverDate("2026-08-05")
	lockedStart := resolverDate("2026-08-12")
	parentID := "parent"
	lockedID := "locked"
	parentAutomatic := false
	lockedAutomatic := true

	resolver, err := New(Project{
		ID:                  "project",
		Name:                "Project Alpha",
		Status:              "open",
		AutomaticScheduling: false,
		SchedulingStartDate: &projectStart,
	}, []Node{
		{
			ID:                          parentID,
			Name:                        "Platform",
			Source:                      SourceOverride,
			AutomaticSchedulingOverride: &parentAutomatic,
			SchedulingStartDateOverride: &parentStart,
			LocalStatus:                 StatusOpen,
		},
		{
			ID:                        lockedID,
			ParentID:                  &parentID,
			Name:                      "API",
			Source:                    SourceInherit,
			LocalStatus:               StatusLocked,
			LockedAutomaticScheduling: &lockedAutomatic,
			LockedSchedulingStartDate: &lockedStart,
		},
		{
			ID:          "task",
			ParentID:    &lockedID,
			Name:        "Task",
			Source:      SourceInherit,
			LocalStatus: StatusOpen,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	effective, err := resolver.EffectiveFor("task")
	if err != nil {
		t.Fatal(err)
	}
	if !effective.AutomaticScheduling || effective.SchedulingStartDate == nil || !effective.SchedulingStartDate.Equal(lockedStart) {
		t.Fatalf("locked effective config=%+v, want frozen ON/%v", effective, lockedStart)
	}
	if effective.Lifecycle != LifecycleLocked || effective.LockOwnerID != lockedID {
		t.Fatalf("lifecycle=%q owner=%q, want Group lock", effective.Lifecycle, effective.LockOwnerID)
	}
}

func TestResolverLockedGroupFreezesNilAnchorInsteadOfFallingBackAfterParentChange_US44_AC17_AC18(t *testing.T) {
	projectStart := resolverDate("2026-09-01")
	lockedAutomatic := true
	groupID := "group"
	resolver, err := New(Project{
		ID:                  "project",
		Name:                "Project Alpha",
		Status:              "open",
		AutomaticScheduling: true,
		SchedulingStartDate: &projectStart,
	}, []Node{
		{
			ID:                        groupID,
			Name:                      "Platform",
			Source:                    SourceInherit,
			LocalStatus:               StatusLocked,
			LockedAutomaticScheduling: &lockedAutomatic,
			// nil is the frozen resolved value from lock time.
			LockedSchedulingStartDate: nil,
		},
		{
			ID:          "task",
			ParentID:    &groupID,
			Name:        "Task",
			Source:      SourceInherit,
			LocalStatus: StatusOpen,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	effective, err := resolver.EffectiveFor("task")
	if err != nil {
		t.Fatal(err)
	}
	if effective.SchedulingStartDate != nil {
		t.Fatalf("locked effective start=%v, want frozen no anchor", effective.SchedulingStartDate)
	}
	if effective.StartDateOwnerID != groupID {
		t.Fatalf("start owner=%q, want locked Group", effective.StartDateOwnerID)
	}
}

func TestResolverProjectLifecycleDominatesGroupLifecycleAndLocalOwnerRemainsDiscoverable_US44_AC15_AC16_AC36(t *testing.T) {
	lockedAutomatic := true
	groupID := "group"
	resolver, err := New(Project{
		ID:                  "project",
		Name:                "Project Alpha",
		Status:              "closed",
		AutomaticScheduling: true,
	}, []Node{
		{
			ID:                        groupID,
			Name:                      "Platform",
			Source:                    SourceInherit,
			LocalStatus:               StatusLocked,
			LockedAutomaticScheduling: &lockedAutomatic,
		},
		{
			ID:          "task",
			ParentID:    &groupID,
			Name:        "Task",
			Source:      SourceInherit,
			LocalStatus: StatusOpen,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	effective, err := resolver.EffectiveFor("task")
	if err != nil {
		t.Fatal(err)
	}
	if effective.Lifecycle != LifecycleClosed || effective.LockOwnerID != "project" {
		t.Fatalf("effective lifecycle=%q owner=%q, want closed Project owner", effective.Lifecycle, effective.LockOwnerID)
	}
	ownerID, ownerName, err := resolver.LocalLockOwnerFor("task")
	if err != nil {
		t.Fatal(err)
	}
	if ownerID != groupID || ownerName != "Platform" {
		t.Fatalf("local lock owner=%q/%q, want nested Group", ownerID, ownerName)
	}
}

func resolverDate(value string) time.Time {
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		panic(err)
	}
	return parsed
}

func TestResolverProjectLockDominatesOpenGroup_US44_AC15_AC16(t *testing.T) {
	groupID := "group"
	resolver, err := New(Project{
		ID:                  "project",
		Name:                "Project Alpha",
		Status:              "locked",
		AutomaticScheduling: true,
	}, []Node{
		{ID: groupID, Name: "Platform", Source: SourceInherit, LocalStatus: StatusOpen},
		{ID: "task", ParentID: &groupID, Name: "Task", Source: SourceInherit, LocalStatus: StatusOpen},
	})
	if err != nil {
		t.Fatal(err)
	}

	effective, err := resolver.EffectiveFor("task")
	if err != nil {
		t.Fatal(err)
	}
	if effective.Lifecycle != LifecycleLocked || effective.LockOwnerID != "project" {
		t.Fatalf("effective lifecycle=%q owner=%q, want locked Project owner", effective.Lifecycle, effective.LockOwnerID)
	}
}

func TestResolverAncestorGroupLockBlocksNestedOverrideFromChangingFrozenConfig_US44_AC14_AC15_AC17(t *testing.T) {
	projectDate := resolverDate("2026-08-03")
	lockedDate := resolverDate("2026-08-05")
	nestedDate := resolverDate("2026-08-10")
	lockedAutomatic := true
	nestedAutomatic := false
	rootID := "root-group"
	nestedID := "nested-group"
	resolver, err := New(Project{
		ID:                  "project",
		Name:                "Project Alpha",
		Status:              "open",
		AutomaticScheduling: true,
		SchedulingStartDate: &projectDate,
	}, []Node{
		{
			ID:                        rootID,
			Name:                      "Platform",
			Source:                    SourceInherit,
			LocalStatus:               StatusLocked,
			LockedAutomaticScheduling: &lockedAutomatic,
			LockedSchedulingStartDate: &lockedDate,
		},
		{
			ID:                          nestedID,
			ParentID:                    &rootID,
			Name:                        "API",
			Source:                      SourceOverride,
			AutomaticSchedulingOverride: &nestedAutomatic,
			SchedulingStartDateOverride: &nestedDate,
			LocalStatus:                 StatusOpen,
		},
		{ID: "task", ParentID: &nestedID, Name: "Task", Source: SourceInherit, LocalStatus: StatusOpen},
	})
	if err != nil {
		t.Fatal(err)
	}

	effective, err := resolver.EffectiveFor("task")
	if err != nil {
		t.Fatal(err)
	}
	if !effective.AutomaticScheduling {
		t.Fatal("nested custom OFF bypassed the ancestor Group lock snapshot")
	}
	if effective.SchedulingStartDate == nil || !effective.SchedulingStartDate.Equal(lockedDate) {
		t.Fatalf("effective start=%v, want frozen ancestor start %v", effective.SchedulingStartDate, lockedDate)
	}
	if effective.AutomaticOwnerID != rootID || effective.StartDateOwnerID != rootID {
		t.Fatalf("effective config owners=%q/%q, want ancestor Group %q", effective.AutomaticOwnerID, effective.StartDateOwnerID, rootID)
	}
	if effective.Lifecycle != LifecycleLocked || effective.LockOwnerID != rootID {
		t.Fatalf("effective lifecycle=%q owner=%q, want ancestor Group lock", effective.Lifecycle, effective.LockOwnerID)
	}
}
