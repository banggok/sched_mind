package gormrepo

import (
	"context"
	"fmt"
	"sort"
)

type wbsContextRow struct {
	ID        string
	ProjectID string
	ParentID  *string
	Position  int
	Name      string
}

type wbsContextValue struct {
	Path       string
	Rank       int
	ParentName string
}

func (repository *Repository) loadWBSContext(ctx context.Context, taskRows []taskProjectionRow) (map[string]wbsContextValue, []wbsContextRow, error) {
	projectSet := make(map[string]struct{})
	for _, task := range taskRows {
		projectSet[task.ProjectID] = struct{}{}
	}
	if len(projectSet) == 0 {
		return map[string]wbsContextValue{}, nil, nil
	}
	projectIDs := make([]string, 0, len(projectSet))
	for projectID := range projectSet {
		projectIDs = append(projectIDs, projectID)
	}
	sort.Strings(projectIDs)

	var rows []wbsContextRow
	if err := repository.database.WithContext(ctx).Table("wbs_nodes").
		Select("id, project_id, parent_id, position, name").
		Where("project_id IN ?", projectIDs).
		Order("project_id ASC").Order("parent_key ASC").Order("position ASC").Order("id ASC").
		Scan(&rows).Error; err != nil {
		return nil, nil, fmt.Errorf("query sprint WBS context: %w", err)
	}

	projectNames := make(map[string]string, len(projectIDs))
	for _, task := range taskRows {
		projectNames[task.ProjectID] = task.ProjectName
	}
	byProject := make(map[string][]wbsContextRow)
	for _, row := range rows {
		byProject[row.ProjectID] = append(byProject[row.ProjectID], row)
	}
	result := make(map[string]wbsContextValue, len(rows))
	for _, projectID := range projectIDs {
		children := make(map[string][]wbsContextRow)
		for _, row := range byProject[projectID] {
			parentID := ""
			if row.ParentID != nil {
				parentID = *row.ParentID
			}
			children[parentID] = append(children[parentID], row)
		}
		for parentID := range children {
			sort.Slice(children[parentID], func(left, right int) bool {
				if children[parentID][left].Position != children[parentID][right].Position {
					return children[parentID][left].Position < children[parentID][right].Position
				}
				return children[parentID][left].ID < children[parentID][right].ID
			})
		}
		rowsByID := make(map[string]wbsContextRow, len(byProject[projectID]))
		for _, row := range byProject[projectID] {
			rowsByID[row.ID] = row
		}
		rank := 0
		visited := make(map[string]struct{}, len(byProject[projectID]))
		var visit func(wbsContextRow, string)
		visit = func(row wbsContextRow, path string) {
			if _, exists := visited[row.ID]; exists {
				return
			}
			visited[row.ID] = struct{}{}
			parentName := projectNames[row.ProjectID]
			if row.ParentID != nil {
				if parent, exists := rowsByID[*row.ParentID]; exists {
					parentName = parent.Name
				} else {
					parentName = ""
				}
			}
			result[row.ID] = wbsContextValue{Path: path, Rank: rank, ParentName: parentName}
			rank++
			for index, child := range children[row.ID] {
				visit(child, fmt.Sprintf("%s.%d", path, index+1))
			}
		}
		for index, root := range children[""] {
			visit(root, fmt.Sprintf("%d", index+1))
		}
		// Defensive fallback for legacy orphaned rows. The deterministic fallback
		// keeps the read available without inventing a hierarchy relationship.
		orphans := make([]wbsContextRow, 0)
		for _, row := range byProject[projectID] {
			if _, exists := visited[row.ID]; !exists {
				orphans = append(orphans, row)
			}
		}
		sort.Slice(orphans, func(left, right int) bool {
			if orphans[left].Position != orphans[right].Position {
				return orphans[left].Position < orphans[right].Position
			}
			return orphans[left].ID < orphans[right].ID
		})
		for index, orphan := range orphans {
			visit(orphan, fmt.Sprintf("?%d", index+1))
		}
	}
	return result, rows, nil
}
