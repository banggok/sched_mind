package domain

import (
	"errors"
	"testing"
	"time"
)

func TestAssigneeRecommendationInputValidate_D03_AC3(t *testing.T) {
	valid := AssigneeRecommendationInput{
		ProjectID:                    "project",
		TaskID:                       "task",
		RoleID:                       "role",
		EffortMinutes:                480,
		LagDays:                      0,
		CapacityAllocationPercentage: 40,
		CalculatedOn:                 time.Date(2026, 8, 5, 0, 0, 0, 0, time.UTC),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid input error=%v", err)
	}

	cases := map[string]func(*AssigneeRecommendationInput){
		"missing role":       func(value *AssigneeRecommendationInput) { value.RoleID = "" },
		"invalid effort":     func(value *AssigneeRecommendationInput) { value.EffortMinutes = 45 },
		"negative lag":       func(value *AssigneeRecommendationInput) { value.LagDays = -1 },
		"zero percentage":    func(value *AssigneeRecommendationInput) { value.CapacityAllocationPercentage = 0 },
		"missing calculated": func(value *AssigneeRecommendationInput) { value.CalculatedOn = time.Time{} },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if err := candidate.Validate(); !errors.Is(err, ErrAssigneeRecommendationInputInvalid) {
				t.Fatalf("error=%v, want input invalid", err)
			}
		})
	}
}

func TestRankAssigneeRecommendations_D10_D11_D12_D13_AC16_AC17_AC18_AC20_AC21_AC22_AC23(t *testing.T) {
	date10 := time.Date(2026, 8, 10, 0, 0, 0, 0, time.UTC)
	date11 := time.Date(2026, 8, 11, 0, 0, 0, 0, time.UTC)
	values := []AssigneeRecommendationItem{
		{MemberID: "z", MemberName: "Zulu", RankGroup: RecommendationNoCompletion},
		{MemberID: "over", MemberName: "Alif", RankGroup: RecommendationOvercapacity, ExecutionEnd: &date10, RemainingExecutionCapacityMinutes: 240, IncrementalOvercapacityMinutes: 30},
		{MemberID: "later", MemberName: "Dewi", RankGroup: RecommendationFeasible, ExecutionEnd: &date11, RemainingExecutionCapacityMinutes: 300},
		{MemberID: "same-b", MemberName: "Tony", RankGroup: RecommendationFeasible, ExecutionEnd: &date10, RemainingExecutionCapacityMinutes: 120},
		{MemberID: "same-a", MemberName: "tony", RankGroup: RecommendationFeasible, ExecutionEnd: &date10, RemainingExecutionCapacityMinutes: 240},
		{MemberID: "a", MemberName: "Alex", RankGroup: RecommendationNoCompletion},
	}

	ranked := RankAssigneeRecommendations(values)
	got := make([]string, 0, len(ranked))
	for _, value := range ranked {
		got = append(got, value.MemberID)
	}
	want := []string{"same-a", "same-b", "later", "over", "a", "z"}
	if len(got) != len(want) {
		t.Fatalf("ranked IDs=%v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("ranked IDs=%v, want %v", got, want)
		}
	}

	duplicateDate := date10
	duplicates := RankAssigneeRecommendations([]AssigneeRecommendationItem{
		{MemberID: "b", MemberName: "Same", RankGroup: RecommendationFeasible, ExecutionEnd: &duplicateDate},
		{MemberID: "a", MemberName: "same", RankGroup: RecommendationFeasible, ExecutionEnd: &duplicateDate},
	})
	if duplicates[0].MemberID != "a" || duplicates[1].MemberID != "b" {
		t.Fatalf("duplicate-name order=%v, want stable ID ascending", duplicates)
	}
}
