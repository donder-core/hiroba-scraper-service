package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"

	"github.com/donder-core/hiroba-scraper-service/internal/parser"
	"github.com/donder-core/hiroba-scraper-service/internal/parser/models"
	"github.com/donder-core/hiroba-scraper-service/internal/scraper"
	"github.com/labstack/echo/v5"
)

const batchWorkerCount = 20

type ScraperHandler struct {
	scraper      *scraper.HtmlScraper
	tokenHandler *TokenHandler
	jobStore     *JobStore
}

func NewScraperHandler(scraper *scraper.HtmlScraper, tokenHandler *TokenHandler) *ScraperHandler {
	return &ScraperHandler{
		scraper:      scraper,
		tokenHandler: tokenHandler,
		jobStore:     newJobStore(),
	}
}

func naiveGetSongIds() []int {
	songIds := make([]int, 1500)
	for i := 0; i < 1500; i++ {
		songIds[i] = i
	}
	return songIds
}

func (sh *ScraperHandler) BatchGetScoreDetail(c *echo.Context) error {
	level, err := strconv.Atoi(c.QueryParam("level"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid level")
	}

	taikoNo, err := strconv.Atoi(c.QueryParam("taiko_no"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid taiko_no")
	}

	songIds := naiveGetSongIds()
	job := sh.jobStore.CreateJob(len(songIds))

	go func() {
		token, err := sh.tokenHandler.GetToken()
		if err != nil {
			slog.Error("batch job failed to get token", "job_id", job.ID, "error", err)
			job.fail()
			return
		}

		songCh := make(chan int, len(songIds))
		for _, id := range songIds {
			songCh <- id
		}
		close(songCh)

		var wg sync.WaitGroup
		for range batchWorkerCount {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for songNo := range songCh {
					html, err := sh.scraper.Scrape(songNo, level, taikoNo, token)
					if err != nil {
						job.incrementProgress(nil, fmt.Sprintf("song_no=%d: scrape error: %s", songNo, err.Error()))
						continue
					}

					scoreDetail, err := parser.ParseScoreDetail(html)
					if err != nil {
						job.incrementProgress(nil, fmt.Sprintf("song_no=%d: parse error: %s", songNo, err.Error()))
						continue
					}

					job.incrementProgress(scoreDetail, "")
				}
			}()
		}
		wg.Wait()

		job.complete()

		job.mu.RLock()
		results := job.Results
		job.mu.RUnlock()

		slog.Info("batch job completed", "job_id", job.ID, "total_results", len(results), "results", results)
	}()

	return c.JSON(http.StatusAccepted, map[string]any{
		"job_id":     job.ID,
		"started_at": job.StartedAt,
	})
}

func (sh *ScraperHandler) GetJobStatus(c *echo.Context) error {
	jobID := c.QueryParam("job_id")
	if jobID == "" {
		return c.String(http.StatusBadRequest, "missing job_id")
	}

	job, ok := sh.jobStore.GetJob(jobID)
	if !ok {
		return c.String(http.StatusNotFound, "job not found")
	}

	return c.JSON(http.StatusOK, job.snapshot())
}

func (sh *ScraperHandler) GetScoreDetail(c *echo.Context) (*models.ScoreDetail, error) {
	songNo, err := strconv.Atoi(c.QueryParam("song_no"))
	if err != nil {
		return nil, fmt.Errorf("invalid song_no")
	}

	level, err := strconv.Atoi(c.QueryParam("level"))
	if err != nil {
		return nil, fmt.Errorf("invalid level")
	}

	taikoNo, err := strconv.Atoi(c.QueryParam("taiko_no"))
	if err != nil {
		return nil, fmt.Errorf("invalid taiko_no")
	}

	token, err := sh.tokenHandler.GetToken()
	if err != nil {
		return nil, err
	}

	html, err := sh.scraper.Scrape(songNo, level, taikoNo, token)
	if err != nil {
		return nil, err
	}

	scoreDetail, err := parser.ParseScoreDetail(html)
	if err != nil {
		return nil, err
	}

	return scoreDetail, nil
}
