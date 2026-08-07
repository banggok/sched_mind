package gormrepo

import (
	"context"
	"testing"
)

func TestMixedGroupSchedulingUsesEffectiveModeAnchorAndOwningProjectBuffer_US44_AC9_AC10_AC12(t *testing.T) {
	t.Run("Project OFF with Group ON uses concrete scheduler and Project Buffer", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		project := automaticProject("project", 1, "2026-08-03", 20)
		project.AutomaticScheduling = false
		seedProject(t, database, project)
		seedMember(t, database, "member", "8", "0")

		groupID := "group"
		automatic := true
		groupAnchor := mustDate("2026-08-10")
		seedTask(t, database, taskModel{
			ID:                       groupID,
			ProjectID:                "project",
			ParentKey:                "",
			Name:                     "Automatic Group",
			Position:                 1,
			GroupSchedulingSource:    "override",
			GroupAutomaticScheduling: &automatic,
			GroupSchedulingStartDate: &groupAnchor,
			GroupLocalStatus:         "open",
			CreatedAt:                mustDate("2026-08-01"),
			UpdatedAt:                mustDate("2026-08-01"),
		})
		task := schedulableTask("task", "project", 1, "member", 480, 0)
		task.ParentID = &groupID
		task.ParentKey = groupID
		seedTask(t, database, task)

		if err := repository.RecalculatePortfolio(context.Background(), []string{"project"}); err != nil {
			t.Fatal(err)
		}

		confirmed := loadScheduledTask(t, database, "task")
		assertDate(t, "execution start", confirmed.ExecutionStart, "2026-08-10")
		assertDate(t, "execution end", confirmed.ExecutionEnd, "2026-08-10")
		assertDate(t, "commitment start", confirmed.CommitmentStart, "2026-08-10")
		assertDate(t, "commitment end", confirmed.CommitmentEnd, "2026-08-11")
	})

	t.Run("Project ON with Group OFF preserves manual dates as fixed work", func(t *testing.T) {
		repository, database := schedulerRepository(t)
		seedProject(t, database, automaticProject("project", 1, "2026-08-03", 20))
		seedMember(t, database, "member", "8", "0")

		groupID := "group"
		automatic := false
		seedTask(t, database, taskModel{
			ID:                       groupID,
			ProjectID:                "project",
			ParentKey:                "",
			Name:                     "Manual Group",
			Position:                 1,
			GroupSchedulingSource:    "override",
			GroupAutomaticScheduling: &automatic,
			GroupLocalStatus:         "open",
			CreatedAt:                mustDate("2026-08-01"),
			UpdatedAt:                mustDate("2026-08-01"),
		})
		manual := schedulableTask("manual", "project", 1, "member", 480, 0)
		manual.ParentID = &groupID
		manual.ParentKey = groupID
		manual.ExecutionStart = datePointer(mustDate("2026-08-12"))
		manual.ExecutionEnd = datePointer(mustDate("2026-08-12"))
		manual.CommitmentStart = datePointer(mustDate("2026-08-12"))
		manual.CommitmentEnd = datePointer(mustDate("2026-08-12"))
		seedTask(t, database, manual)

		if err := repository.RecalculatePortfolio(context.Background(), []string{"project"}); err != nil {
			t.Fatal(err)
		}

		confirmed := loadScheduledTask(t, database, "manual")
		assertDate(t, "manual execution start", confirmed.ExecutionStart, "2026-08-12")
		assertDate(t, "manual execution end", confirmed.ExecutionEnd, "2026-08-12")
		assertDate(t, "manual commitment start", confirmed.CommitmentStart, "2026-08-12")
		assertDate(t, "manual commitment end", confirmed.CommitmentEnd, "2026-08-12")
	})
}
