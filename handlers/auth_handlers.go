package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/alanrb/badminton/backend/auth"
	"github.com/alanrb/badminton/backend/database"
	"github.com/alanrb/badminton/backend/models"
	"github.com/alanrb/badminton/backend/models/dto"
	"github.com/labstack/echo/v4"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

func HandleGoogleLogin(c echo.Context, cfg *oauth2.Config) error {
	url := cfg.AuthCodeURL("state")
	if err := c.Redirect(http.StatusTemporaryRedirect, url); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to redirect to Google login: %v", err)})
	}
	return nil
}

func HandleGoogleCallback(c echo.Context, jwtSecret interface{}, websiteURL string, cfg *oauth2.Config) error {
	code := c.QueryParam("code")

	token, err := cfg.Exchange(c.Request().Context(), code)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to exchange token"})
	}

	// Fetch user info from Google
	client := cfg.Client(c.Request().Context(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch user info"})
	}
	defer resp.Body.Close()

	var userInfo struct {
		Email string `json:"email"`
		Name  string `json:"name"`
		ID    string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to decode user info"})
	}

	// Check if the user already exists in the database
	var user models.User
	if err := database.DB.Where("google_id = ?", userInfo.ID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// If the user doesn't exist, create a new user
			user = models.User{
				GoogleID: userInfo.ID,
				Email:    userInfo.Email,
				Name:     userInfo.Name,
				Role:     models.UserRolePlayer, // Default role
			}
			database.DB.Create(&user)
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch user"})
		}
	}

	// Generate a JWT token for the user
	jwtToken, err := auth.GenerateJWTToken(user, jwtSecret)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to generate JWT token"})
	}

	// Redirect to the frontend with the JWT token
	frontendURL := fmt.Sprintf("%v/login?token=%s", websiteURL, jwtToken)
	return c.Redirect(http.StatusTemporaryRedirect, frontendURL)
}

func HandleCognitoUser(c echo.Context) error {
	cc := c.(*auth.Context)
	var userInfo struct {
		ID      string `json:"id"`
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := c.Bind(&userInfo); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
	}

	if cc.AuthUser().ID != userInfo.ID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to fetch user."})
	}

	// Check if the user already exists in the database
	var user *models.User
	if err := database.DB.Where("email = ?", userInfo.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// If the user doesn't exist, create a new user
			user = &models.User{
				BaseModel: models.BaseModel{
					ID: userInfo.ID,
				},
				GoogleID:  userInfo.Email,
				Email:     userInfo.Email,
				Name:      userInfo.Name,
				AvatarURL: userInfo.Picture,
				Role:      cc.AuthUser().Role,
			}

			if err := database.DB.Create(&user).Error; err != nil {
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to init user: %v", err)})
			}
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch user"})
		}
	}

	if err := database.DB.Model(&user).Updates(map[string]interface{}{"id": userInfo.ID, "name": userInfo.Name, "avatar_url": userInfo.Picture}).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update user"})
	}

	permissions, err := GetPermissions(database.DB, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch permissions"})
	}

	roles, err := GetRoles(database.DB, user.ID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch roles"})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user":        dto.ToUserResponse(user),
		"permissions": permissions,
		"roles":       roles,
	})
}
