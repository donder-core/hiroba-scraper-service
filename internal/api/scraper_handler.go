package api

import (
	"net/http"
	"strconv"

	"github.com/donder-core/hiroba-scraper-service/internal/service"
	"github.com/labstack/echo/v5"
)

type ScraperHandler struct {
	scoreService *service.ScoreService
}

func NewScraperHandler(scoreService *service.ScoreService) *ScraperHandler {
	return &ScraperHandler{scoreService: scoreService}
}

func (sh *ScraperHandler) ScrapeScoreDetail(c *echo.Context) error {
	songNo, err := strconv.Atoi(c.QueryParam("song_no"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid song_no")
	}

	level, err := strconv.Atoi(c.QueryParam("level"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid level")
	}

	taikoNo := c.QueryParam("taiko_no")
	if taikoNo == "" {
		return c.String(http.StatusBadRequest, "missing taiko_no")
	}

	scoreDetail, err := sh.scoreService.ScrapeScoreDetail(songNo, level, taikoNo)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, scoreDetail)
}

func (sh *ScraperHandler) BatchGetScoreDetail(c *echo.Context) error {
	level, err := strconv.Atoi(c.QueryParam("level"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid level")
	}

	taikoNo := c.QueryParam("taiko_no")
	if taikoNo == "" {
		return c.String(http.StatusBadRequest, "missing taiko_no")
	}

	jobID, startedAt := sh.scoreService.BatchGetScoreDetail(level, taikoNo)

	return c.JSON(http.StatusAccepted, map[string]any{
		"job_id":     jobID,
		"started_at": startedAt,
	})
}

func (sh *ScraperHandler) GetJobStatus(c *echo.Context) error {
	jobID := c.QueryParam("job_id")
	if jobID == "" {
		return c.String(http.StatusBadRequest, "missing job_id")
	}

	snapshot, ok := sh.scoreService.GetJobStatus(jobID)
	if !ok {
		return c.String(http.StatusNotFound, "job not found")
	}

	return c.JSON(http.StatusOK, snapshot)
}
