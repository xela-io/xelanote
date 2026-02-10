package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/db"
)

// AdminStatsResponse represents the response for system stats
type AdminStatsResponse struct {
	TotalUsers    int     `json:"total_users"`
	TotalNotes    int     `json:"total_notes"`
	TotalFolders  int     `json:"total_folders"`
	TotalTags     int     `json:"total_tags"`
	StorageUsedMB float64 `json:"storage_used_mb"`
}

// DetailedStatsResponse represents the response for detailed stats
type DetailedStatsResponse struct {
	Stats        AdminStatsResponse `json:"stats"`
	UserGrowth   []db.DailyCount    `json:"user_growth"`
	NoteGrowth   []db.DailyCount    `json:"note_growth"`
	StorageTrend []db.DailyFloat    `json:"storage_trend"`
}

// AdminUserResponse represents a user in admin panel
type AdminUserResponse struct {
	ID                 int     `json:"id"`
	Username           string  `json:"username"`
	Email              string  `json:"email"`
	IsAdmin            bool    `json:"is_admin"`
	NoteCount          int     `json:"note_count"`
	StorageMB          float64 `json:"storage_mb"`
	CreatedAt          string  `json:"created_at"`
	TOTPEnabled        bool    `json:"totp_enabled"`
	TOTPVerifiedAt     string  `json:"totp_verified_at,omitempty"`
	TOTPDisabledAt     string  `json:"totp_disabled_at,omitempty"`
	TOTPSetupStartedAt string  `json:"totp_setup_started_at,omitempty"`
}

// SetAdminRequest represents the request to set admin status
type SetAdminRequest struct {
	IsAdmin bool `json:"is_admin"`
}

// ActivityLogsResponse represents the response for activity logs
type ActivityLogsResponse struct {
	Logs  []db.ActivityLog `json:"logs"`
	Total int              `json:"total"`
}

// SettingsResponse represents system settings
type SettingsResponse map[string]string

// UpdateSettingsRequest represents the request to update settings
type UpdateSettingsRequest map[string]string

// getAdminStats returns basic system statistics
func (s *Server) getAdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.adminService.GetSystemStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get stats")
		return
	}

	respondJSON(w, http.StatusOK, AdminStatsResponse{
		TotalUsers:    stats.TotalUsers,
		TotalNotes:    stats.TotalNotes,
		TotalFolders:  stats.TotalFolders,
		TotalTags:     stats.TotalTags,
		StorageUsedMB: stats.StorageUsedMB,
	})
}

// getDetailedStats returns detailed statistics with time series
func (s *Server) getDetailedStats(w http.ResponseWriter, r *http.Request) {
	detailed, err := s.adminService.GetDetailedStats()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get detailed stats")
		return
	}

	respondJSON(w, http.StatusOK, DetailedStatsResponse{
		Stats: AdminStatsResponse{
			TotalUsers:    detailed.Stats.TotalUsers,
			TotalNotes:    detailed.Stats.TotalNotes,
			TotalFolders:  detailed.Stats.TotalFolders,
			TotalTags:     detailed.Stats.TotalTags,
			StorageUsedMB: detailed.Stats.StorageUsedMB,
		},
		UserGrowth:   detailed.UserGrowth,
		NoteGrowth:   detailed.NoteGrowth,
		StorageTrend: detailed.StorageTrend,
	})
}

// listAllUsers returns all users with stats
func (s *Server) listAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.adminService.GetAllUsers()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get users")
		return
	}

	// Convert to response type
	var response []AdminUserResponse
	for _, u := range users {
		response = append(response, AdminUserResponse{
			ID:                 u.ID,
			Username:           u.Username,
			Email:              u.Email,
			IsAdmin:            u.IsAdmin,
			NoteCount:          u.NoteCount,
			StorageMB:          u.StorageMB,
			CreatedAt:          u.CreatedAt,
			TOTPEnabled:        u.TOTPEnabled,
			TOTPVerifiedAt:     u.TOTPVerifiedAt,
			TOTPDisabledAt:     u.TOTPDisabledAt,
			TOTPSetupStartedAt: u.TOTPSetupStartedAt,
		})
	}

	respondJSON(w, http.StatusOK, response)
}

// getUserDetails returns details for a single user
func (s *Server) getUserDetails(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	user, err := s.adminService.GetUserDetails(id)
	if err != nil {
		if err == db.ErrNotFound {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get user details")
		return
	}

	respondJSON(w, http.StatusOK, AdminUserResponse{
		ID:                 user.ID,
		Username:           user.Username,
		Email:              user.Email,
		IsAdmin:            user.IsAdmin,
		NoteCount:          user.NoteCount,
		StorageMB:          user.StorageMB,
		CreatedAt:          user.CreatedAt,
		TOTPEnabled:        user.TOTPEnabled,
		TOTPVerifiedAt:     user.TOTPVerifiedAt,
		TOTPDisabledAt:     user.TOTPDisabledAt,
		TOTPSetupStartedAt: user.TOTPSetupStartedAt,
	})
}

// toggleUserAdmin sets the admin status of a user
func (s *Server) toggleUserAdmin(w http.ResponseWriter, r *http.Request) {
	adminID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	targetID, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	var req SetAdminRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.adminService.SetUserAdmin(adminID, targetID, req.IsAdmin); err != nil {
		if err.Error() == "cannot demote yourself" {
			respondError(w, http.StatusForbidden, err.Error())
			return
		}
		if err == db.ErrNotFound {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to update admin status")
		return
	}

	// Log the activity
	ipAddress := getClientIPSafe(r)
	userAgent := r.UserAgent()
	targetUser, _ := s.adminService.GetUserDetails(targetID)
	targetUsername := ""
	if targetUser != nil {
		targetUsername = targetUser.Username
	}
	s.activityService.LogUserAdminSet(adminID, targetID, req.IsAdmin, targetUsername, ipAddress, userAgent)

	w.WriteHeader(http.StatusNoContent)
}

// deleteUserAdmin deletes a user
func (s *Server) deleteUserAdmin(w http.ResponseWriter, r *http.Request) {
	adminID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	targetID, err := strconv.Atoi(idStr)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Get username before deletion for logging
	targetUser, _ := s.adminService.GetUserDetails(targetID)
	targetUsername := ""
	if targetUser != nil {
		targetUsername = targetUser.Username
	}

	if err := s.adminService.DeleteUser(adminID, targetID); err != nil {
		if err.Error() == "cannot delete yourself" {
			respondError(w, http.StatusForbidden, err.Error())
			return
		}
		if err == db.ErrNotFound {
			respondError(w, http.StatusNotFound, "user not found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete user")
		return
	}

	// Log the activity
	ipAddress := getClientIPSafe(r)
	userAgent := r.UserAgent()
	s.activityService.LogUserDelete(adminID, targetID, targetUsername, ipAddress, userAgent)

	w.WriteHeader(http.StatusNoContent)
}

// getActivityLogs returns activity logs with pagination and filters
func (s *Server) getActivityLogs(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	pageStr := r.URL.Query().Get("page")
	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	offset := (page - 1) * limit

	// Build filter
	var filter *db.ActivityFilter
	action := r.URL.Query().Get("action")
	userIDStr := r.URL.Query().Get("user_id")
	targetType := r.URL.Query().Get("target_type")
	dateFrom := r.URL.Query().Get("date_from")
	dateTo := r.URL.Query().Get("date_to")

	if action != "" || userIDStr != "" || targetType != "" || dateFrom != "" || dateTo != "" {
		filter = &db.ActivityFilter{}
		if action != "" {
			filter.Action = &action
		}
		if userIDStr != "" {
			if uid, err := strconv.Atoi(userIDStr); err == nil {
				filter.UserID = &uid
			}
		}
		if targetType != "" {
			filter.TargetType = &targetType
		}
		if dateFrom != "" {
			filter.DateFrom = &dateFrom
		}
		if dateTo != "" {
			filter.DateTo = &dateTo
		}
	}

	logs, total, err := s.activityService.GetActivityLogs(limit, offset, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get activity logs")
		return
	}

	respondJSON(w, http.StatusOK, ActivityLogsResponse{
		Logs:  logs,
		Total: total,
	})
}

// getSettings returns all system settings
func (s *Server) getSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.settingsService.GetSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get settings")
		return
	}

	respondJSON(w, http.StatusOK, SettingsResponse(settings))
}

// updateSettings updates system settings
func (s *Server) updateSettings(w http.ResponseWriter, r *http.Request) {
	adminID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req UpdateSettingsRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req) == 0 {
		respondError(w, http.StatusBadRequest, "no settings provided")
		return
	}

	changedKeys, err := s.settingsService.UpdateSettings(map[string]string(req))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log the activity
	ipAddress := getClientIPSafe(r)
	userAgent := r.UserAgent()
	s.activityService.LogSettingsChange(adminID, changedKeys, ipAddress, userAgent)

	// Return updated settings
	settings, err := s.settingsService.GetSettings()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to get updated settings")
		return
	}

	respondJSON(w, http.StatusOK, SettingsResponse(settings))
}
