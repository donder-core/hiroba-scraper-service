package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/donder-core/hiroba-scraper-service/internal/parser"
	"github.com/donder-core/hiroba-scraper-service/internal/parser/models"
)

const batchWorkerCount = 2000

type TokenProvider interface {
	GetToken() (string, error)
}

type Scraper interface {
	Scrape(songNo, level int, taikoNo string, token string) (string, error)
	ScrapeScoreSummary(genre int, taikoNo string, token string) (string, error)
}

type ScoreRepository interface {
	UpsertScoreDetail(ctx context.Context, songNo, level int, taikoNo string, detail *models.ScoreDetail) (*models.ScoreDetail, error)
	BatchUpsertScoreDetails(ctx context.Context, taikoNo string, records []models.ScoredRecord) error
	GetScoresByTaikoNo(ctx context.Context, taikoNo string) ([]models.ScoredRecord, error)
}

type ScoreService struct {
	scraper       Scraper
	tokenProvider TokenProvider
	repo          ScoreRepository
	jobStore      *JobStore
}

func NewScoreService(scraper Scraper, tokenProvider TokenProvider, repo ScoreRepository) *ScoreService {
	return &ScoreService{
		scraper:       scraper,
		tokenProvider: tokenProvider,
		repo:          repo,
		jobStore:      newJobStore(),
	}
}

func (s *ScoreService) ScrapeScoreDetail(songNo, level int, taikoNo string) (*models.ScoreDetail, error) {
	token, err := s.tokenProvider.GetToken()
	if err != nil {
		return nil, err
	}

	html, err := s.scraper.Scrape(songNo, level, taikoNo, token)
	if err != nil {
		return nil, err
	}

	detail, err := parser.ParseScoreDetail(html)
	if err != nil {
		return nil, err
	}

	if s.repo != nil {
		persisted, err := s.repo.UpsertScoreDetail(context.Background(), songNo, level, taikoNo, detail)
		if err != nil {
			slog.Warn("failed to persist score detail", "song_no", songNo, "level", level, "taiko_no", taikoNo, "error", err)
		} else {
			return persisted, nil
		}
	}

	return detail, nil
}

func (s *ScoreService) BatchGetScoreDetail(level int, taikoNo string) (string, time.Time) {
	songIds := naiveGetSongIds()
	targets := make([]models.ScrapeTarget, len(songIds))
	for i, id := range songIds {
		targets[i] = models.ScrapeTarget{SongNo: id, Level: level}
	}
	job := s.jobStore.CreateJob(len(targets))

	go func() {
		token, err := s.tokenProvider.GetToken()
		if err != nil {
			slog.Error("batch job failed to get token", "job_id", job.ID, "error", err)
			job.fail()
			return
		}

		records := s.scrapeTargets(job, taikoNo, token, targets)

		if s.repo != nil && len(records) > 0 {
			if err := s.repo.BatchUpsertScoreDetails(context.Background(), taikoNo, records); err != nil {
				slog.Warn("batch upsert failed", "job_id", job.ID, "count", len(records), "error", err)
			}
		}

		job.complete()
		slog.Info("batch job completed", "job_id", job.ID, "total_results", len(records))
	}()

	return job.ID, job.StartedAt
}

const genreCount = 8

// SmartBatchScrape first discovers which (song_no, level) pairs have recorded
// scores by scraping all genre summary pages, then fetches only those score
// details. This avoids the naive approach of blindly trying every song ID.
func (s *ScoreService) SmartBatchScrape(taikoNo string) (string, time.Time) {
	// Create a placeholder job; total is updated once discovery completes.
	job := s.jobStore.CreateJob(0)

	go func() {
		token, err := s.tokenProvider.GetToken()
		if err != nil {
			slog.Error("smart batch job failed to get token", "job_id", job.ID, "error", err)
			job.fail()
			return
		}

		// Phase 1: fan out genre summary scrapes concurrently since each is an
		// independent HTTP request. Genres rarely share songs, but the seen map
		// guards against duplicates if they do.
		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			seen = make(map[models.ScrapeTarget]struct{})
		)
		var targets []models.ScrapeTarget

		for genre := 1; genre <= genreCount; genre++ {
			wg.Add(1)
			go func(g int) {
				defer wg.Done()
				html, err := s.scraper.ScrapeScoreSummary(g, taikoNo, token)
				if err != nil {
					slog.Warn("smart batch: failed to scrape summary", "job_id", job.ID, "genre", g, "error", err)
					return
				}
				found, err := parser.ParseScoreSummary(html)
				if err != nil {
					slog.Warn("smart batch: failed to parse summary", "job_id", job.ID, "genre", g, "error", err)
					return
				}
				mu.Lock()
				for _, t := range found {
					if _, exists := seen[t]; !exists {
						seen[t] = struct{}{}
						targets = append(targets, t)
					}
				}
				mu.Unlock()
			}(genre)
		}
		wg.Wait()

		slog.Info("smart batch: discovery complete", "job_id", job.ID, "targets", len(targets))
		job.setTotal(len(targets))

		// Phase 2: scrape score details for discovered targets only.
		records := s.scrapeTargets(job, taikoNo, token, targets)

		if s.repo != nil && len(records) > 0 {
			if err := s.repo.BatchUpsertScoreDetails(context.Background(), taikoNo, records); err != nil {
				slog.Warn("smart batch upsert failed", "job_id", job.ID, "count", len(records), "error", err)
			}
		}

		job.complete()
		slog.Info("smart batch job completed", "job_id", job.ID, "total_results", len(records))
	}()

	return job.ID, job.StartedAt
}

// scrapeTargets runs a fixed-size worker pool that fetches and parses a score
// detail page for each target, returning all successfully scraped records.
func (s *ScoreService) scrapeTargets(job *Job, taikoNo, token string, targets []models.ScrapeTarget) []models.ScoredRecord {
	targetCh := make(chan models.ScrapeTarget, len(targets))
	for _, t := range targets {
		targetCh <- t
	}
	close(targetCh)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		records = make([]models.ScoredRecord, 0, len(targets))
	)

	for range batchWorkerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range targetCh {
				html, err := s.scraper.Scrape(target.SongNo, target.Level, taikoNo, token)
				if err != nil {
					job.incrementProgress(nil, fmt.Sprintf("song_no=%d level=%d: scrape error: %s", target.SongNo, target.Level, err.Error()))
					continue
				}

				scoreDetail, err := parser.ParseScoreDetail(html)
				if err != nil {
					job.incrementProgress(nil, fmt.Sprintf("song_no=%d level=%d: parse error: %s", target.SongNo, target.Level, err.Error()))
					continue
				}

				job.incrementProgress(scoreDetail, "")

				mu.Lock()
				records = append(records, models.ScoredRecord{SongNo: target.SongNo, Level: target.Level, Detail: *scoreDetail})
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	return records
}

func (s *ScoreService) GetScoresByTaikoNo(ctx context.Context, taikoNo string) ([]models.ScoredRecord, error) {
	if s.repo == nil {
		return nil, fmt.Errorf("no repository configured")
	}
	return s.repo.GetScoresByTaikoNo(ctx, taikoNo)
}

func (s *ScoreService) GetJobStatus(jobID string) (*JobSnapshot, bool) {
	job, ok := s.jobStore.GetJob(jobID)
	if !ok {
		return nil, false
	}
	snap := job.snapshot()
	return &snap, true
}

func naiveGetSongIds() []int {
	songIds := make([]int, 1500)
	for i := range songIds {
		songIds[i] = i
	}
	return songIds
}
