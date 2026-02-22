package api

import (
	"net/http"

	"github.com/donder-core/hiroba-scraper-service/internal/scraper"
	"github.com/labstack/echo/v5"
)

func SetupRoutes(e *echo.Echo, tokenHandler *TokenHandler, scraper *scraper.HtmlScraper) error {
	if err := SetupTokenRoutes(e, tokenHandler); err != nil {
		return err
	}
	if err := SetupScraperRoutes(e, NewScraperHandler(scraper, tokenHandler)); err != nil {
		return err
	}
	return nil
}

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

func SetupScraperRoutes(e *echo.Echo, scraperHandler *ScraperHandler) error {
	e.GET("/score_detail", func(c *echo.Context) error {
		scoreDetail, err := scraperHandler.GetScoreDetail(c)
		if err != nil {
			return c.String(http.StatusInternalServerError, err.Error())
		}
		return c.JSON(http.StatusOK, scoreDetail)
	})
	e.POST("/batch_score_detail", func(c *echo.Context) error {
		return scraperHandler.BatchGetScoreDetail(c)
	})
	e.GET("/job_status", func(c *echo.Context) error {
		return scraperHandler.GetJobStatus(c)
	})
	return nil
}
