package domain

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrAssigneeRecommendationInputInvalid = errors.New("assignee recommendation input is invalid")
	ErrTaskNotRecommendable               = errors.New("task is not recommendable")
	ErrAssigneeRecommendationStale        = errors.New("assignee recommendation snapshot is stale")
	ErrAssigneeRecommendationUnavailable  = errors.New("assignee recommendation is unavailable")
)

type RecommendationMode string

const (
	RecommendationAutomatic      RecommendationMode = "automatic"
	RecommendationManualAdvisory RecommendationMode = "manual-advisory"
)

type RecommendationRankGroup string

const (
	RecommendationFeasible     RecommendationRankGroup = "feasible"
	RecommendationOvercapacity RecommendationRankGroup = "overcapacity"
	RecommendationNoCompletion RecommendationRankGroup = "no-completion"
)

type AssigneeRecommendationInput struct {
	ProjectID                    string
	TaskID                       string
	RoleID                       string
	EffortMinutes                int
	LagDays                      int
	CapacityAllocationPercentage int
	ExecutionStart               *time.Time
	CalculatedOn                 time.Time
}

func (input AssigneeRecommendationInput) Validate() error {
	if strings.TrimSpace(input.ProjectID) == "" || strings.TrimSpace(input.TaskID) == "" || strings.TrimSpace(input.RoleID) == "" {
		return ErrAssigneeRecommendationInputInvalid
	}
	if input.EffortMinutes < 30 || input.EffortMinutes%30 != 0 || input.LagDays < 0 || input.CapacityAllocationPercentage < 1 || input.CapacityAllocationPercentage > 100 {
		return ErrAssigneeRecommendationInputInvalid
	}
	if input.ExecutionStart != nil && input.ExecutionStart.IsZero() {
		return ErrAssigneeRecommendationInputInvalid
	}
	if input.CalculatedOn.IsZero() {
		return ErrAssigneeRecommendationInputInvalid
	}
	return nil
}

type AssigneeRecommendationItem struct {
	MemberID                          string
	MemberName                        string
	RoleID                            string
	RankGroup                         RecommendationRankGroup
	ExecutionEnd                      *time.Time
	RemainingExecutionCapacityMinutes int
	IncrementalOvercapacityMinutes    int
	ReasonCode                        *string
}

type AssigneeRecommendationResult struct {
	CalculatedOnDate        time.Time
	ProjectScheduleVersions map[string]int64
	Mode                    RecommendationMode
	Items                   []AssigneeRecommendationItem
}

func RankAssigneeRecommendations(values []AssigneeRecommendationItem) []AssigneeRecommendationItem {
	out := append([]AssigneeRecommendationItem{}, values...)
	sort.SliceStable(out, func(left, right int) bool {
		l, r := out[left], out[right]
		if rankGroupOrder(l.RankGroup) != rankGroupOrder(r.RankGroup) {
			return rankGroupOrder(l.RankGroup) < rankGroupOrder(r.RankGroup)
		}
		if l.RankGroup != RecommendationNoCompletion && l.ExecutionEnd != nil && r.ExecutionEnd != nil && !l.ExecutionEnd.Equal(*r.ExecutionEnd) {
			return l.ExecutionEnd.Before(*r.ExecutionEnd)
		}
		if l.RankGroup != RecommendationNoCompletion && l.RemainingExecutionCapacityMinutes != r.RemainingExecutionCapacityMinutes {
			return l.RemainingExecutionCapacityMinutes > r.RemainingExecutionCapacityMinutes
		}
		leftName := strings.ToLower(strings.TrimSpace(l.MemberName))
		rightName := strings.ToLower(strings.TrimSpace(r.MemberName))
		if leftName != rightName {
			return leftName < rightName
		}
		return l.MemberID < r.MemberID
	})
	return out
}

func rankGroupOrder(value RecommendationRankGroup) int {
	switch value {
	case RecommendationFeasible:
		return 0
	case RecommendationOvercapacity:
		return 1
	default:
		return 2
	}
}
