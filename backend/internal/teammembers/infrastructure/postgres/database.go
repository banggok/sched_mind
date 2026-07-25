package postgres

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gorm.io/gorm"
)

//go:embed migrations/*.up.sql
var migrations embed.FS

type schemaMigration struct {
	Version string `gorm:"primaryKey"`
}

func (schemaMigration) TableName() string {
	return "schema_migrations"
}

func Migrate(database *gorm.DB) error {
	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read team member migrations: %w", err)
	}
	sort.Slice(entries, func(left, right int) bool {
		return entries[left].Name() < entries[right].Name()
	})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		var count int64
		if err := database.Model(&schemaMigration{}).
			Where("version = ?", entry.Name()).
			Count(&count).Error; err != nil {
			return fmt.Errorf("check team member migration %s: %w", entry.Name(), err)
		}
		if count > 0 {
			continue
		}
		content, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read team member migration %s: %w", entry.Name(), err)
		}
		if err := database.Transaction(func(transaction *gorm.DB) error {
			if err := transaction.Exec(string(content)).Error; err != nil {
				return err
			}
			return transaction.Create(
				&schemaMigration{Version: entry.Name()},
			).Error
		}); err != nil {
			return fmt.Errorf("apply team member migration %s: %w", entry.Name(), err)
		}
	}
	return nil
}
