package handlers

import (
	"errors"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func GetParamID(c echo.Context, param string) (string, error) {
	id := c.Param(param)
	if err := uuid.Validate(id); err != nil {
		return "", errors.New("invalid ID")
	}
	return id, nil
}
