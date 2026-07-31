package gormrepo

import (
	"fmt"
	"time"

	dependencydomain "github.com/banggok/sched_mind/backend/internal/dependencies/domain"
	"gorm.io/gorm"
)

type schedulePreviewDependencyRow struct {
	ID, BlockingTaskID, BlockedTaskID string
	ManualOwned, AutomaticOwned       bool
	CreatedAt, UpdatedAt              time.Time
	TaskID, TaskName                  string
	ProjectID, ProjectName            string
	ActualEnd, ExpectedStart          *time.Time
}

func loadSchedulePreviewDependencies(tx *gorm.DB, taskID string) (dependencydomain.Detail, error) {
	incoming, err := loadSchedulePreviewDependencyRows(tx, taskID, true)
	if err != nil {
		return dependencydomain.Detail{}, err
	}
	outgoing, err := loadSchedulePreviewDependencyRows(tx, taskID, false)
	if err != nil {
		return dependencydomain.Detail{}, err
	}
	return dependencydomain.Detail{
		BlockedBy: mapSchedulePreviewDependencyRows(incoming),
		Blocks:    mapSchedulePreviewDependencyRows(outgoing),
	}, nil
}

func loadSchedulePreviewDependencyRows(tx *gorm.DB, taskID string, incoming bool) ([]schedulePreviewDependencyRow, error) {
	query := tx.Table("task_dependencies d").
		Select("d.id,d.blocking_task_id,d.blocked_task_id,d.manual_owned,d.automatic_owned,d.created_at,d.updated_at,t.id task_id,t.name task_name,t.project_id,p.name project_name,t.actual_end,t.execution_start expected_start")
	if incoming {
		query = query.
			Joins("JOIN wbs_nodes t ON t.id = d.blocking_task_id").
			Joins("JOIN projects p ON p.id = t.project_id").
			Where("d.blocked_task_id = ?", taskID)
	} else {
		query = query.
			Joins("JOIN wbs_nodes t ON t.id = d.blocked_task_id").
			Joins("JOIN projects p ON p.id = t.project_id").
			Where("d.blocking_task_id = ?", taskID)
	}
	var rows []schedulePreviewDependencyRow
	if err := query.
		Order("LOWER(p.name) ASC").
		Order("t.parent_key ASC").
		Order("t.position ASC").
		Order("t.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load schedule preview dependencies: %w", err)
	}
	return rows, nil
}

func mapSchedulePreviewDependencyRows(rows []schedulePreviewDependencyRow) []dependencydomain.Item {
	items := make([]dependencydomain.Item, 0, len(rows))
	for _, row := range rows {
		items = append(items, dependencydomain.Item{
			Dependency: dependencydomain.Dependency{
				ID:             row.ID,
				BlockingTaskID: row.BlockingTaskID,
				BlockedTaskID:  row.BlockedTaskID,
				ManualOwned:    row.ManualOwned,
				AutomaticOwned: row.AutomaticOwned,
				CreatedAt:      row.CreatedAt,
				UpdatedAt:      row.UpdatedAt,
			},
			Task: dependencydomain.Task{
				ID:            row.TaskID,
				Name:          row.TaskName,
				ProjectID:     row.ProjectID,
				ProjectName:   row.ProjectName,
				ActualEnd:     row.ActualEnd,
				ExpectedStart: row.ExpectedStart,
			},
		})
	}
	return items
}
