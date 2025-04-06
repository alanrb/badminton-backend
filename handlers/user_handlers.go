package handlers

import (
	"net/http"
	"strconv"

	"github.com/alanrb/badminton/backend/auth"
	"github.com/alanrb/badminton/backend/database"
	"github.com/alanrb/badminton/backend/models"
	"github.com/alanrb/badminton/backend/models/dto"
	"github.com/alanrb/badminton/backend/rbac"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func CreateUser(c echo.Context) error {
	var user models.User
	if err := c.Bind(&user); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
	// Generate a new UUID for the session
	user.ID = uuid.New().String()

	database.DB.Create(&user)
	return c.JSON(http.StatusOK, user)
}

// @Summary Get all users
// @Description Get a paginated list of users
// @Tags users
// @Accept json
// @Produce json
// @Param page query int false "Page number (default: 1)"
// @Param limit query int false "Number of items per page (default: 10)"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/users [get]
func GetUsers(c echo.Context) error {
	// Parse query parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	limit, _ := strconv.Atoi(c.QueryParam("limit"))

	// Set default values if not provided
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10 // Default limit
	}

	// Calculate offset
	offset := (page - 1) * limit

	// Fetch paginated users
	var users []*models.User
	if err := database.DB.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch users"})
	}

	// Convert users to DTOs
	var userResponses = make([]*dto.UserResponse, 0)
	for _, user := range users {
		userResponse := dto.ToUserResponse(user)
		userResponses = append(userResponses, &userResponse)
	}

	// Get total count of users
	var total int64
	if err := database.DB.Model(&models.User{}).Count(&total).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to count users"})
	}

	// Return paginated response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data":  userResponses,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}

func GetProfile(c echo.Context) error {
	cc := c.(*auth.Context)

	ctxUser := cc.AuthUser()
	if ctxUser == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Failed to get profile",
		})
	}

	// Fetch the user from the database
	var user *models.User
	if err := database.DB.First(&user, "id = ?", cc.AuthUser().ID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	roles, err := GetRoles(database.DB, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch roles"})
	}

	permissions, err := GetPermissions(database.DB, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch permissions"})
	}

	// Return the response with user details, roles, and permissions
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":        dto.ToUserResponse(user),
		"roles":       roles,
		"permissions": permissions,
	})
}

func GetUser(c echo.Context) error {
	userID, err := GetParamID(c, "user_id")
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found."})
	}

	// Fetch the user from the database
	var user *models.User
	if err := database.DB.First(&user, "id = ?", userID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}

	roles, err := GetRoles(database.DB, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch roles"})
	}

	permissions, err := GetPermissions(database.DB, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch permissions"})
	}

	// Return the response with user details, roles, and permissions
	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":        dto.ToUserResponse(user),
		"roles":       roles,
		"permissions": permissions,
	})
}

func GetAttendedSessions(c echo.Context) error {
	cc := c.(*auth.Context)
	userID := cc.AuthUser().ID

	// Query to fetch attended sessions for the user
	var sessions []models.Session
	result := database.DB.Table("sessions").
		Joins("JOIN session_attendees ON sessions.id = session_attendees.session_id").
		Where("session_attendees.user_id = ?", userID).
		Scan(&sessions)

	if result.Error != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Failed to fetch attended sessions",
		})
	}

	// Convert users to DTOs
	var sessionResponses []dto.SessionResponse
	for _, session := range sessions {
		sessionResponses = append(sessionResponses, dto.ToSessionResponse(&session))
	}

	// Return the attended sessions
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": sessionResponses,
	})
}

func UpdateUser(c echo.Context) error {
	userID, err := GetParamID(c, "user_id")
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}
	cc := c.(*auth.Context)
	ctxUser := cc.AuthUser()
	isAdmin := IsAdmin(database.DB, ctxUser.ID)

	type updateUserRequest struct {
		AvatarURL string   `json:"avatar_url"`
		Roles     []string `json:"roles"`
	}

	// Parse the request body
	var req *updateUserRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request payload"})
	}

	// Start a transaction
	tranErr := database.RunInTransaction(database.DB, func(tx *gorm.DB) error {
		// Find the user by ID
		var user *models.User
		if err := tx.First(&user, "id = ?", userID).Error; err != nil {
			return err
		}

		// Update the user's avatar URL
		if req.AvatarURL != "" {
			user.AvatarURL = req.AvatarURL
		}

		// Validate roles if provided
		if len(req.Roles) > 0 {
			// Check if the current user has admin privileges
			if !isAdmin {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Only admins can update user roles"})
			}

			// Validate that all provided roles are valid
			for _, roleName := range req.Roles {
				var role models.Role
				if err := tx.Where("name = ?", roleName).First(&role).Error; err != nil {
					return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid role: " + roleName})
				}
			}

			// Update the user's role if roles are provided
			if len(req.Roles) > 0 && len(req.Roles) == 1 {
				// Set the primary role to the first role in the list
				user.Role = req.Roles[0] // TODO: Add priority

				// Validate that the role is valid
				if !models.ValidUserRole(user.Role) {
					return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid primary role"})
				}
			}
		}

		// Save the updated user
		if err := tx.Save(&user).Error; err != nil {
			return err
		}

		// Updates user roles
		if len(req.Roles) > 0 && ctxUser.Role == models.UserRoleAdmin {
			// Clear existing roles
			if err := tx.Model(&user).Association("Roles").Clear(); err != nil {
				return err
			}

			// Assign new roles to the user
			for _, role := range req.Roles {
				if err := rbac.AssignRoleToUser(tx, user.ID, role); err != nil {
					return err
				}
			}
		}

		return nil
	})
	if tranErr != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "User updated successfully"})
}

func GetPermissions(db *gorm.DB, userID string) ([]string, error) {
	// Fetch the user's permissions using raw SQL
	permissions := make([]string, 0)
	query := `
			SELECT p.name
			FROM permissions p
			JOIN role_permissions rp ON p.id = rp.permission_id
			JOIN roles r ON rp.role_id = r.id
			JOIN user_roles ur ON r.id = ur.role_id
			WHERE ur.user_id = ?
		`
	if err := db.Raw(query, userID).Scan(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

func GetRoles(db *gorm.DB, userID string) ([]string, error) {
	// Fetch the user's roles using raw SQL
	roles := make([]string, 0)
	rolesQuery := `
	SELECT r.name
	FROM roles r
	JOIN user_roles ur ON r.id = ur.role_id
	WHERE ur.user_id = ?
`
	if err := db.Raw(rolesQuery, userID).Scan(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// IsAdmin checks if a user has the admin role
func IsAdmin(db *gorm.DB, userID string) bool {
	var isAdmin bool
	err := db.Model(&models.UserRole{}).
		Joins("JOIN roles ON user_roles.role_id = roles.id").
		Where("user_roles.user_id = ? AND roles.name = ?", userID, models.UserRoleAdmin).
		Select("COUNT(*) > 0").
		Scan(&isAdmin).Error
	if err != nil {
		return false
	}
	return isAdmin
}
