package gormrepo

import (
	"fmt"

	schedulingdomain "github.com/banggok/sched_mind/backend/internal/scheduling/domain"
	"gorm.io/gorm"
)

func (repository *Repository) reconcileAutomaticOwnership(
	database *gorm.DB,
	state *portfolioState,
	execution *timelineResult,
) (bool, map[string]struct{}, error) {
	desired := execution.automaticBlocker
	now := repository.now()
	changed := false
	dirtyProjects := make(map[string]struct{})

	byBlocked := make(map[string][]int)
	for index, dependency := range state.dependencies {
		byBlocked[dependency.BlockedTaskID] = append(byBlocked[dependency.BlockedTaskID], index)
	}

	for taskID, task := range state.tasks {
		if _, leaf := state.leafOrder[taskID]; !leaf {
			continue
		}
		project := state.projects[task.ProjectID]
		if project.Status != "open" || !project.AutomaticScheduling || task.ActualStart != nil && task.ActualEnd != nil {
			continue
		}

		desiredBlocker := desired[taskID]
		desiredFound := false
		for _, index := range byBlocked[taskID] {
			dependency := &state.dependencies[index]
			if !dependency.AutomaticOwned {
				continue
			}
			if desiredBlocker != "" && dependency.BlockingTaskID == desiredBlocker && !desiredFound {
				desiredFound = true
				continue
			}

			changed = true
			dirtyProjects[task.ProjectID] = struct{}{}
			if dependency.ManualOwned {
				if err := database.Model(&dependencyModel{}).
					Where("id = ? AND automatic_owned = ?", dependency.ID, true).
					Updates(map[string]any{"automatic_owned": false, "updated_at": now}).Error; err != nil {
					return false, nil, fmt.Errorf("remove stale automatic dependency ownership: %w", err)
				}
				dependency.AutomaticOwned = false
				dependency.UpdatedAt = now
				continue
			}
			if err := database.Delete(&dependencyModel{}, "id = ?", dependency.ID).Error; err != nil {
				return false, nil, fmt.Errorf("delete stale automatic dependency: %w", err)
			}
			dependency.ID = ""
		}

		if desiredBlocker == "" || desiredFound {
			continue
		}
		if _, exists := state.tasks[desiredBlocker]; !exists {
			return false, nil, fmt.Errorf("%w: automatic blocker %s does not exist", schedulingdomain.ErrDataIntegrity, desiredBlocker)
		}

		var existing dependencyModel
		result := database.Where("blocking_task_id = ? AND blocked_task_id = ?", desiredBlocker, taskID).
			Limit(1).
			Find(&existing)
		if result.Error != nil {
			return false, nil, fmt.Errorf("load automatic dependency endpoint pair: %w", result.Error)
		}
		if result.RowsAffected > 0 {
			if !existing.AutomaticOwned {
				if updateErr := database.Model(&dependencyModel{}).
					Where("id = ? AND automatic_owned = ?", existing.ID, false).
					Updates(map[string]any{"automatic_owned": true, "updated_at": now}).Error; updateErr != nil {
					return false, nil, fmt.Errorf("add automatic dependency ownership: %w", updateErr)
				}
				changed = true
				dirtyProjects[task.ProjectID] = struct{}{}
			}
			updated := false
			for index := range state.dependencies {
				if state.dependencies[index].ID == existing.ID {
					state.dependencies[index].AutomaticOwned = true
					state.dependencies[index].UpdatedAt = now
					updated = true
					break
				}
			}
			if !updated {
				existing.AutomaticOwned = true
				existing.UpdatedAt = now
				state.dependencies = append(state.dependencies, existing)
			}
			continue
		}

		id, idErr := repository.newID()
		if idErr != nil {
			return false, nil, fmt.Errorf("generate automatic dependency ID: %w", idErr)
		}
		created := dependencyModel{
			ID: id, BlockingTaskID: desiredBlocker, BlockedTaskID: taskID,
			ManualOwned: false, AutomaticOwned: true, CreatedAt: now, UpdatedAt: now,
		}
		if createErr := database.Create(&created).Error; createErr != nil {
			return false, nil, fmt.Errorf("create automatic dependency: %w", createErr)
		}
		state.dependencies = append(state.dependencies, created)
		changed = true
		dirtyProjects[task.ProjectID] = struct{}{}
	}

	compacted := state.dependencies[:0]
	for _, dependency := range state.dependencies {
		if dependency.ID != "" {
			compacted = append(compacted, dependency)
		}
	}
	state.dependencies = compacted
	return changed, dirtyProjects, nil
}
