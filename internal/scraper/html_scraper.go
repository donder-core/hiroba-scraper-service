package scraper

import (
	"fmt"
	"io"
	"net/http"
)

type HtmlScraper struct {
	client *http.Client
}

func NewHtmlScraper(client *http.Client) *HtmlScraper {
	return &HtmlScraper{client: client}
}

func (s *HtmlScraper) Scrape(songNo int, level int, taikoNo string, token string) (string, error) {
	targeturl := fmt.Sprintf("https://donderhiroba.jp/score_detail.php?song_no=%d&level=%d&taiko_no=%s", songNo, level, taikoNo)
	req, err := http.NewRequest("GET", targeturl, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("accept", "text/html,application/xhtml+xml")
	req.Header.Set("user-agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/145.0.0.0 Mobile Safari/537.36")
	req.Header.Set("cookie", "_token_v2="+token)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyText), nil
}
