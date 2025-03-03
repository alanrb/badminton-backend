package handlers

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/alanrb/badminton/backend/database"
	"github.com/alanrb/badminton/backend/models"
	"github.com/alanrb/badminton/backend/models/dto"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/shopspring/decimal"
)

// CreateBadmintonCourt creates a new badminton court
func CreateBadmintonCourt(c echo.Context) error {
	var court models.BadmintonCourt
	if err := c.Bind(&court); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if len(court.GoogleMapURL) > 0 {
		// Validate Google Map URL
		if !validateURL(court.GoogleMapURL) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid Google Map URL"})
		}
	}

	// Validate EstimatePricePerHour
	if court.EstimatePricePerHour.IsNegative() {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid price"})
	}

	database.DB.Create(&court)
	return c.JSON(http.StatusOK, dto.ToBadmintonCourtResponse(court))
}

// GetBadmintonCourt fetch all badminton courts
func GetBadmintonCourts(c echo.Context) error {
	var courts []models.BadmintonCourt
	if err := database.DB.Find(&courts).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch badminton courts"})
	}

	// Convert courts to DTOs
	var courtResponses []dto.BadmintonCourtResponse
	for _, court := range courts {
		courtResponses = append(courtResponses, dto.ToBadmintonCourtResponse(court))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"data": courtResponses,
	})
}

// GetBadmintonCourt retrieves a badminton court by ID
func GetBadmintonCourt(c echo.Context) error {
	courtID, err := getCourtID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid court ID"})
	}

	var court models.BadmintonCourt
	if err := database.DB.First(&court, "id = ?", courtID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Court not found"})
	}

	return c.JSON(http.StatusOK, dto.ToBadmintonCourtResponse(court))
}

// UpdateBadmintonCourt updates a badminton court
func UpdateBadmintonCourt(c echo.Context) error {
	var updateData struct {
		Name                 string          `json:"name"`
		Address              string          `json:"address"`
		GoogleMapURL         string          `json:"google_map_url"`
		EstimatePricePerHour decimal.Decimal `json:"estimate_price_per_hour"`
		Contact              string          `json:"contact"`
	}
	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	courtID, err := getCourtID(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid court ID"})
	}

	var court models.BadmintonCourt
	if err := database.DB.First(&court, "id = ?", courtID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Court not found"})
	}

	if len(updateData.GoogleMapURL) > 0 {
		// Validate Google Map URL
		if !validateURL(updateData.GoogleMapURL) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid Google Map URL"})
		}
		court.GoogleMapURL = updateData.GoogleMapURL
	}

	// Validate EstimatePricePerHour
	if updateData.EstimatePricePerHour.IsNegative() {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid price"})
	}

	// Update fields
	court.Name = updateData.Name
	court.Address = updateData.Address
	court.EstimatePricePerHour = updateData.EstimatePricePerHour
	court.Contact = updateData.Contact

	database.DB.Save(&court)
	return c.JSON(http.StatusOK, dto.ToBadmintonCourtResponse(court))
}

// DeleteBadmintonCourt deletes a badminton court
func DeleteBadmintonCourt(c echo.Context) error {
	courtID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid court ID"})
	}

	var court models.BadmintonCourt
	if err := database.DB.First(&court, "id = ?", courtID).Error; err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Court not found"})
	}

	database.DB.Delete(&court)
	return c.JSON(http.StatusOK, map[string]string{"message": "Court deleted"})
}

// ValidateURL checks if a string is a valid URL
func validateURL(urlString string) bool {
	_, err := url.ParseRequestURI(urlString)
	return err == nil
}

func getCourtID(c echo.Context) (string, error) {
	courtID := c.Param("id")
	if err := uuid.Validate(courtID); err != nil {
		return "", errors.New("invalid court ID")
	}
	return courtID, nil
}
