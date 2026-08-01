package gormrepo

import (
	"encoding/json"
	"time"
)

type savedFilterModel struct {
	ID         string
	Name       string
	NameKey    string
	ProjectIDs json.RawMessage `gorm:"column:project_ids;type:jsonb"`
	Version    int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (savedFilterModel) TableName() string { return "portfolio_saved_filters" }
