package gormrepo

import "time"

type projectModel struct {
	ID                  string
	Status              string
	Priority            int
	AutomaticScheduling bool
	SchedulingStartDate *time.Time
	ProjectBuffer       int
	StartDate           *time.Time
	EndDate             *time.Time
	ScheduleVersion     int64
}

func (projectModel) TableName() string { return "projects" }

type taskModel struct {
	ID, ProjectID, ParentKey, Name string
	ParentID                       *string
	Position                       int
	AssigneeID                     *string
	EffortMinutes                  *int
	LagDays                        int
	ExecutionStart                 *time.Time
	ExecutionEnd                   *time.Time
	CommitmentStart                *time.Time
	CommitmentEnd                  *time.Time
	ActualEnd                      *time.Time
	ExecutionUnscheduledReason     *string
	CommitmentUnscheduledReason    *string
	CreatedAt                      time.Time
	UpdatedAt                      time.Time
}

func (taskModel) TableName() string { return "wbs_nodes" }

type memberModel struct {
	ID               string
	DailyCapacity    string
	BufferPercentage string
	DeletedAt        *time.Time
}

func (memberModel) TableName() string { return "team_members" }

type capacityOverrideModel struct {
	ID, TeamMemberID string
	StartDate        time.Time
	EndDate          time.Time
	Capacity         string
	DeletedAt        *time.Time
}

func (capacityOverrideModel) TableName() string { return "capacity_overrides" }

type holidayDateModel struct {
	PublicHolidayID string
	Date            time.Time
}

func (holidayDateModel) TableName() string { return "public_holiday_dates" }

type dependencyModel struct {
	ID, BlockingTaskID, BlockedTaskID string
	ManualOwned, AutomaticOwned       bool
	CreatedAt, UpdatedAt              time.Time
}

func (dependencyModel) TableName() string { return "task_dependencies" }

type allocationModel struct {
	TaskID                   string
	AssigneeID               string
	Timeline                 string
	AllocationDate           time.Time
	AllocatedMinutes         string
	RemainingCapacityMinutes string
	Sequence                 int
}

func (allocationModel) TableName() string { return "task_schedule_allocations" }
