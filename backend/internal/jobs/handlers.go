package jobs

import (
	"context"
	"fmt"
	"time"

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

		job.Progress = 0.1
		job.UpdatedAt = time.Now() // Update timestamp

		// Execute the rename operation
		result, err := noteService.RenameNote(ctx, job.UserID, noteID, newTitle)
		if err != nil {
			return fmt.Errorf("failed to rename note: %w", err)
		}

		job.Progress = 1.0
		job.Result = result
		return nil
	}
}
