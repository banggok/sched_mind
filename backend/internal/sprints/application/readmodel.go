package application

import "time"

type DailyValue struct {
	Date    time.Time
	Minutes int64
}

type DailySummary struct {
	Date                      time.Time
	CapacityMinutes           int64
	SelectedAllocationMinutes int64
	RemainingMinutes          int64
	OvercapacityMinutes       int64
}

type MemberProjection struct {
	ID                        string
	Name                      string
	RoleName                  string
	DailyCapacity             []DailyValue
	DailySummaries            []DailySummary
	CapacityMinutes           int64
	InSprintAllocationMinutes int64
	RemainingMinutes          int64
	OvercapacityMinutes       int64
	TotalAllocationMinutes    int64
}

type TaskProjection struct {
	ID                        string
	ProjectID                 string
	ProjectName               string
	ParentName                string
	ProjectStatus             string
	ProjectPriority           int
	Name                      string
	WBSOrder                  string
	WBSPath                   string
	WBSRank                   int
	AssigneeID                *string
	AssigneeName              *string
	EffortMinutes             *int
	ExecutionStart            *time.Time
	ExecutionEnd              *time.Time
	CommitmentStart           *time.Time
	CommitmentEnd             *time.Time
	DailyPlanOrderDate        *time.Time
	Completed                 bool
	Allocations               []DailyValue
	InSprintAllocationMinutes int64
	OutsideAllocationMinutes  int64
	TotalAllocationMinutes    int64
	Warnings                  []string
}

type Totals struct {
	CapacityMinutes                 int64
	SelectedMemberAllocationMinutes int64
	RemainingMinutes                int64
	OvercapacityMinutes             int64
	NeedsReviewAllocationMinutes    int64
	NeedsReviewDailyAllocation      []DailyValue
	AllTaskInSprintMinutes          int64
	AllTaskTotalMinutes             int64
	DailySummaries                  []DailySummary
}

type Detail struct {
	Sprint          SprintSummary
	Members         []MemberProjection
	Tasks           []TaskProjection
	Totals          Totals
	ProjectionToken string
}

type SprintSummary struct {
	ID        string
	Name      string
	StartDate time.Time
	EndDate   time.Time
	Status    string
	Version   int64
	StartedAt *time.Time
}

type SuggestionInput struct {
	StartDate time.Time
	EndDate   time.Time
	MemberIDs []string
}

type SuggestedTask struct {
	Task   TaskProjection
	Reason string
}

type Suggestion struct {
	Members         []MemberProjection
	Tasks           []SuggestedTask
	Totals          Totals
	ProjectionToken string
}
