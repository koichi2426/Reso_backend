package controller

import (
	"net/http"
	"strings"

	"app/src/usecase"
	"github.com/labstack/echo/v4"
)

type GetUserSpotsController struct {
	usecase usecase.GetUserSpotsUseCase
}

func NewGetUserSpotsController(u usecase.GetUserSpotsUseCase) *GetUserSpotsController {
	return &GetUserSpotsController{usecase: u}
}

func (ctrl *GetUserSpotsController) Execute(c echo.Context) error {
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Missing or invalid authorization header"})
	}
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")

	input := usecase.GetUserSpotsInput{Token: tokenString}
	output, err := ctrl.usecase.Execute(c.Request().Context(), input)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, output)
}
