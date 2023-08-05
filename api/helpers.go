package api

import (
	"encoding/json"
	"github.com/labstack/echo/v4"
)

type ErrorMessage struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error   string          `json:"error"`
	Details []*ErrorMessage `json:"details"`
}

func GetPayload[T any](c echo.Context) (*T, error) {
	var payload T
	err := json.NewDecoder(c.Request().Body).Decode(&payload)
	if err != nil {
		return nil, err
	}
	return &payload, nil
}

func GetErrorResponse(err error) *ErrorMessage {
	return &ErrorMessage{
		Message: err.Error(),
	}
}
