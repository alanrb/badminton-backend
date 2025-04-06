package middleware

import (
	"net/http"
	"os"
	"strings"

	"github.com/alanrb/badminton/backend/auth"
	"github.com/alanrb/badminton/backend/database"
	"github.com/alanrb/badminton/backend/models"
	"github.com/golang-jwt/jwt/v5"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

// AdminOnly middleware to check if the user is an admin
func AdminOnly(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc, ok := c.(*auth.Context)
		if !ok {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Invalid context type"})
		}

		authUser := cc.AuthUser()
		if authUser == nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "User not found"})
		}

		// Early return if the user role is already known not to be admin
		if authUser.Role != models.UserRoleAdmin {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Admin access required"})
		}

		// Double-check admin status in the database
		var isAdmin bool
		err := database.DB.Model(&models.UserRole{}).
			Joins("JOIN roles ON user_roles.role_id = roles.id").
			Where("user_roles.user_id = ? AND roles.name = ?", authUser.ID, models.UserRoleAdmin).
			Select("COUNT(*) > 0").
			Scan(&isAdmin).Error
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Admin access required"})
		}

		if !isAdmin {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "Admin access required"})
		}

		return next(c)
	}
}

func Context(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		cc := &auth.Context{Context: c}

		// Get user from AWS Cognito token if available
		authorization := c.Request().Header.Get("CognitoAuthorization")
		if len(authorization) > 0 {
			// Parse JWT token from header and extract user ID
			jwtString := strings.Split(authorization, "Bearer ")[1]
			claims := jwt.MapClaims{}
			_, _, err := jwt.NewParser().ParseUnverified(jwtString, claims)
			if err != nil {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "MapClaims"})
			}

			issuer, err := claims.GetIssuer()
			if err != nil {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Issuer"})
			}

			if issuer != os.Getenv("COGNITO_ISSUER") {
				return c.JSON(http.StatusForbidden, map[string]string{"error": "Invalid issuer"})
			}

			groups := claims["cognito:groups"]
			// Set user in context for later use
			if groups == nil {
				// Set user in context for later use
				cc.SetAuthUser(&models.AuthUser{
					ID:     claims["sub"].(string),
					Role:   models.UserRolePlayer,
					Source: auth.UserSourceCognito,
				})
			} else {
				cc.SetAuthUser(&models.AuthUser{
					ID:     claims["sub"].(string),
					Role:   groups.([]interface{})[0].(string),
					Source: auth.UserSourceCognito,
				})
			}
		}
		return next(cc)
	}
}

// CORS middleware to allow cross-origin requests
func CORS() echo.MiddlewareFunc {
	return echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept", "Authorization", "CognitoAuthorization"},
	})
}

// JWTConfig middleware to handle JWT token verification
func JWTConfig(jwtSecret interface{}) echo.MiddlewareFunc {
	return echojwt.WithConfig(echojwt.Config{
		SigningKey: jwtSecret, // Secret key to validate token
		ContextKey: "user",    // Key to store the user in the context
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return &auth.JwtAccessPayload{}
		},
		SuccessHandler: func(c echo.Context) {
			loginUser := c.Get("user")
			token, ok := loginUser.(*jwt.Token)
			if !ok {
				c.Logger().Warnf("context public_mobile type is invalid: %T", loginUser)
				return
			}

			payload, ok := token.Claims.(*auth.JwtAccessPayload)
			if !ok {
				c.Logger().Warnf("invalid claims type: %T", token.Claims)
				return
			}

			cc := c.(*auth.Context)
			// Set user in context for later use
			cc.SetAuthUser(&models.AuthUser{
				ID:     payload.Subject,
				Role:   payload.Role,
				Source: payload.Source,
			})
		},
	})
}
