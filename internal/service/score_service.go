package service

import (
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
	Scrape(songNo, level, taikoNo int, token string) (string, error)
}

type ScoreService struct {
	scraper       Scraper
	tokenProvider TokenProvider
	jobStore      *JobStore
}

func NewScoreService(scraper Scraper, tokenProvider TokenProvider) *ScoreService {
	return &ScoreService{
		scraper:       scraper,
		tokenProvider: tokenProvider,
		jobStore:      newJobStore(),
	}
}

func (s *ScoreService) GetScoreDetail(songNo, level, taikoNo int) (*models.ScoreDetail, error) {
	token, err := s.tokenProvider.GetToken()
	if err != nil {
		return nil, err
	}

	html, err := s.scraper.Scrape(songNo, level, taikoNo, token)
	if err != nil {
		return nil, err
	}

	return parser.ParseScoreDetail(html)
}

func (s *ScoreService) BatchGetScoreDetail(level, taikoNo int) (string, time.Time) {
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

		var wg sync.WaitGroup
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
				}
			}()
		}
		wg.Wait()

		job.complete()

		job.mu.RLock()
		results := job.Results
		job.mu.RUnlock()

		slog.Info("batch job completed", "job_id", job.ID, "total_results", len(results), "results", results)
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
