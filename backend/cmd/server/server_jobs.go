package main

import (
	"context"
	"log"
	"time"

	"github.com/xela-io/xelanote/internal/jobs"
	"github.com/xela-io/xelanote/internal/service"
)

func startJobManager(noteService *service.NoteService) *jobs.JobManager {
	const jobWorkerCount = 4
	jobManager := jobs.NewJobManager(jobWorkerCount)
	jobManager.RegisterHandler(jobs.JobTypeRenameNote, jobs.HandleRenameNoteJob(noteService))
	jobManager.Start()
	log.Printf("Job manager started with %d workers", jobWorkerCount)
	return jobManager
}

// Start version pruning job (runs daily, keeps 100 versions per note).
func startVersionPruner(noteService *service.NoteService) context.CancelFunc {
	const maxVersionsToKeep = 100

	pruneCtx, pruneCancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		// Run once at startup
		pruned, err := noteService.PruneAllVersions(maxVersionsToKeep)
		if err != nil {
			log.Printf("Version pruning failed: %v", err)
		} else if pruned > 0 {
			log.Printf("Pruned %d old versions at startup", pruned)
		}

		for {
			select {
			case <-ticker.C:
				pruned, err := noteService.PruneAllVersions(maxVersionsToKeep)
				if err != nil {
					log.Printf("Version pruning failed: %v", err)
				} else if pruned > 0 {
					log.Printf("Pruned %d old versions", pruned)
				}
			case <-pruneCtx.Done():
				return
			}
		}
	}()

	return pruneCancel
}
