package middleware

import (
	"net/http"

	"github.com/alanrb/badminton/backend/auth"
	"github.com/alanrb/badminton/backend/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func RBAC(db *gorm.DB, permission string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get the authenticated user's ID
			cc := c.(*auth.Context)
			userID := cc.AuthUser().ID

			// Check if the user has the required permission
			var count int64
			err := db.Model(&models.UserRole{}).
				Joins("JOIN role_permissions ON user_roles.role_id = role_permissions.role_id").
				Joins("JOIN permissions ON role_permissions.permission_id = permissions.id").
				Where("user_roles.user_id = ? AND permissions.name = ?", userID, permission).
				Count(&count).Error

			if err != nil || count == 0 {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "You do not have permission to perform this action"})
			}

			return next(c)
		}
	}
}
