package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/banggok/sched_mind/backend/internal/shared/persistence"
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
	return MigrateWithPostStep(database, nil)
}

func MigrateWithPostStep(database *gorm.DB, postOwnershipRemoval func(context.Context) error) error {
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
			if entry.Name() == "000023_remove_automatic_dependency_ownership.up.sql" && postOwnershipRemoval != nil {
				ctx, release := persistence.SerializeScheduleMutation(context.Background())
				defer release()
				if err := postOwnershipRemoval(persistence.WithTransaction(ctx, transaction)); err != nil {
					return fmt.Errorf("recalculate active portfolio after dependency ownership removal: %w", err)
				}
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
