package parser

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/donder-core/hiroba-scraper-service/internal/parser/models"
)

func ParseScoreDetail(html string) (*models.ScoreDetail, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	// Scope to the first .scoreDetailTable to avoid picking up
	// per-section breakdown stats that reuse the same class names.
	table := doc.Find(".scoreDetailTable").First()

	detail := &models.ScoreDetail{
		CrownSrc:         doc.Find("img.crown").AttrOr("src", ""),
		BestScoreIconSrc: doc.Find("img.best_score_icon").AttrOr("src", ""),
		Ranking:          parseStatStr(table, ".ranking span", "位"),
		HighScore:        parseStatInt(table, ".high_score span", "点"),
		Good:             parseStatInt(table, ".good_cnt span", "回"),
		Combo:            parseStatInt(table, ".combo_cnt span", "回"),
		OK:               parseStatInt(table, ".ok_cnt span", "回"),
		Drumroll:         parseStatInt(table, ".pound_cnt span", "回"),
		Bad:              parseStatInt(table, ".ng_cnt span", "回"),
	}

	return detail, nil
}

// ParseScoreSummary parses a genre score summary page and returns the set of
// (song_no, level) pairs for which the user has a recorded score. Entries with
// crown_button_none (unplayed) are excluded.
func ParseScoreSummary(html string) ([]models.ScrapeTarget, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	var targets []models.ScrapeTarget

	doc.Find("#songList li.contentBox").Each(func(_ int, song *goquery.Selection) {
		song.Find(".buttonList li a").Each(func(_ int, btn *goquery.Selection) {
			imgSrc, exists := btn.Find("img").Attr("src")
			if !exists || strings.Contains(imgSrc, "crown_button_none") {
				return
			}

			href, exists := btn.Attr("href")
			if !exists {
				return
			}

			parsed, err := url.Parse(href)
			if err != nil {
				return
			}
			q := parsed.Query()

			songNo, err := strconv.Atoi(q.Get("song_no"))
			if err != nil {
				return
			}
			level, err := strconv.Atoi(q.Get("level"))
			if err != nil {
				return
			}

			targets = append(targets, models.ScrapeTarget{SongNo: songNo, Level: level})
		})
	})

	return targets, nil
}

func parseStatStr(sel *goquery.Selection, selector, suffix string) string {
	return strings.TrimSuffix(strings.TrimSpace(sel.Find(selector).Text()), suffix)
}

func parseStatInt(sel *goquery.Selection, selector, suffix string) int {
	raw := parseStatStr(sel, selector, suffix)
	n, _ := strconv.Atoi(raw)
	return n
}
