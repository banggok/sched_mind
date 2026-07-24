package gormrepo

import "time"

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
	ID     string    `gorm:"type:uuid;primaryKey"`
	RoleID string    `gorm:"type:uuid;not null;index"`
	Role   roleModel `gorm:"foreignKey:RoleID;references:ID;constraint:OnDelete:RESTRICT"`
}

func (teamMemberModel) TableName() string {
	return "team_members"
}
