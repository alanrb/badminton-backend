package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/alanrb/badminton/backend/auth"
	"github.com/alanrb/badminton/backend/database"
	"github.com/alanrb/badminton/backend/models"
	"github.com/alanrb/badminton/backend/models/dto"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func CreateSession(c echo.Context) error {
	var request dto.NewSessionRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	cc, ok := c.(*auth.Context)
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get authentication context"})
	}

	session := models.Session{
		CreatedBy:   cc.AuthUser().ID,
		Description: request.Description,
		Status:      models.SessionStatusOpen,
		MaxMembers:  request.MaxMembers,
	}

	if len(request.BadmintonCourtID) > 0 {
		// Validate BadmintonCourtID
		if err := uuid.Validate(request.BadmintonCourtID); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid BadmintonCourtID"})
		}

		var court models.BadmintonCourt
		if err := database.DB.First(&court, "id = ?", request.BadmintonCourtID).Error; err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Badminton Court not found"})
		}

		session.BadmintonCourtID = &request.BadmintonCourtID
	}

	if len(request.GroupID) > 0 {
		// Safety check
		if err := uuid.Validate(request.GroupID); err != nil {
			return errors.New("invalid group ID")
		}

		// Validate the group ID
		var group *models.Group
		if err := database.DB.First(&group, "id = ?", request.GroupID).Error; err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Group not found"})
		}

		// Check if the user is a member of the group
		isMember, err := IsGroupMember(database.DB, group.ID, cc.AuthUser().ID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check group membership"})
		}

		if !isMember && cc.AuthUser().Role != models.UserRoleAdmin {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Only group members can create sessions for the group"})
		}

		// Associate the session with the group
		session.GroupID = &group.ID
	}

	if request.DateTime == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid DateTime"})
	}
	session.DateTime = request.DateTime

	// Generate a new UUID for the session
	session.ID = uuid.New().String()

	database.DB.Create(&session)
	return c.JSON(http.StatusOK, dto.ToSessionResponse(&session))
}

func AttendSession(c echo.Context) error {
	// Parse session ID from the request
	sessionID, err := getSessionID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	// Parse the request body
	var req dto.AttendSessionRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request",
		})
	}

	if req.Slot == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid Slot",
		})
	}

	cc := c.(*auth.Context)
	userID := cc.AuthUser().ID

	// Start a transaction
	tx := database.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Fetch the session from the database
	var session models.Session
	if err := tx.First(&session, "id = ?", sessionID).Error; err != nil {
		tx.Rollback()
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Session not found"})
	}

	// Validate session status
	if !session.CanAttend() {
		tx.Rollback()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Session is not open for attendance"})
	}

	// Check if the user is already attending the session
	var existingAttendee models.SessionAttendee
	if err := tx.Where("session_id = ? AND user_id = ?", sessionID, userID).First(&existingAttendee).Error; err == nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			tx.Rollback()
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "User is already attending this session"})
		}
	}

	// Calculate the total slots already occupied
	var totalSlots int64
	if err := tx.Model(&models.SessionAttendee{}).
		Where("session_id = ? AND status = ?", sessionID, models.ApprovalStatusApproved).
		Select("COALESCE(SUM(slot), 0)").
		Scan(&totalSlots).Error; err != nil {
		tx.Rollback()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Failed to calculate total slots: %v", err)})
	}

	// Check if adding the new slots exceeds MaxMembers
	if totalSlots+int64(req.Slot) > int64(session.MaxMembers) {
		tx.Rollback()
		return c.JSON(http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("Session is full: max members is %v", session.MaxMembers)})
	}

	// Add the attendee to the session
	attendee := models.SessionAttendee{
		SessionID: sessionID,
		UserID:    userID,
		Status:    models.ApprovalStatusApproved, // Default status
		Slot:      req.Slot,
	}
	if err := tx.Create(&attendee).Error; err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to attend session"})
	}

	// Commit the transaction
	if err := tx.Commit().Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to commit transaction: %v", err)})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Successfully attended the session"})
}

func CancelAttendance(c echo.Context) error {
	// Parse session ID from the request
	sessionID := c.Param("session_id")

	if err := uuid.Validate(sessionID); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid session ID"})
	}

	cc := c.(*auth.Context)
	userID := cc.AuthUser().ID

	// Delete the attendee from the session
	result := database.DB.Where("session_id = ? AND user_id = ?", sessionID, userID).Delete(&models.SessionAttendee{})
	if result.Error != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to cancel attendance",
		})
	}

	if result.RowsAffected == 0 {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "Attendance record not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Successfully canceled attendance",
	})
}

func GetSessionDetails(c echo.Context) error {
	// Parse session ID from the request
	sessionID, err := getSessionID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var session models.Session
	result := database.DB.Preload("Group").First(&session, "id = ?", sessionID) // Query the database
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Session not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch session details"})
	}

	return c.JSON(http.StatusOK, dto.ToSessionResponse(&session)) // Return the session details as JSON
}

func GetSessions(c echo.Context) error {
	var sessions []models.Session
	currentTime := time.Now()
	database.DB.Table("sessions").
		Select("sessions.*, users.name as created_by_name").
		Joins("left join users on users.id = sessions.created_by").
		Where("sessions.date_time > ?", currentTime).
		Preload("BadmintonCourt").
		Preload("Attendees.User").
		Preload("Group").
		Find(&sessions)

	// Convert users to DTOs
	var sessionResponses []dto.SessionResponse
	for _, session := range sessions {
		sessionResponses = append(sessionResponses, dto.ToSessionResponse(&session))
	}

	// Return paginated response
	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": sessionResponses,
	})
}

func UpdateSession(c echo.Context) error {
	sessionID, err := getSessionID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var session models.Session
	if err := database.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Session not found"})
	}

	if session.Status != models.SessionStatusOpen {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Session is not open for update"})
	}

	cc := c.(*auth.Context)
	if session.CreatedBy != cc.AuthUser().ID && cc.AuthUser().Role != models.UserRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You are not the creator of this session"})
	}

	var request dto.UpdateSessionRequest
	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if request.DateTime != nil {
		if request.DateTime.Before(time.Now()) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid DateTime"})
		}
		session.DateTime = request.DateTime
	}

	if len(request.BadmintonCourtID) > 0 {
		// Validate BadmintonCourtID
		if err := uuid.Validate(request.BadmintonCourtID); err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid BadmintonCourtID"})
		}

		var court models.BadmintonCourt
		if err := database.DB.First(&court, "id = ?", request.BadmintonCourtID).Error; err != nil {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Badminton Court not found"})
		}

		session.BadmintonCourtID = &request.BadmintonCourtID
	}

	// Update session details
	session.Description = request.Description
	session.MaxMembers = request.MaxMembers

	if err := database.DB.Save(&session).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update session"})
	}

	return c.JSON(http.StatusOK, dto.ToSessionResponse(&session))
}

// @Summary Update session status
// @Description Update the status of a session (open, on-going, completed)
// @Tags sessions
// @Accept json
// @Produce json
// @Param id path string true "Session ID"
// @Param status body string true "New status (open, on-going, completed)"
// @Security ApiKeyAuth
// @Success 200 {object} models.Session
// @Router /api/sessions/{id}/status [put]
func UpdateSessionStatus(c echo.Context) error {
	sessionID, err := getSessionID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var session models.Session
	if err := database.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Session not found"})
	}

	var updateData struct {
		Status string `json:"status"`
	}
	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if updateData.Status == session.Status {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Session status is already " + updateData.Status})
	}

	if updateData.Status != models.SessionStatusOpen && updateData.Status != models.SessionStatusOngoing && updateData.Status != models.SessionStatusCompleted {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid session status"})
	}

	// Validate session status
	if !models.ValidSessionStatus(updateData.Status) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid session status"})
	}

	// Update session status
	session.Status = updateData.Status
	database.DB.Save(&session)

	return c.JSON(http.StatusOK, dto.ToSessionResponse(&session))
}

func DeleteSession(c echo.Context) error {
	sessionID, err := getSessionID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	var session models.Session
	if err := database.DB.First(&session, "id = ?", sessionID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Session not found"})
	}

	cc := c.(*auth.Context)
	if session.CreatedBy != cc.AuthUser().ID && cc.AuthUser().Role != models.UserRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "You are not the creator of this session"})
	}

	database.DB.Where("id = ?", sessionID).Delete(&models.Session{})
	return c.JSON(http.StatusOK, map[string]string{"message": "Session deleted"})
}

func getSessionID(c echo.Context) (string, error) {
	sessionID := c.Param("session_id")
	if err := uuid.Validate(sessionID); err != nil {
		return "", errors.New("invalid session ID")
	}
	return sessionID, nil
}
