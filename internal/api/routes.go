package api

import (
	"net/http"

	"github.com/donder-core/hiroba-scraper-service/internal/service"
	"github.com/labstack/echo/v5"
)

func SetupRoutes(e *echo.Echo, tokenHandler *TokenHandler, scoreService *service.ScoreService) error {
	if err := SetupTokenRoutes(e, tokenHandler); err != nil {
		return err
	}
	if err := SetupScraperRoutes(e, NewScraperHandler(scoreService)); err != nil {
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
	e.POST("/scrape_score_detail", func(c *echo.Context) error {
		return scraperHandler.ScrapeScoreDetail(c)
	})
	e.POST("/batch_score_detail", func(c *echo.Context) error {
		return scraperHandler.BatchGetScoreDetail(c)
	})
	e.GET("/job_status", func(c *echo.Context) error {
		return scraperHandler.GetJobStatus(c)
	})
	return nil
}
