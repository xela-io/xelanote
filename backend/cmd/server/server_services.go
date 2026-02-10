package main

import (
	"log"
	"log/slog"

	"github.com/xela-io/xelanote/internal/db"
	"github.com/xela-io/xelanote/internal/service"
)

type coreServices struct {
	note     *service.NoteService
	tfa      *service.TwoFactorService
	auth     *service.AuthService
	template *service.TemplateService
	snippet  *service.SnippetService
	user     *service.UserService
	admin    *service.AdminService
	activity *service.ActivityService
	settings *service.SettingsService
}

func initCoreServices(database *db.DB, jwtSecret []byte, dataDir string, logger *slog.Logger) coreServices {
	noteService := service.NewNoteService(database)
	tfaService := service.NewTwoFactorService(database, logger)
	authService := service.NewAuthService(database, jwtSecret, tfaService)
	templateService := service.NewTemplateService(database)
	snippetService := service.NewSnippetService(database)
	userService := service.NewUserService(database)
	adminService := service.NewAdminService(database, dataDir)
	activityService := service.NewActivityService(database)
	settingsService := service.NewSettingsService(database)

	return coreServices{
		note:     noteService,
		tfa:      tfaService,
		auth:     authService,
		template: templateService,
		snippet:  snippetService,
		user:     userService,
		admin:    adminService,
		activity: activityService,
		settings: settingsService,
	}
}

func postStartupMaintenance(activityService *service.ActivityService, database *db.DB) {
	// Cleanup old activity logs at startup
	if cleaned, err := activityService.CleanupOldActivity(); err != nil {
		log.Printf("Activity cleanup failed: %v", err)
	} else if cleaned > 0 {
		log.Printf("Cleaned up %d old activity logs", cleaned)
	}

	// Backfill due dates for existing notes (one-time after migration 042)
	if backfilled, err := database.BackfillDueDates(); err != nil {
		log.Printf("Due dates backfill failed: %v", err)
	} else if backfilled > 0 {
		log.Printf("Backfilled %d due dates from existing notes", backfilled)
	}
}

func initGraphService(database *db.DB, noteService *service.NoteService) *service.GraphService {
	graphService := service.NewGraphService(database, noteService.GetCache())
	noteService.SetGraphService(graphService)
	return graphService
}
