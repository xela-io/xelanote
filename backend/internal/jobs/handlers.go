package jobs

import (
	"context"
	"fmt"

	"github.com/xela-io/xelanote/internal/service"
)

// HandleRenameNoteJob handles the rename note job
func HandleRenameNoteJob(noteService *service.NoteService) JobHandler {
	return func(ctx context.Context, job *Job) error {
		noteID, ok := job.Metadata["noteID"].(string)
		if !ok {
			return fmt.Errorf("invalid noteID in job metadata")
		}

		newTitle, ok := job.Metadata["newTitle"].(string)
		if !ok {
			return fmt.Errorf("invalid newTitle in job metadata")
		}

		job.UpdateProgress(0.1)

		// Execute the rename operation
		result, err := noteService.RenameNote(ctx, job.UserID, noteID, newTitle)
		if err != nil {
			return fmt.Errorf("failed to rename note: %w", err)
		}

		job.mu.Lock()
		job.Progress = 1.0
		job.Result = result
		job.mu.Unlock()
		return nil
	}
}
