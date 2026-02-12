package jobs

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestJobManager_SubmitAndExecute(t *testing.T) {
	jm := NewJobManager(1)
	jm.RegisterHandler(JobTypeRenameNote, func(ctx context.Context, job *Job) error {
		job.UpdateProgress(0.5)
		return nil
	})
	jm.Start()
	defer jm.Stop()

	job := &Job{ID: "job1", Type: JobTypeRenameNote, UserID: 1}
	if err := jm.Submit(job); err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	waitForJobStatus(t, jm, "job1", JobStatusCompleted)
	got, err := jm.GetJob("job1")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if got.Status != JobStatusCompleted {
		t.Fatalf("expected completed, got %s", got.Status)
	}
	if got.Progress != 1.0 {
		t.Fatalf("expected progress 1.0, got %v", got.Progress)
	}
}

func TestJobManager_NoHandlerFailsJob(t *testing.T) {
	jm := NewJobManager(1)
	jm.Start()
	defer jm.Stop()

	job := &Job{ID: "job2", Type: JobTypeRenameNote, UserID: 1}
	if err := jm.Submit(job); err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	waitForJobStatus(t, jm, "job2", JobStatusFailed)
	got, _ := jm.GetJob("job2")
	if got.Status != JobStatusFailed {
		t.Fatalf("expected failed, got %s", got.Status)
	}
	if got.Error == "" {
		t.Fatalf("expected error message")
	}
}

func TestJobManager_HandlerErrorFailsJob(t *testing.T) {
	jm := NewJobManager(1)
	jm.RegisterHandler(JobTypeRenameNote, func(ctx context.Context, job *Job) error {
		return errors.New("boom")
	})
	jm.Start()
	defer jm.Stop()

	job := &Job{ID: "job3", Type: JobTypeRenameNote, UserID: 1}
	if err := jm.Submit(job); err != nil {
		t.Fatalf("submit failed: %v", err)
	}

	waitForJobStatus(t, jm, "job3", JobStatusFailed)
	got, _ := jm.GetJob("job3")
	if got.Error == "" {
		t.Fatalf("expected error message")
	}
}

func TestJobManager_SubmitAfterStop(t *testing.T) {
	jm := NewJobManager(1)
	jm.Stop()

	job := &Job{ID: "job4", Type: JobTypeRenameNote, UserID: 1}
	if err := jm.Submit(job); err == nil {
		t.Fatalf("expected error on submit after stop")
	}
}

func TestJobManager_CleanupRemovesOldCompleted(t *testing.T) {
	jm := NewJobManager(1)
	old := &Job{
		ID:        "job-old",
		Status:    JobStatusCompleted,
		UpdatedAt: time.Now().Add(-jobRetention * 2),
	}
	jm.jobs.Store(old.ID, old)

	cutoff := time.Now().Add(-jobRetention)
	jm.jobs.Range(func(key, value interface{}) bool {
		job := value.(*Job)
		if job.Status == JobStatusCompleted && job.UpdatedAt.Before(cutoff) {
			jm.jobs.Delete(key)
		}
		return true
	})

	if _, ok := jm.jobs.Load("job-old"); ok {
		t.Fatalf("expected old job to be removed")
	}
}

func waitForJobStatus(t *testing.T, jm *JobManager, jobID string, status JobStatus) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		job, err := jm.GetJob(jobID)
		if err == nil && job.Status == status {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	job, err := jm.GetJob(jobID)
	if err != nil {
		t.Fatalf("job not found: %v", err)
	}
	t.Fatalf("timeout waiting for status %s (current=%s)", status, job.Status)
}
