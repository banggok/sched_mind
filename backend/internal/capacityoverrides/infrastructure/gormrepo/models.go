package gormrepo

import (
	"time"

	"gorm.io/gorm"
)

type capacityOverrideModel struct {
	ID           string         `gorm:"type:uuid;primaryKey"`
	TeamMemberID string         `gorm:"type:uuid;not null;index:capacity_overrides_member_period_idx,priority:1"`
	Description  string         `gorm:"size:100;not null"`
	StartDate    time.Time      `gorm:"type:date;not null;index:capacity_overrides_member_period_idx,priority:2"`
	EndDate      time.Time      `gorm:"type:date;not null;index:capacity_overrides_member_period_idx,priority:3"`
	Capacity     string         `gorm:"type:numeric(4,1);not null"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (capacityOverrideModel) TableName() string { return "capacity_overrides" }

type teamMemberModel struct {
	ID            string         `gorm:"type:uuid;primaryKey"`
	DailyCapacity string         `gorm:"type:numeric(4,1);not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (teamMemberModel) TableName() string { return "team_members" }

type publicHolidayDateModel struct {
	Date time.Time `gorm:"type:date;primaryKey"`
}

func (publicHolidayDateModel) TableName() string { return "public_holiday_dates" }
