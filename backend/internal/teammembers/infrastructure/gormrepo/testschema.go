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
			&assignmentProjectModel{},
			&assignmentWBSNodeModel{},
			&capacityOverrideModel{},
			&sprintMemberModel{},
		)
	})
}

func CreateRoleForTest(database *gorm.DB, id, name string) error {
	return database.Create(&roleModel{ID: id, Name: name}).Error
}

func AssignTaskForTest(database *gorm.DB, id, memberID string) error {
	if err := database.FirstOrCreate(&assignmentProjectModel{ID: "assignment-project", Status: "open"}, "id = ?", "assignment-project").Error; err != nil {
		return err
	}
	return database.Create(&assignmentWBSNodeModel{ID: id, ProjectID: "assignment-project", AssigneeID: &memberID}).Error
}

func CreateOverrideForTest(database *gorm.DB, id, memberID string) error {
	return database.Create(
		&capacityOverrideModel{ID: id, TeamMemberID: memberID},
	).Error
}
