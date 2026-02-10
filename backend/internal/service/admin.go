package service

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/xela-io/xelanote/internal/db"
)

// Sentinel errors for admin operations.
var (
	ErrSelfDemotion = errors.New("cannot demote yourself")
	ErrSelfDeletion = errors.New("cannot delete yourself")
)

// AdminService handles admin-related operations
type AdminService struct {
	db      *db.DB
	dataDir string
}

// NewAdminService creates a new AdminService
func NewAdminService(database *db.DB, dataDir string) *AdminService {
	return &AdminService{
		db:      database,
		dataDir: dataDir,
	}
}

// GetSystemStats returns basic system statistics
func (s *AdminService) GetSystemStats() (*db.SystemStats, error) {
	stats, err := s.db.GetSystemStats()
	if err != nil {
		return nil, err
	}

	// Calculate storage from uploads directory
	stats.StorageUsedMB = s.calculateTotalStorageMB()

	return stats, nil
}

// GetDetailedStats returns detailed statistics with time series
func (s *AdminService) GetDetailedStats() (*db.DetailedStats, error) {
	detailed, err := s.db.GetDetailedStats(30) // Last 30 days
	if err != nil {
		return nil, err
	}

	// Add storage to stats
	detailed.Stats.StorageUsedMB = s.calculateTotalStorageMB()

	return detailed, nil
}

// GetAllUsers returns all users with their stats
func (s *AdminService) GetAllUsers() ([]db.AdminUser, error) {
	users, err := s.db.GetAllUsersWithStats()
	if err != nil {
		return nil, err
	}

	// Calculate storage for each user
	for i := range users {
		users[i].StorageMB = s.calculateUserStorageMB(users[i].ID)
	}

	return users, nil
}

// GetUserDetails returns detailed stats for a single user
func (s *AdminService) GetUserDetails(userID int) (*db.AdminUser, error) {
	user, err := s.db.GetUserStats(userID)
	if err != nil {
		return nil, err
	}

	user.StorageMB = s.calculateUserStorageMB(userID)

	return user, nil
}

// SetUserAdmin sets the admin status of a user
// Returns error if trying to demote self
func (s *AdminService) SetUserAdmin(adminID, targetUserID int, isAdmin bool) error {
	// Prevent self-demotion
	if adminID == targetUserID && !isAdmin {
		return ErrSelfDemotion
	}

	return s.db.SetUserAdmin(targetUserID, isAdmin)
}

// DeleteUser deletes a user and all their data
// Returns error if trying to delete self
func (s *AdminService) DeleteUser(adminID, targetUserID int) error {
	// Prevent self-deletion
	if adminID == targetUserID {
		return ErrSelfDeletion
	}

	// Delete uploads directory for the user
	uploadDir := filepath.Join(s.dataDir, "uploads", strconv.Itoa(targetUserID))
	os.RemoveAll(uploadDir) // Ignore errors - directory might not exist

	return s.db.DeleteUserByAdmin(targetUserID)
}

// IsUserAdmin checks if a user is an admin
func (s *AdminService) IsUserAdmin(userID int) (bool, error) {
	return s.db.IsUserAdmin(userID)
}

// CountUsers returns the total number of users
func (s *AdminService) CountUsers() (int, error) {
	return s.db.CountUsers()
}

// GetRecentUsers returns recently created users
func (s *AdminService) GetRecentUsers(limit int) ([]db.AdminUser, error) {
	return s.db.GetRecentUsers(limit)
}

// calculateTotalStorageMB calculates total storage used in uploads directory
func (s *AdminService) calculateTotalStorageMB() float64 {
	uploadDir := filepath.Join(s.dataDir, "uploads")
	var totalSize int64

	filepath.WalkDir(uploadDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				totalSize += info.Size()
			}
		}
		return nil
	})

	return float64(totalSize) / (1024 * 1024) // Convert to MB
}

// GetUserStorageMB returns current storage usage in MB for a user
func (s *AdminService) GetUserStorageMB(userID int) float64 {
	return s.calculateUserStorageMB(userID)
}

// calculateUserStorageMB calculates storage used by a specific user
func (s *AdminService) calculateUserStorageMB(userID int) float64 {
	uploadDir := filepath.Join(s.dataDir, "uploads", strconv.Itoa(userID))
	var totalSize int64

	filepath.WalkDir(uploadDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue on errors
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err == nil {
				totalSize += info.Size()
			}
		}
		return nil
	})

	return float64(totalSize) / (1024 * 1024) // Convert to MB
}

// GetActiveUsersCount returns count of users active in last 30 days
func (s *AdminService) GetActiveUsersCount() (int, error) {
	return s.db.GetActiveUsersLast30Days()
}

// GetNotesCreatedToday returns count of notes created today
func (s *AdminService) GetNotesCreatedToday() (int, error) {
	return s.db.GetNotesCreatedToday()
}
