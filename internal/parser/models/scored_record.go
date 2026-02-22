package models

// ScoredRecord pairs the composite key fields with a ScoreDetail,
// since song_no and level are not part of the ScoreDetail model itself.
type ScoredRecord struct {
	SongNo int
	Level  int
	Detail ScoreDetail
}
