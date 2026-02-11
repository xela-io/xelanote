package db

// SystemStats holds basic system statistics
type SystemStats struct {
	TotalUsers    int     `json:"total_users"`
	TotalNotes    int     `json:"total_notes"`
	TotalFolders  int     `json:"total_folders"`
	TotalTags     int     `json:"total_tags"`
	StorageUsedMB float64 `json:"storage_used_mb"`
}

// DetailedStats holds detailed statistics with time series data
type DetailedStats struct {
	Stats        SystemStats  `json:"stats"`
	UserGrowth   []DailyCount `json:"user_growth"`
	NoteGrowth   []DailyCount `json:"note_growth"`
	StorageTrend []DailyFloat `json:"storage_trend"`
}

// DailyCount holds a date and count pair
type DailyCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// DailyFloat holds a date and float value pair
type DailyFloat struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// AdminUser represents a user with stats for admin panel
type AdminUser struct {
	ID                 int     `json:"id"`
	Username           string  `json:"username"`
	Email              string  `json:"email"`
	IsAdmin            bool    `json:"is_admin"`
	NoteCount          int     `json:"note_count"`
	StorageMB          float64 `json:"storage_mb"`
	CreatedAt          string  `json:"created_at"`
	TOTPEnabled        bool    `json:"totp_enabled"`
	TOTPVerifiedAt     string  `json:"totp_verified_at"`
	TOTPDisabledAt     string  `json:"totp_disabled_at"`
	TOTPSetupStartedAt string  `json:"totp_setup_started_at"`
}
