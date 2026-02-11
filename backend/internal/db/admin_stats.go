package db

// GetSystemStats returns basic system statistics
func (db *DB) GetSystemStats() (*SystemStats, error) {
	stats := &SystemStats{}

	// Count users
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&stats.TotalUsers)
	if err != nil {
		return nil, err
	}

	// Count notes (active only)
	err = db.QueryRow("SELECT COUNT(*) FROM notes WHERE deleted_at IS NULL").Scan(&stats.TotalNotes)
	if err != nil {
		return nil, err
	}

	// Count folders
	err = db.QueryRow("SELECT COUNT(*) FROM folders").Scan(&stats.TotalFolders)
	if err != nil {
		return nil, err
	}

	// Count tags
	err = db.QueryRow("SELECT COUNT(*) FROM tags").Scan(&stats.TotalTags)
	if err != nil {
		return nil, err
	}

	// Storage is calculated at runtime from uploads directory

	return stats, nil
}

// GetDetailedStats returns detailed statistics with time series data
func (db *DB) GetDetailedStats(days int) (*DetailedStats, error) {
	stats, err := db.GetSystemStats()
	if err != nil {
		return nil, err
	}

	detailed := &DetailedStats{
		Stats:        *stats,
		UserGrowth:   []DailyCount{},
		NoteGrowth:   []DailyCount{},
		StorageTrend: []DailyFloat{},
	}

	// User growth over last N days
	rows, err := db.Query(`
		SELECT date(created_at) as day, COUNT(*) as count
		FROM users
		WHERE created_at >= date('now', '-' || ? || ' days')
		GROUP BY date(created_at)
		ORDER BY day
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dc DailyCount
		if err := rows.Scan(&dc.Date, &dc.Count); err != nil {
			return nil, err
		}
		detailed.UserGrowth = append(detailed.UserGrowth, dc)
	}

	// Note growth over last N days
	rows, err = db.Query(`
		SELECT date(created_at) as day, COUNT(*) as count
		FROM notes
		WHERE created_at >= date('now', '-' || ? || ' days')
		AND deleted_at IS NULL
		GROUP BY date(created_at)
		ORDER BY day
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var dc DailyCount
		if err := rows.Scan(&dc.Date, &dc.Count); err != nil {
			return nil, err
		}
		detailed.NoteGrowth = append(detailed.NoteGrowth, dc)
	}

	return detailed, nil
}
