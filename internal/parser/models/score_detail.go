package models

type ScoreDetail struct {
	CrownSrc         string `json:"crown_src"`
	BestScoreIconSrc string `json:"best_score_icon_src"`
	Ranking          string `json:"ranking"`
	HighScore        int    `json:"high_score"`
	Good             int    `json:"good"`
	Combo            int    `json:"combo"`
	OK               int    `json:"ok"`
	Drumroll         int    `json:"drumroll"`
	Bad              int    `json:"bad"`
}
