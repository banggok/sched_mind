package gormrepo

import "time"

type projectModel struct {
	ID                       string     `gorm:"type:uuid;primaryKey"`
	Name                     string     `gorm:"size:100;not null"`
	NameKey                  string     `gorm:"size:100;not null;uniqueIndex"`
	Status                   string     `gorm:"size:10;not null;index"`
	StartDate                *time.Time `gorm:"type:date"`
	EndDate                  *time.Time `gorm:"type:date"`
	AutoCalculateDate        bool       `gorm:"not null"`
	AutoDependencyByAssignee bool       `gorm:"not null"`
	AutomaticScheduling      bool       `gorm:"not null;default:true"`
	SchedulingStartDate      *time.Time `gorm:"type:date"`
	ProjectBuffer            int        `gorm:"not null;default:20"`
	Priority                 int        `gorm:"not null;uniqueIndex"`
	ClosedAt                 *time.Time
	LockedExecutionSnapshot  *string
	LockedCommitmentSnapshot *string
	CreatedAt                time.Time `gorm:"not null"`
	UpdatedAt                time.Time `gorm:"not null"`
}

func (projectModel) TableName() string { return "projects" }
