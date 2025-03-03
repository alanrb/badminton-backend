package auth

import (
	"time"

	"github.com/alanrb/badminton/backend/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Context struct {
	echo.Context
	user *models.AuthUser
}

func (c *Context) AuthUser() *models.AuthUser {
	return c.user
}

func (c *Context) SetAuthUser(u *models.AuthUser) {
	c.user = u
}

type JwtAccessPayload struct {
	jwt.RegisteredClaims
	Role   string `json:"role,omitempty"`
	Source string `json:"source,omitempty"`
}

const (
	UserSourceCognito = "cognito"
	UserSourceInit    = "badminton"
)

func GenerateJWTToken(user models.User, jwtSecret interface{}) (string, error) {
	claims := JwtAccessPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    UserSourceInit,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 24)),
			Subject:   user.ID,
		},
		Role:   user.Role,
		Source: UserSourceInit,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}
