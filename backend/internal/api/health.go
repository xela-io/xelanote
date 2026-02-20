package api

import (
	"net/http"
	"syscall"
)

// minDiskSpaceMB is the minimum free disk space before the health check fails.
const minDiskSpaceMB = 100

// handleHealth checks DB connectivity and available disk space.
// Returns 200 "ok" when healthy, 503 with a diagnostic word otherwise.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check database connectivity
	if s.dbPing != nil {
		if err := s.dbPing(); err != nil {
			s.logger().Error("health check: database ping failed", "error", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("db_error"))
			return
		}
	}

	// Check available disk space on the data directory
	if s.dataDir != "" {
		availableMB, err := availableDiskMB(s.dataDir)
		if err == nil && availableMB < minDiskSpaceMB {
			s.logger().Error("health check: disk space low", "available_mb", availableMB, "min_mb", minDiskSpaceMB)
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("disk_low"))
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// availableDiskMB returns the available disk space in MB for the given path.
func availableDiskMB(path string) (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize) / (1024 * 1024), nil
}
