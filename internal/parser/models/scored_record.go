package models

// ScoredRecord pairs the composite key fields with a ScoreDetail,
// since song_no is not part of the ScoreDetail model itself.
type ScoredRecord struct {
	SongNo int
	Detail ScoreDetail
}
