package parser

import (
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

func parseStatStr(sel *goquery.Selection, selector, suffix string) string {
	return strings.TrimSuffix(strings.TrimSpace(sel.Find(selector).Text()), suffix)
}

func parseStatInt(sel *goquery.Selection, selector, suffix string) int {
	raw := parseStatStr(sel, selector, suffix)
	n, _ := strconv.Atoi(raw)
	return n
}
