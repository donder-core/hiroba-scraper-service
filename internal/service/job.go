package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/donder-core/hiroba-scraper-service/internal/parser/models"
)

const (
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

type Job struct {
	ID         string
	Status     string
	Progress   int
	Total      int
	Results    []models.ScoreDetail
	Errors     []string
	StartedAt  time.Time
	FinishedAt time.Time
	mu         sync.RWMutex
}

func (j *Job) incrementProgress(result *models.ScoreDetail, errMsg string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Progress++
	if result != nil {
		j.Results = append(j.Results, *result)
	}
	if errMsg != "" {
		j.Errors = append(j.Errors, errMsg)
	}
}

func (j *Job) complete() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = StatusCompleted
	j.FinishedAt = time.Now()
}

func (j *Job) fail() {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = StatusFailed
	j.FinishedAt = time.Now()
}

type JobSnapshot struct {
	ID         string     `json:"job_id"`
	Status     string     `json:"status"`
	Progress   int        `json:"progress"`
	Total      int        `json:"total"`
	ErrorCount int        `json:"error_count"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`
}

func (j *Job) snapshot() JobSnapshot {
	j.mu.RLock()
	defer j.mu.RUnlock()
	snap := JobSnapshot{
		ID:         j.ID,
		Status:     j.Status,
		Progress:   j.Progress,
		Total:      j.Total,
		ErrorCount: len(j.Errors),
	}
	if !j.FinishedAt.IsZero() {
		t := j.FinishedAt
		snap.FinishedAt = &t
	}
	return snap
}

type JobStore struct {
	mu   sync.RWMutex
	jobs map[string]*Job
}

func newJobStore() *JobStore {
	return &JobStore{jobs: make(map[string]*Job)}
}

func (js *JobStore) CreateJob(total int) *Job {
	id := fmt.Sprintf("%d", time.Now().UnixNano())
	job := &Job{
		ID:        id,
		Status:    StatusRunning,
		Total:     total,
		Results:   make([]models.ScoreDetail, 0, total),
		Errors:    make([]string, 0),
		StartedAt: time.Now(),
	}
	js.mu.Lock()
	js.jobs[id] = job
	js.mu.Unlock()
	return job
}

func (js *JobStore) GetJob(id string) (*Job, bool) {
	js.mu.RLock()
	defer js.mu.RUnlock()
	job, ok := js.jobs[id]
	return job, ok
}
