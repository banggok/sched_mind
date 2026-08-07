package gormrepo

import (
	"time"

	"gorm.io/gorm"
)

type roleModel struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:100;not null"`
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (roleModel) TableName() string {
	return "roles"
}

type teamMemberModel struct {
	ID        string         `gorm:"type:uuid;primaryKey"`
	Name      string         `gorm:"size:100;not null"`
	RoleID    string         `gorm:"type:uuid;index"`
	Role      roleModel      `gorm:"foreignKey:RoleID;references:ID;constraint:OnDelete:RESTRICT"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (teamMemberModel) TableName() string {
	return "team_members"
}

type wbsNodeModel struct {
	ID     string  `gorm:"type:uuid;primaryKey"`
	RoleID *string `gorm:"type:uuid;index"`
}

func (wbsNodeModel) TableName() string {
	return "wbs_nodes"
}
