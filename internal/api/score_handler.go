package api

import (
	"context"
	"net/http"

	"github.com/donder-core/hiroba-scraper-service/internal/service"
	"github.com/labstack/echo/v5"
)

type ScoreHandler struct {
	scoreService *service.ScoreService
}

func NewScoreHandler(scoreService *service.ScoreService) *ScoreHandler {
	return &ScoreHandler{scoreService: scoreService}
}

func (sh *ScoreHandler) GetScores(c *echo.Context) error {
	taikoNo := c.QueryParam("taiko_no")
	if taikoNo == "" {
		return c.String(http.StatusBadRequest, "missing taiko_no")
	}

	records, err := sh.scoreService.GetScoresByTaikoNo(context.Background(), taikoNo)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, records)
}
