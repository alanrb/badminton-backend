package rbac

import (
	"log"

	"github.com/alanrb/badminton/backend/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Define the PermissionName type
type PermissionName string

const (
	PermissionListUsers   PermissionName = "list_users"
	PermissionEditUsers   PermissionName = "edit_users"
	PermissionDeleteUsers PermissionName = "delete_users"
	PermissionCreateUsers PermissionName = "create_users"

	PermissionListGroups     PermissionName = "list_groups"
	PermissionCreateGroups   PermissionName = "create_groups"
	PermissionEditGroups     PermissionName = "edit_groups"
	PermissionDeleteGroups   PermissionName = "delete_groups"
	PermissionAddGroupPlayer PermissionName = "add_group_player"

	PermissionListCourts   PermissionName = "list_courts"
	PermissionCreateCourts PermissionName = "create_courts"
	PermissionEditCourts   PermissionName = "edit_courts"
	PermissionDeleteCourts PermissionName = "delete_courts"

	PermissionListSessions   PermissionName = "list_sessions"
	PermissionCreateSessions PermissionName = "create_sessions"
	PermissionDeleteSessions PermissionName = "delete_sessions"
	PermissionEditSessions   PermissionName = "edit_sessions"
)

func AssignRoleToUser(db *gorm.DB, userID string, roleName string) error {
	var role models.Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
		return err
	}

	userRole := models.UserRole{
		UserID: userID,
		RoleID: role.ID,
	}

	return db.Clauses(clause.OnConflict{DoNothing: true}).Create(&userRole).Error
}

func SeedRolesAndPermissions(db *gorm.DB) {
	// Start a new transaction
	tx := db.Begin()
	if tx.Error != nil {
		log.Fatalf("Failed to start transaction: %v", tx.Error)
	}
	// Defer a function to handle rollback in case of errors
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Fatalf("Seeding failed, transaction rolled back: %v", r)
		}
	}()

	// Create roles
	roles := []*models.Role{
		{Name: models.UserRoleAdmin},
		{Name: models.UserRoleGroupOwner},
		{Name: models.UserRolePlayer},
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&roles).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to seed roles: %v", err)
	}

	// Create permissions
	permissions := []*models.Permission{
		{Name: string(PermissionListUsers)},
		{Name: string(PermissionEditUsers)},
		{Name: string(PermissionDeleteUsers)},
		{Name: string(PermissionCreateUsers)},
		{Name: string(PermissionListGroups)},
		{Name: string(PermissionCreateGroups)},
		{Name: string(PermissionEditGroups)},
		{Name: string(PermissionDeleteGroups)},
		{Name: string(PermissionAddGroupPlayer)},
		{Name: string(PermissionListCourts)},
		{Name: string(PermissionCreateCourts)},
		{Name: string(PermissionEditCourts)},
		{Name: string(PermissionDeleteCourts)},
		{Name: string(PermissionListSessions)},
		{Name: string(PermissionCreateSessions)},
		{Name: string(PermissionDeleteSessions)},
		{Name: string(PermissionEditSessions)},
	}

	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&permissions).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to seed permissions: %v", err)
	}

	// Assign permissions to roles
	var adminRole, groupOwnerRole, playerRole *models.Role
	tx.Where("name = ?", models.UserRoleAdmin).First(&adminRole)
	tx.Where("name = ?", models.UserRoleGroupOwner).First(&groupOwnerRole)
	tx.Where("name = ?", models.UserRolePlayer).First(&playerRole)

	var listUserPerm,
		editUserPerm,
		deleteUserPerm,
		createUserPerm,
		listGroupPerm,
		createGroupPerm,
		editGroupPerm,
		deleteGroupPerm,
		addPlayerPerm,
		listCourtPerm,
		createCourtPerm,
		editCourtPerm,
		deleteCourtPerm,
		listSessionPerm,
		createSessionPerm,
		editSessionPerm,
		deleteSessionPerm *models.Permission
	tx.Where("name = ?", PermissionListUsers).First(&listUserPerm)
	tx.Where("name = ?", PermissionEditUsers).First(&editUserPerm)
	tx.Where("name = ?", PermissionDeleteUsers).First(&deleteUserPerm)
	tx.Where("name = ?", PermissionCreateUsers).First(&createUserPerm)
	tx.Where("name = ?", PermissionListGroups).First(&listGroupPerm)
	tx.Where("name = ?", PermissionCreateGroups).First(&createGroupPerm)
	tx.Where("name = ?", PermissionEditGroups).First(&editGroupPerm)
	tx.Where("name = ?", PermissionDeleteGroups).First(&deleteGroupPerm)
	tx.Where("name = ?", PermissionAddGroupPlayer).First(&addPlayerPerm)
	tx.Where("name = ?", PermissionListCourts).First(&listCourtPerm)
	tx.Where("name = ?", PermissionCreateCourts).First(&createCourtPerm)
	tx.Where("name = ?", PermissionEditCourts).First(&editCourtPerm)
	tx.Where("name = ?", PermissionDeleteCourts).First(&deleteCourtPerm)
	tx.Where("name = ?", PermissionListSessions).First(&listSessionPerm)
	tx.Where("name = ?", PermissionCreateSessions).First(&createSessionPerm)
	tx.Where("name = ?", PermissionDeleteSessions).First(&deleteSessionPerm)
	tx.Where("name = ?", PermissionEditSessions).First(&editSessionPerm)

	// Admin has all permissions
	adminRole.Permissions = append(adminRole.Permissions, []*models.Permission{
		listUserPerm,
		editUserPerm,
		deleteUserPerm,
		createUserPerm,
		listGroupPerm,
		createGroupPerm,
		editGroupPerm,
		deleteGroupPerm,
		addPlayerPerm,
		listCourtPerm,
		createCourtPerm,
		editCourtPerm,
		deleteCourtPerm,
		listSessionPerm,
		createSessionPerm,
		editSessionPerm,
		deleteSessionPerm}...)
	if err := tx.Save(&adminRole).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to assign permissions to roles: %v", err)
	}

	// if err := tx.Model(&groupOwnerRole).Association("Permissions").Append([]*models.Permission{
	// 	createGroupPerm,
	// 	listGroupPerm,
	// 	addPlayerPerm,
	// 	listCourtPerm,
	// 	listSessionPerm,
	// 	createSessionPerm,
	// 	deleteSessionPerm}); err != nil {
	// 	panic(fmt.Sprintf("Failed to assign group permissions: %v", err))
	// }

	playerRole.Permissions = append(playerRole.Permissions, []*models.Permission{
		listGroupPerm,
		listCourtPerm,
		listSessionPerm,
		createSessionPerm,
		editSessionPerm,
		deleteSessionPerm,
	}...)
	if err := tx.Save(&playerRole).Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to assign player permissions: %v", err)
	}

	// Commit the transaction if everything succeeds
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	log.Println("Roles and permissions seeded successfully")
}
