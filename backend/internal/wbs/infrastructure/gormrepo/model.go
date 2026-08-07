package gormrepo

import "time"

type nodeModel struct {
	ID, ProjectID, ParentKey, Name, NameKey                      string
	ParentID                                                     *string
	Position                                                     int
	RoleID, AssigneeID                                           *string
	EffortMinutes                                                *int
	LagDays                                                      int
	CapacityAllocationPercentage                                 int `gorm:"default:100"`
	ExecutionStart, ExecutionEnd, CommitmentStart, CommitmentEnd *time.Time
	ActualStart, ActualEnd                                       *time.Time
	ExecutionUnscheduledReason, CommitmentUnscheduledReason      *string
	GroupSchedulingSource                                        string `gorm:"size:10;not null;default:inherit"`
	GroupAutomaticScheduling                                     *bool
	GroupSchedulingStartDate                                     *time.Time
	GroupLocalStatus                                             string `gorm:"size:10;not null;default:open"`
	GroupSchedulingVersion                                       int64  `gorm:"not null;default:0"`
	GroupLockedAutomaticScheduling                               *bool
	GroupLockedSchedulingStartDate                               *time.Time
	CreatedAt, UpdatedAt                                         time.Time
}

func (nodeModel) TableName() string { return "wbs_nodes" }

type projectModel struct {
	ID, Name                 string
	Status                   string
	AutomaticScheduling      bool
	SchedulingStartDate      *time.Time
	ProjectBuffer            int
	Priority                 int
	ScheduleVersion          int64
	LockedExecutionSnapshot  *string
	LockedCommitmentSnapshot *string
	UpdatedAt                time.Time
}

func (projectModel) TableName() string { return "projects" }

type memberModel struct {
	ID, RoleID string
	DeletedAt  *time.Time
}

func (memberModel) TableName() string { return "team_members" }

type dependencyLinkModel struct {
	ID, BlockingTaskID, BlockedTaskID string
}

func (dependencyLinkModel) TableName() string { return "task_dependencies" }
