package service

import (
	"strconv"

	"github.com/xela-io/xelanote/internal/db"
)

// Activity action constants
const (
	ActionLogin          = "login"
	ActionLogout         = "logout"
	ActionRegister       = "register"
	ActionNoteCreate     = "note_create"
	ActionNoteUpdate     = "note_update"
	ActionNoteDelete     = "note_delete"
	ActionNoteRestore    = "note_restore"
	ActionFolderCreate   = "folder_create"
	ActionFolderDelete   = "folder_delete"
	ActionFolderRename   = "folder_rename"
	ActionUserAdminSet   = "user_admin_set"
	ActionUserDelete     = "user_delete"
	ActionSettingsChange = "settings_change"
)

// Target type constants
const (
	TargetTypeNote     = "note"
	TargetTypeFolder   = "folder"
	TargetTypeUser     = "user"
	TargetTypeSettings = "settings"
)

// ActivityService handles activity logging operations
type ActivityService struct {
	db *db.DB
}

// NewActivityService creates a new ActivityService
func NewActivityService(database *db.DB) *ActivityService {
	return &ActivityService{
		db: database,
	}
}

// LogActivity creates a new activity log entry
func (s *ActivityService) LogActivity(userID *int, action string, targetType, targetID *string, details interface{}, ipAddress, userAgent string) error {
	var ip, ua *string
	if ipAddress != "" {
		ip = &ipAddress
	}
	if userAgent != "" {
		ua = &userAgent
	}
	return s.db.LogActivity(userID, action, targetType, targetID, details, ip, ua)
}

// LogLogin logs a login event
func (s *ActivityService) LogLogin(userID int, ipAddress, userAgent string) error {
	return s.LogActivity(&userID, ActionLogin, nil, nil, nil, ipAddress, userAgent)
}

// LogLogout logs a logout event
func (s *ActivityService) LogLogout(userID int, ipAddress, userAgent string) error {
	return s.LogActivity(&userID, ActionLogout, nil, nil, nil, ipAddress, userAgent)
}

// LogRegister logs a registration event
func (s *ActivityService) LogRegister(userID int, username, ipAddress, userAgent string) error {
	details := map[string]string{"username": username}
	return s.LogActivity(&userID, ActionRegister, nil, nil, details, ipAddress, userAgent)
}

// LogNoteCreate logs a note creation event
func (s *ActivityService) LogNoteCreate(userID int, noteID, noteTitle, ipAddress, userAgent string) error {
	targetType := TargetTypeNote
	details := map[string]string{"title": noteTitle}
	return s.LogActivity(&userID, ActionNoteCreate, &targetType, &noteID, details, ipAddress, userAgent)
}

// LogNoteUpdate logs a note update event
func (s *ActivityService) LogNoteUpdate(userID int, noteID, noteTitle, ipAddress, userAgent string) error {
	targetType := TargetTypeNote
	details := map[string]string{"title": noteTitle}
	return s.LogActivity(&userID, ActionNoteUpdate, &targetType, &noteID, details, ipAddress, userAgent)
}

// LogNoteDelete logs a note deletion event
func (s *ActivityService) LogNoteDelete(userID int, noteID, noteTitle, ipAddress, userAgent string) error {
	targetType := TargetTypeNote
	details := map[string]string{"title": noteTitle}
	return s.LogActivity(&userID, ActionNoteDelete, &targetType, &noteID, details, ipAddress, userAgent)
}

// LogNoteRestore logs a note restore event
func (s *ActivityService) LogNoteRestore(userID int, noteID, noteTitle, ipAddress, userAgent string) error {
	targetType := TargetTypeNote
	details := map[string]string{"title": noteTitle}
	return s.LogActivity(&userID, ActionNoteRestore, &targetType, &noteID, details, ipAddress, userAgent)
}

// LogFolderCreate logs a folder creation event
func (s *ActivityService) LogFolderCreate(userID int, folderID int, folderPath, ipAddress, userAgent string) error {
	targetType := TargetTypeFolder
	folderIDStr := strconv.Itoa(folderID)
	details := map[string]string{"path": folderPath}
	return s.LogActivity(&userID, ActionFolderCreate, &targetType, &folderIDStr, details, ipAddress, userAgent)
}

// LogFolderDelete logs a folder deletion event
func (s *ActivityService) LogFolderDelete(userID int, folderID int, folderPath, ipAddress, userAgent string) error {
	targetType := TargetTypeFolder
	folderIDStr := strconv.Itoa(folderID)
	details := map[string]string{"path": folderPath}
	return s.LogActivity(&userID, ActionFolderDelete, &targetType, &folderIDStr, details, ipAddress, userAgent)
}

// LogUserAdminSet logs an admin status change event
func (s *ActivityService) LogUserAdminSet(adminID, targetUserID int, isAdmin bool, targetUsername, ipAddress, userAgent string) error {
	targetType := TargetTypeUser
	targetIDStr := strconv.Itoa(targetUserID)
	details := map[string]interface{}{
		"target_username": targetUsername,
		"is_admin":        isAdmin,
	}
	return s.LogActivity(&adminID, ActionUserAdminSet, &targetType, &targetIDStr, details, ipAddress, userAgent)
}

// LogUserDelete logs a user deletion event (by admin)
func (s *ActivityService) LogUserDelete(adminID, targetUserID int, targetUsername, ipAddress, userAgent string) error {
	targetType := TargetTypeUser
	targetIDStr := strconv.Itoa(targetUserID)
	details := map[string]string{"target_username": targetUsername}
	return s.LogActivity(&adminID, ActionUserDelete, &targetType, &targetIDStr, details, ipAddress, userAgent)
}

// LogSettingsChange logs a settings change event
func (s *ActivityService) LogSettingsChange(adminID int, changedKeys []string, ipAddress, userAgent string) error {
	targetType := TargetTypeSettings
	details := map[string]interface{}{"changed_keys": changedKeys}
	return s.LogActivity(&adminID, ActionSettingsChange, &targetType, nil, details, ipAddress, userAgent)
}

// GetActivityLogs returns activity logs with pagination and filters
func (s *ActivityService) GetActivityLogs(limit, offset int, filter *db.ActivityFilter) ([]db.ActivityLog, int, error) {
	return s.db.GetActivityLogs(limit, offset, filter)
}

// GetRecentActivity returns recent activity logs
func (s *ActivityService) GetRecentActivity(limit int) ([]db.ActivityLog, error) {
	return s.db.GetRecentActivity(limit)
}

// GetActivityByUser returns activity logs for a specific user
func (s *ActivityService) GetActivityByUser(userID int, limit int) ([]db.ActivityLog, error) {
	return s.db.GetActivityByUser(userID, limit)
}

// CleanupOldActivity removes old activity logs based on retention setting
func (s *ActivityService) CleanupOldActivity() (int64, error) {
	retentionDays, err := s.db.GetActivityRetentionDays()
	if err != nil {
		return 0, err
	}

	return s.db.CleanupOldActivity(retentionDays)
}

// GetDistinctActions returns all distinct action types
func (s *ActivityService) GetDistinctActions() ([]string, error) {
	return s.db.GetDistinctActions()
}

// GetActivityCountByAction returns activity counts grouped by action
func (s *ActivityService) GetActivityCountByAction(days int) (map[string]int, error) {
	return s.db.GetActivityCountByAction(days)
}

// GetUserActivitySummary returns a summary of user activity
func (s *ActivityService) GetUserActivitySummary(days, limit int) ([]db.UserActivitySummary, error) {
	return s.db.GetUserActivitySummary(days, limit)
}
