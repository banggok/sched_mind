package gormrepo

import "time"

type nodeModel struct {
	ID, ProjectID, ParentKey, Name, NameKey                                 string
	ParentID                                                                *string
	Position                                                                int
	RoleID, AssigneeID                                                      *string
	EffortMinutes                                                           *int
	LagDays                                                                 int
	ExecutionStart, ExecutionEnd, CommitmentStart, CommitmentEnd, ActualEnd *time.Time
	ExecutionUnscheduledReason, CommitmentUnscheduledReason                 *string
	CreatedAt, UpdatedAt                                                    time.Time
}

func (nodeModel) TableName() string { return "wbs_nodes" }

type projectModel struct {
	ID, Status          string
	AutomaticScheduling bool
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
