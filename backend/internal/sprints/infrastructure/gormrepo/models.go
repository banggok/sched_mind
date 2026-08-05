package gormrepo

import "time"

type sprintModel struct {
	ID        string    `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:200;not null"`
	NameKey   string    `gorm:"size:200;not null;uniqueIndex"`
	StartDate time.Time `gorm:"type:date;not null;index:idx_sprints_list,sort:desc"`
	EndDate   time.Time `gorm:"type:date;not null;index:idx_sprints_list,sort:desc"`
	Status    string    `gorm:"size:10;not null"`
	Version   int64     `gorm:"not null;default:1"`
	StartedAt *time.Time
	CreatedAt time.Time `gorm:"not null"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (sprintModel) TableName() string { return "sprints" }

type sprintMemberModel struct {
	SprintID string `gorm:"type:uuid;primaryKey"`
	MemberID string `gorm:"type:uuid;primaryKey;index"`
}

func (sprintMemberModel) TableName() string { return "sprint_members" }

type sprintTaskModel struct {
	SprintID string `gorm:"type:uuid;primaryKey"`
	TaskID   string `gorm:"type:uuid;primaryKey;index"`
}

func (sprintTaskModel) TableName() string { return "sprint_tasks" }

type memberModel struct {
	ID               string `gorm:"type:uuid;primaryKey"`
	Name             string
	RoleID           string
	DailyCapacity    string
	BufferPercentage string
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}

func (memberModel) TableName() string { return "team_members" }
