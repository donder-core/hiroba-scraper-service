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

const batchWorkerCount = 20

type TokenProvider interface {
	GetToken() (string, error)
}

type Scraper interface {
	Scrape(songNo, level int, taikoNo string, token string) (string, error)
}

type ScoreRepository interface {
	UpsertScoreDetail(ctx context.Context, songNo, level int, taikoNo string, detail *models.ScoreDetail) (*models.ScoreDetail, error)
	BatchUpsertScoreDetails(ctx context.Context, level int, taikoNo string, records []models.ScoredRecord) error
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
	job := s.jobStore.CreateJob(len(songIds))

	go func() {
		token, err := s.tokenProvider.GetToken()
		if err != nil {
			slog.Error("batch job failed to get token", "job_id", job.ID, "error", err)
			job.fail()
			return
		}

		songCh := make(chan int, len(songIds))
		for _, id := range songIds {
			songCh <- id
		}
		close(songCh)

		var (
			wg sync.WaitGroup
			mu sync.Mutex
		)
		records := make([]models.ScoredRecord, 0, len(songIds))

		for range batchWorkerCount {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for songNo := range songCh {
					html, err := s.scraper.Scrape(songNo, level, taikoNo, token)
					if err != nil {
						job.incrementProgress(nil, fmt.Sprintf("song_no=%d: scrape error: %s", songNo, err.Error()))
						continue
					}

					scoreDetail, err := parser.ParseScoreDetail(html)
					if err != nil {
						job.incrementProgress(nil, fmt.Sprintf("song_no=%d: parse error: %s", songNo, err.Error()))
						continue
					}

					job.incrementProgress(scoreDetail, "")

				mu.Lock()
				records = append(records, models.ScoredRecord{SongNo: songNo, Detail: *scoreDetail})
				mu.Unlock()
				}
			}()
		}
		wg.Wait()

		if s.repo != nil && len(records) > 0 {
			if err := s.repo.BatchUpsertScoreDetails(context.Background(), level, taikoNo, records); err != nil {
				slog.Warn("batch upsert failed", "job_id", job.ID, "count", len(records), "error", err)
			}
		}

		job.complete()

		job.mu.RLock()
		results := job.Results
		job.mu.RUnlock()

		slog.Info("batch job completed", "job_id", job.ID, "total_results", len(results))
	}()

	return job.ID, job.StartedAt
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
