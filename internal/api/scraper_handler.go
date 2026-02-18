package api

import (
	"fmt"
	"strconv"

	"github.com/donder-core/hiroba-scraper-service/internal/parser"
	"github.com/donder-core/hiroba-scraper-service/internal/parser/models"
	"github.com/donder-core/hiroba-scraper-service/internal/scraper"
	"github.com/labstack/echo/v5"
)

type ScraperHandler struct {
	scraper      *scraper.HtmlScraper
	tokenHandler *TokenHandler
}

func NewScraperHandler(scraper *scraper.HtmlScraper, tokenHandler *TokenHandler) *ScraperHandler {
	return &ScraperHandler{scraper: scraper, tokenHandler: tokenHandler}
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
