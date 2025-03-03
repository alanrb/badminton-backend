package handlers

import (
	"net/http"
	"strconv"

	"github.com/alanrb/badminton/backend/auth"
	"github.com/alanrb/badminton/backend/database"
	"github.com/alanrb/badminton/backend/models"
	"github.com/alanrb/badminton/backend/models/dto"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
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
	var users []models.User
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
	var user models.User
	if err := database.DB.First(&user, "id = ?", cc.AuthUser().ID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "User not found"})
	}
	// Return user profile
	return c.JSON(http.StatusOK, dto.ToUserResponse(user))
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
