package domain

import (
	"cmp"
	"slices"
	"time"
)

type Candidate struct {
	TaskID                    string
	MemberID                  string
	ExecutionEnd              time.Time
	ProjectPriority           int
	WBSOrder                  int
	InSprintAllocationMinutes int64
}

type SuggestionReason string

const (
	SuggestionMandatory    SuggestionReason = "mandatory"
	SuggestionCapacityFill SuggestionReason = "capacity_fill"
)

type SuggestedTask struct {
	TaskID string
	Reason SuggestionReason
}

func Suggest(memberIDs []string, sprintEnd time.Time, capacityMinutes map[string]int64, candidates []Candidate) []SuggestedTask {
	ordered := slices.Clone(candidates)
	slices.SortStableFunc(ordered, func(left, right Candidate) int {
		if result := left.ExecutionEnd.Compare(right.ExecutionEnd); result != 0 {
			return result
		}
		if result := cmp.Compare(left.ProjectPriority, right.ProjectPriority); result != 0 {
			return result
		}
		if result := cmp.Compare(left.WBSOrder, right.WBSOrder); result != 0 {
			return result
		}
		return cmp.Compare(left.TaskID, right.TaskID)
	})

	result := make([]SuggestedTask, 0, len(ordered))
	for _, memberID := range uniqueIDs(memberIDs) {
		selectedAllocation := int64(0)
		for _, candidate := range ordered {
			if candidate.MemberID == memberID && !dateOnly(candidate.ExecutionEnd).After(dateOnly(sprintEnd)) {
				result = append(result, SuggestedTask{TaskID: candidate.TaskID, Reason: SuggestionMandatory})
				selectedAllocation += candidate.InSprintAllocationMinutes
			}
		}
		if selectedAllocation >= capacityMinutes[memberID] {
			continue
		}
		for _, candidate := range ordered {
			if candidate.MemberID != memberID || !dateOnly(candidate.ExecutionEnd).After(dateOnly(sprintEnd)) || candidate.InSprintAllocationMinutes <= 0 {
				continue
			}
			result = append(result, SuggestedTask{TaskID: candidate.TaskID, Reason: SuggestionCapacityFill})
			selectedAllocation += candidate.InSprintAllocationMinutes
			if selectedAllocation >= capacityMinutes[memberID] {
				break
			}
		}
	}
	return result
}
