package gormrepo

import "gorm.io/gorm"

func MigrateTestSchema(database *gorm.DB) error {
	return database.Transaction(func(transaction *gorm.DB) error {
		if err := transaction.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			return err
		}
		return transaction.AutoMigrate(
			&roleModel{},
			&teamMemberModel{},
			&executableLeafModel{},
			&capacityOverrideModel{},
		)
	})
}

func CreateRoleForTest(database *gorm.DB, id, name string) error {
	return database.Create(&roleModel{ID: id, Name: name}).Error
}

func AssignTaskForTest(database *gorm.DB, id, memberID string) error {
	return database.Create(
		&executableLeafModel{ID: id, AssigneeID: memberID},
	).Error
}

func CreateOverrideForTest(database *gorm.DB, id, memberID string) error {
	return database.Create(
		&capacityOverrideModel{ID: id, TeamMemberID: memberID},
	).Error
}
