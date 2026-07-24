package postgres

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"gorm.io/driver/postgres"
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

func Open(databaseURL string) (*gorm.DB, error) {
	database, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open PostgreSQL: %w", err)
	}
	return database, nil
}

func Migrate(database *gorm.DB) error {
	if err := database.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`).Error; err != nil {
		return fmt.Errorf("create schema migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
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
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if count > 0 {
			continue
		}

		content, err := migrations.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		if err := database.Transaction(func(transaction *gorm.DB) error {
			if err := transaction.Exec(string(content)).Error; err != nil {
				return err
			}
			return transaction.Create(
				&schemaMigration{Version: entry.Name()},
			).Error
		}); err != nil {
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}
