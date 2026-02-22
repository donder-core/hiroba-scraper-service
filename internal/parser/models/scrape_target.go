package models

// ScrapeTarget identifies a specific song/difficulty pair that has a recorded score.
type ScrapeTarget struct {
	SongNo int
	Level  int
}
