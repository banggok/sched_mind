package gormrepo

import "gorm.io/gorm"

func MigrateTestSchema(database *gorm.DB) error {
	return database.Transaction(func(transaction *gorm.DB) error {
		if err := transaction.AutoMigrate(&roleModel{}, &teamMemberModel{}, &wbsNodeModel{}); err != nil {
			return err
		}
		if err := transaction.Exec(
			"CREATE UNIQUE INDEX IF NOT EXISTS roles_name_ci_unique ON roles (LOWER(name))",
		).Error; err != nil {
			return err
		}
		return transaction.Exec("PRAGMA foreign_keys = ON").Error
	})
}

func AssignRoleForTest(database *gorm.DB, teamMemberID, roleID string) error {
	return database.Create(&teamMemberModel{ID: teamMemberID, Name: teamMemberID, RoleID: roleID}).Error
}
