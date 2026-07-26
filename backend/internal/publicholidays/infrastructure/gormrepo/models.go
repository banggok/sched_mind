package gormrepo

import "time"

type publicHolidayModel struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	StartDate   time.Time `gorm:"type:date;not null;index:public_holidays_list_idx,priority:1"`
	EndDate     time.Time `gorm:"type:date;not null;index:public_holidays_status_idx,priority:1;index:public_holidays_list_idx,priority:2"`
	Description string    `gorm:"size:100;not null"`
	CreatedAt   time.Time `gorm:"not null"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (publicHolidayModel) TableName() string { return "public_holidays" }

type publicHolidayDateModel struct {
	PublicHolidayID string    `gorm:"type:uuid;primaryKey;index:public_holiday_dates_parent_idx"`
	Date            time.Time `gorm:"type:date;primaryKey;uniqueIndex:public_holiday_dates_date_uidx"`
}

func (publicHolidayDateModel) TableName() string { return "public_holiday_dates" }
