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

	detail := &models.ScoreDetail{
		CrownSrc:         doc.Find("img.crown").AttrOr("src", ""),
		BestScoreIconSrc: doc.Find("img.best_score_icon").AttrOr("src", ""),
		Ranking:          parseStatStr(doc, ".ranking span", "位"),
		HighScore:        parseStatInt(doc, ".high_score span", "点"),
		Good:             parseStatInt(doc, ".good_cnt span", "回"),
		Combo:            parseStatInt(doc, ".combo_cnt span", "回"),
		OK:               parseStatInt(doc, ".ok_cnt span", "回"),
		Drumroll:         parseStatInt(doc, ".pound_cnt span", "回"),
		Bad:              parseStatInt(doc, ".ng_cnt span", "回"),
	}

	return detail, nil
}

func parseStatStr(doc *goquery.Document, selector, suffix string) string {
	return strings.TrimSuffix(strings.TrimSpace(doc.Find(selector).Text()), suffix)
}

func parseStatInt(doc *goquery.Document, selector, suffix string) int {
	raw := parseStatStr(doc, selector, suffix)
	n, _ := strconv.Atoi(raw)
	return n
}
