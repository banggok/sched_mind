package gormrepo

import (
	"time"

	"gorm.io/gorm"
)

type teamMemberModel struct {
	ID               string         `gorm:"type:uuid;primaryKey"`
	Name             string         `gorm:"size:100;not null"`
	RoleID           string         `gorm:"type:uuid;not null;index"`
	Role             roleModel      `gorm:"foreignKey:RoleID;references:ID"`
	DailyCapacity    string         `gorm:"type:numeric(4,1);not null"`
	BufferPercentage string         `gorm:"type:numeric(5,2);not null"`
	CreatedAt        time.Time      `gorm:"not null"`
	UpdatedAt        time.Time      `gorm:"not null"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (teamMemberModel) TableName() string {
	return "team_members"
}

type roleModel struct {
	ID   string `gorm:"type:uuid;primaryKey"`
	Name string `gorm:"size:100;not null"`
}

func (roleModel) TableName() string {
	return "roles"
}

type assignmentProjectModel struct {
	ID     string `gorm:"type:uuid;primaryKey"`
	Status string `gorm:"not null"`
}

func (assignmentProjectModel) TableName() string { return "projects" }

type assignmentWBSNodeModel struct {
	ID         string `gorm:"type:uuid;primaryKey"`
	ProjectID  string `gorm:"type:uuid;not null;index"`
	ParentID   *string
	AssigneeID *string `gorm:"type:uuid;index"`
}

func (assignmentWBSNodeModel) TableName() string { return "wbs_nodes" }

type capacityOverrideModel struct {
	ID           string         `gorm:"type:uuid;primaryKey"`
	TeamMemberID string         `gorm:"type:uuid;not null;index"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (capacityOverrideModel) TableName() string {
	return "capacity_overrides"
}
