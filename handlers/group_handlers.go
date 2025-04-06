package handlers

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/alanrb/badminton/backend/auth"
	"github.com/alanrb/badminton/backend/database"
	"github.com/alanrb/badminton/backend/models"
	"github.com/alanrb/badminton/backend/models/dto"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func CreateGroup(c echo.Context) error {
	var request dto.NewGroupRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	cc := c.(*auth.Context)

	if len(request.Name) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid name"})
	}

	group := models.Group{
		OwnerID:  cc.AuthUser().ID,
		Name:     request.Name,
		ImageUrl: request.ImageUrl,
		Remark:   request.Remark,
	}

	if err := database.RunInTransaction(database.DB, func(tx *gorm.DB) error {

		// Save the group to the database
		if err := tx.Create(&group).Error; err != nil {
			return err
		}

		// Add the player to the group
		groupMember := models.GroupMember{
			GroupID: group.ID,
			UserID:  group.OwnerID,
		}

		if err := tx.Create(&groupMember).Error; err != nil {
			return err
		}
		return nil
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create group"})
	}

	return c.JSON(http.StatusCreated, group)
}

func ListGroups(c echo.Context) error {
	cc := c.(*auth.Context)
	var userID = cc.AuthUser().ID

	// Get pagination parameters using the utility
	pagination := database.GetPagination(c.QueryParam("page"), c.QueryParam("limit"))

	// Check if the user is an admin
	var isAdmin bool
	err := database.DB.Model(&models.UserRole{}).
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name = ?", userID, "admin").
		Select("COUNT(*) > 0").
		Scan(&isAdmin).Error
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check admin role"})
	}

	var groups []*models.Group
	var total int64

	if isAdmin {
		// If the user is an admin, fetch all groups with pagination
		if err := database.DB.Model(&models.Group{}).Count(&total).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch total groups"})
		}
		if err := database.DB.Offset(pagination.Offset).Limit(pagination.PageSize).Preload("Members").Find(&groups).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch groups"})
		}
	} else {
		// If the user is not an admin, fetch groups where the user is a member or the owner with pagination
		if err := database.DB.Model(&models.Group{}).
			Joins("LEFT JOIN group_members ON groups.id = group_members.group_id").
			Where("groups.owner_id = ? OR group_members.user_id = ?", userID, userID).
			Count(&total).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch total groups"})
		}
		if err := database.DB.Model(&models.Group{}).
			Joins("LEFT JOIN group_members ON groups.id = group_members.group_id").
			Where("groups.owner_id = ? OR group_members.user_id = ?", userID, userID).
			Offset(pagination.Offset).Limit(pagination.PageSize).Preload("Members").Find(&groups).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch groups"})
		}
	}

	// Convert users to DTOs
	var groupResponses []*dto.GroupResponse
	for _, group := range groups {
		groupResponses = append(groupResponses, dto.ToGroupResponse(group))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":      groupResponses,
		"total":     total,
		"page":      pagination.Page,
		"page_size": pagination.PageSize,
	})
}

func AddPlayerToGroup(c echo.Context) error {
	groupID, err := getGroupID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var request struct {
		UserEmail string `json:"user_email"`
	}
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if len(request.UserEmail) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email is invalid"})
	}

	if !isValidEmail(request.UserEmail) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Email is invalid"})
	}

	// Check if the user making the request is the group owner
	var group *models.Group
	if err := database.DB.First(&group, "id = ?", groupID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Group not found"})
	}

	cc := c.(*auth.Context)
	if group.OwnerID != cc.AuthUser().ID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "Only the group owner can add players"})
	}

	// Find the user by email
	var user *models.User
	if err := database.DB.Where("email = ?", request.UserEmail).First(&user).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	// Check if the user is already a member of the group
	isMember, err := IsGroupMember(database.DB, groupID, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check group membership"})
	}
	if isMember {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "User is already a member of the group"})
	}

	// Add the player to the group
	groupMember := models.GroupMember{
		GroupID: groupID,
		UserID:  user.ID,
	}

	if err := database.DB.Create(&groupMember).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add player to group"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Player added to group"})
}

func DeleteGroup(c echo.Context) error {
	// Get the group ID from the URL
	groupID, err := getGroupID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Check if the user has permission to delete the group
	cc := c.(*auth.Context)

	userID := cc.AuthUser().ID
	var group models.Group
	if err := database.DB.Where("id = ?", groupID).First(&group).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Group not found"})
	}

	// Only the group owner or admin can delete the group
	if group.OwnerID != userID && cc.AuthUser().Role != models.UserRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have permission to delete this group"})
	}

	// Delete the group
	if err := database.DB.Delete(&group).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete group"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Group deleted successfully"})
}

func GetGroupDetails(c echo.Context) error {
	// Get the group ID from the URL
	groupID, err := getGroupID(c)
	if err != nil {
		return err
	}

	cc := c.(*auth.Context)

	// Check if the user is a member of the group
	isMember, err := IsGroupMember(database.DB, groupID, cc.AuthUser().ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check group membership"})
	}

	if !isMember && cc.AuthUser().Role != models.UserRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You are not a member of this group"})
	}

	// Fetch the group from the database
	var group *models.Group
	if err := database.DB.Preload("Members").First(&group, "id = ?", groupID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Group not found"})
	}

	return c.JSON(http.StatusOK, dto.ToGroupResponse(group))
}

// IsGroupMember checks if a user is a member of a group
func IsGroupMember(db *gorm.DB, groupID string, userID string) (bool, error) {
	var count int64
	err := db.Model(&models.GroupMember{}).
		Where("group_id = ? AND user_id = ?", groupID, userID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func getGroupID(c echo.Context) (string, error) {
	groupID := c.Param("group_id")
	if err := uuid.Validate(groupID); err != nil {
		return "", errors.New("invalid group ID")
	}
	return groupID, nil
}

func isValidEmail(email string) bool {
	// Regular expression for basic email validation
	// This is a simple regex and may not cover all edge cases
	emailRegex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	return regexp.MustCompile(emailRegex).MatchString(email)
}
