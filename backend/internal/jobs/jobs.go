package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// JobType represents the type of job
type JobType string

const (
	JobTypeRenameNote JobType = "rename_note"
)

// JobStatus represents the current status of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)

// Job represents a background job
type Job struct {
	ID        string
	Type      JobType
	UserID    int
	Status    JobStatus
	Progress  float64 // 0.0 to 1.0
	Result    interface{}
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Metadata  map[string]interface{}
}

// JobHandler is a function that executes a job
type JobHandler func(ctx context.Context, job *Job) error

// JobManager manages background job execution
type JobManager struct {
	jobs     sync.Map // map[string]*Job
	queue    chan *Job
	workers  int
	ctx      context.Context
	cancel   context.CancelFunc
	handlers map[JobType]JobHandler
}

const (
	jobCleanupInterval = 5 * time.Minute
	jobRetention       = 24 * time.Hour
)

// NewJobManager creates a new JobManager
func NewJobManager(workers int) *JobManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &JobManager{
		queue:    make(chan *Job, 1000),
		workers:  workers,
		ctx:      ctx,
		cancel:   cancel,
		handlers: make(map[JobType]JobHandler),
	}
}

// RegisterHandler registers a handler for a job type
func (jm *JobManager) RegisterHandler(jobType JobType, handler JobHandler) {
	jm.handlers[jobType] = handler
}

// Start starts the worker pool
func (jm *JobManager) Start() {
	for i := 0; i < jm.workers; i++ {
		go jm.worker()
	}
	go jm.cleanupLoop()
}

// Stop stops the job manager
func (jm *JobManager) Stop() {
	jm.cancel()
}

// Submit submits a job for execution
func (jm *JobManager) Submit(job *Job) error {
	job.Status = JobStatusPending
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
	jm.jobs.Store(job.ID, job)

	select {
	case jm.queue <- job:
		return nil
	case <-jm.ctx.Done():
		return fmt.Errorf("job manager stopped")
	}
}

// GetJob retrieves a job by ID
func (jm *JobManager) GetJob(jobID string) (*Job, error) {
	if job, ok := jm.jobs.Load(jobID); ok {
		return job.(*Job), nil
	}
	return nil, fmt.Errorf("job not found")
}

// worker processes jobs from the queue
func (jm *JobManager) worker() {
	for {
		select {
		case <-jm.ctx.Done():
			return
		case job := <-jm.queue:
			jm.executeJob(job)
		}
	}
}

// cleanupLoop periodically removes completed/failed jobs to prevent unbounded growth.
func (jm *JobManager) cleanupLoop() {
	ticker := time.NewTicker(jobCleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-jm.ctx.Done():
			return
		case <-ticker.C:
			cutoff := time.Now().Add(-jobRetention)
			jm.jobs.Range(func(key, value interface{}) bool {
				job, ok := value.(*Job)
				if !ok {
					jm.jobs.Delete(key)
					return true
				}
				if (job.Status == JobStatusCompleted || job.Status == JobStatusFailed) && job.UpdatedAt.Before(cutoff) {
					jm.jobs.Delete(key)
				}
				return true
			})
		}
	}
}

// executeJob executes a single job
func (jm *JobManager) executeJob(job *Job) {
	job.Status = JobStatusRunning
	job.UpdatedAt = time.Now()
	job.Progress = 0.0
	jm.jobs.Store(job.ID, job)

	handler, ok := jm.handlers[job.Type]
	if !ok {
		job.Status = JobStatusFailed
		job.Error = fmt.Sprintf("no handler registered for job type: %s", job.Type)
		job.UpdatedAt = time.Now()
		jm.jobs.Store(job.ID, job)
		return
	}

	err := handler(jm.ctx, job)
	if err != nil {
		job.Status = JobStatusFailed
		job.Error = err.Error()
	} else {
		job.Status = JobStatusCompleted
		job.Progress = 1.0
	}
	job.UpdatedAt = time.Now()
	jm.jobs.Store(job.ID, job)
}
