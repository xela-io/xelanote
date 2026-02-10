package main

import (
	"context"
	"log"
	"time"

	"github.com/xela-io/xelanote/internal/jobs"
	"github.com/xela-io/xelanote/internal/service"
)

func startJobManager(noteService *service.NoteService) *jobs.JobManager {
	jobManager := jobs.NewJobManager(4) // 4 workers
	jobManager.RegisterHandler(jobs.JobTypeRenameNote, jobs.HandleRenameNoteJob(noteService))
	jobManager.Start()
	log.Println("Job manager started with 4 workers")
	return jobManager
}

// Start version pruning job (runs daily, keeps 100 versions per note).
func startVersionPruner(noteService *service.NoteService) context.CancelFunc {
	pruneCtx, pruneCancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		// Run once at startup
		pruned, err := noteService.PruneAllVersions(100)
		if err != nil {
			log.Printf("Version pruning failed: %v", err)
		} else if pruned > 0 {
			log.Printf("Pruned %d old versions at startup", pruned)
		}

		for {
			select {
			case <-ticker.C:
				pruned, err := noteService.PruneAllVersions(100)
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
