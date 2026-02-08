package api

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

func SetupTokenRoutes(e *echo.Echo, tokenHandler *TokenHandler) error {

	e.GET("/token", func(c *echo.Context) error {
		token, err := tokenHandler.GetToken()
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		return c.String(http.StatusOK, token)
	})

	return nil
}
