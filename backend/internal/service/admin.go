package service

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/xela-io/xelanote/internal/cache"
	"github.com/xela-io/xelanote/internal/db"
)

// Sentinel errors for admin operations.
var (
	ErrSelfDemotion = errors.New("cannot demote yourself")
	ErrSelfDeletion = errors.New("cannot delete yourself")
)

// Cache key prefixes for admin stats.
const (
	adminStatsCachePrefix = "cache:admin:stats:"
)

// AdminService handles admin-related operations
type AdminService struct {
	db      *db.DB
	dataDir string
	cache   *cache.Cache
}

// NewAdminService creates a new AdminService
func NewAdminService(database *db.DB, dataDir string) *AdminService {
	return &AdminService{
		db:      database,
		dataDir: dataDir,
		cache:   cache.New(5 * time.Minute), // 5 minute TTL for admin stats
	}
}

// Close releases background resources held by the service.
func (s *AdminService) Close() {
	if s.cache != nil {
		s.cache.Close()
	}
}

// GetSystemStats returns basic system statistics
func (s *AdminService) GetSystemStats() (*db.SystemStats, error) {
	const cacheKey = adminStatsCachePrefix + "system"
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*db.SystemStats), nil
	}

	stats, err := s.db.GetSystemStats()
	if err != nil {
		return nil, err
	}

	// Calculate storage from uploads directory
	stats.StorageUsedMB = s.calculateTotalStorageMB()

	s.cache.Set(cacheKey, stats)
	return stats, nil
}

// GetDetailedStats returns detailed statistics with time series
func (s *AdminService) GetDetailedStats() (*db.DetailedStats, error) {
	const cacheKey = adminStatsCachePrefix + "detailed"
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*db.DetailedStats), nil
	}

	detailed, err := s.db.GetDetailedStats(30) // Last 30 days
	if err != nil {
		return nil, err
	}

	// Add storage to stats
	detailed.Stats.StorageUsedMB = s.calculateTotalStorageMB()

	s.cache.Set(cacheKey, detailed)
	return detailed, nil
}

// GetAllUsers returns all users with their stats
func (s *AdminService) GetAllUsers() ([]db.AdminUser, error) {
	const cacheKey = adminStatsCachePrefix + "allusers"
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.([]db.AdminUser), nil
	}

	users, err := s.db.GetAllUsersWithStats()
	if err != nil {
		return nil, err
	}

	// Calculate storage for all users in one pass
	storageMap := s.calculateAllUserStorageMB()
	for i := range users {
		users[i].StorageMB = storageMap[users[i].ID]
	}

	s.cache.Set(cacheKey, users)
	return users, nil
}

// GetUserDetails returns detailed stats for a single user
func (s *AdminService) GetUserDetails(userID int) (*db.AdminUser, error) {
	cacheKey := fmt.Sprintf("%suser:%d", adminStatsCachePrefix, userID)
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(*db.AdminUser), nil
	}

	user, err := s.db.GetUserStats(userID)
	if err != nil {
		return nil, err
	}

	user.StorageMB = s.calculateUserStorageMB(userID)

	s.cache.Set(cacheKey, user)
	return user, nil
}

// SetUserAdmin sets the admin status of a user
// Returns error if trying to demote self
func (s *AdminService) SetUserAdmin(adminID, targetUserID int, isAdmin bool) error {
	// Prevent self-demotion
	if adminID == targetUserID && !isAdmin {
		return ErrSelfDemotion
	}

	if err := s.db.SetUserAdmin(targetUserID, isAdmin); err != nil {
		return err
	}

	s.invalidateStatsCache()
	return nil
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
	_ = os.RemoveAll(uploadDir) //nolint:gosec // best-effort cleanup, directory might not exist

	if err := s.db.DeleteUserByAdmin(targetUserID); err != nil {
		return err
	}

	s.invalidateStatsCache()
	return nil
}

// IsUserAdmin checks if a user is an admin
func (s *AdminService) IsUserAdmin(userID int) (bool, error) {
	return s.db.IsUserAdmin(userID)
}

// CountUsers returns the total number of users
func (s *AdminService) CountUsers() (int, error) {
	const cacheKey = adminStatsCachePrefix + "usercount"
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(int), nil
	}

	count, err := s.db.CountUsers()
	if err != nil {
		return 0, err
	}

	s.cache.Set(cacheKey, count)
	return count, nil
}

// GetRecentUsers returns recently created users
func (s *AdminService) GetRecentUsers(limit int) ([]db.AdminUser, error) {
	return s.db.GetRecentUsers(limit)
}

// calculateTotalStorageMB calculates total storage used in uploads directory
func (s *AdminService) calculateTotalStorageMB() float64 {
	uploadDir := filepath.Join(s.dataDir, "uploads")
	var totalSize int64

	_ = filepath.WalkDir(uploadDir, func(path string, d fs.DirEntry, err error) error { //nolint:gosec // best-effort size calculation
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

// calculateUserStorageMB calculates storage used by a specific user.
func (s *AdminService) calculateUserStorageMB(userID int) float64 {
	uploadDir := filepath.Join(s.dataDir, "uploads", strconv.Itoa(userID))
	var totalSize int64

	_ = filepath.WalkDir(uploadDir, func(path string, d fs.DirEntry, err error) error { //nolint:gosec // best-effort size calculation
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

// calculateAllUserStorageMB traverses the uploads directory once and returns
// storage usage per user ID. This avoids N separate WalkDir calls.
func (s *AdminService) calculateAllUserStorageMB() map[int]float64 {
	result := make(map[int]float64)
	uploadsDir := filepath.Join(s.dataDir, "uploads")

	entries, err := os.ReadDir(uploadsDir)
	if err != nil {
		return result // uploads dir may not exist
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		userID, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue // skip non-numeric directories
		}
		var totalSize int64
		userDir := filepath.Join(uploadsDir, entry.Name())
		_ = filepath.WalkDir(userDir, func(_ string, d fs.DirEntry, err error) error { //nolint:gosec // best-effort size calculation
			if err != nil {
				return nil
			}
			if !d.IsDir() {
				info, err := d.Info()
				if err == nil {
					totalSize += info.Size()
				}
			}
			return nil
		})
		result[userID] = float64(totalSize) / (1024 * 1024)
	}

	return result
}

// GetActiveUsersCount returns count of users active in last 30 days
func (s *AdminService) GetActiveUsersCount() (int, error) {
	const cacheKey = adminStatsCachePrefix + "activeusers"
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(int), nil
	}

	count, err := s.db.GetActiveUsersLast30Days()
	if err != nil {
		return 0, err
	}

	s.cache.Set(cacheKey, count)
	return count, nil
}

// GetNotesCreatedToday returns count of notes created today
func (s *AdminService) GetNotesCreatedToday() (int, error) {
	const cacheKey = adminStatsCachePrefix + "notestoday"
	if cached, ok := s.cache.Get(cacheKey); ok {
		return cached.(int), nil
	}

	count, err := s.db.GetNotesCreatedToday()
	if err != nil {
		return 0, err
	}

	s.cache.Set(cacheKey, count)
	return count, nil
}

// invalidateStatsCache clears all cached admin statistics.
func (s *AdminService) invalidateStatsCache() {
	s.cache.DeleteByPrefix(adminStatsCachePrefix)
}
