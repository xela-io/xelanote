package db

import (
	"database/sql"
	"encoding/json"
)

// ActivityLog represents a single activity log entry
type ActivityLog struct {
	ID         int              `json:"id"`
	UserID     *int             `json:"user_id"`
	Username   *string          `json:"username"` // Populated from JOIN
	Action     string           `json:"action"`
	TargetType *string          `json:"target_type"`
	TargetID   *string          `json:"target_id"`
	Details    *json.RawMessage `json:"details"`
	IPAddress  *string          `json:"ip_address"`
	UserAgent  *string          `json:"user_agent"`
	CreatedAt  string           `json:"created_at"`
}

// ActivityFilter contains filter options for activity logs
type ActivityFilter struct {
	UserID     *int
	Action     *string
	TargetType *string
	DateFrom   *string
	DateTo     *string
}

const maxUserAgentLength = 512

// LogActivity creates a new activity log entry
func (db *DB) LogActivity(userID *int, action string, targetType, targetID *string, details interface{}, ipAddress, userAgent *string) error {
	var detailsJSON []byte
	var err error

	if details != nil {
		detailsJSON, err = json.Marshal(details)
		if err != nil {
			return err
		}
	}

	// Truncate user agent if too long
	if userAgent != nil && len(*userAgent) > maxUserAgentLength {
		truncated := (*userAgent)[:maxUserAgentLength]
		userAgent = &truncated
	}

	_, err = db.Exec(`
		INSERT INTO activity_logs (user_id, action, target_type, target_id, details, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, userID, action, targetType, targetID, detailsJSON, ipAddress, userAgent)

	return err
}

// GetActivityLogs returns activity logs with pagination and optional filters
func (db *DB) GetActivityLogs(limit, offset int, filter *ActivityFilter) ([]ActivityLog, int, error) {
	// Build query with filters
	query := `
		SELECT al.id, al.user_id, u.username, al.action, al.target_type, al.target_id,
		       al.details, al.ip_address, al.user_agent, al.created_at
		FROM activity_logs al
		LEFT JOIN users u ON al.user_id = u.id
		WHERE 1=1
	`
	countQuery := `
		SELECT COUNT(*)
		FROM activity_logs al
		WHERE 1=1
	`

	var args []interface{}
	var countArgs []interface{}

	if filter != nil {
		if filter.UserID != nil {
			query += " AND al.user_id = ?"
			countQuery += " AND al.user_id = ?"
			args = append(args, *filter.UserID)
			countArgs = append(countArgs, *filter.UserID)
		}
		if filter.Action != nil && *filter.Action != "" {
			query += " AND al.action = ?"
			countQuery += " AND al.action = ?"
			args = append(args, *filter.Action)
			countArgs = append(countArgs, *filter.Action)
		}
		if filter.TargetType != nil && *filter.TargetType != "" {
			query += " AND al.target_type = ?"
			countQuery += " AND al.target_type = ?"
			args = append(args, *filter.TargetType)
			countArgs = append(countArgs, *filter.TargetType)
		}
		if filter.DateFrom != nil && *filter.DateFrom != "" {
			query += " AND al.created_at >= ?"
			countQuery += " AND al.created_at >= ?"
			args = append(args, *filter.DateFrom)
			countArgs = append(countArgs, *filter.DateFrom)
		}
		if filter.DateTo != nil && *filter.DateTo != "" {
			query += " AND al.created_at <= ?"
			countQuery += " AND al.created_at <= ?"
			args = append(args, *filter.DateTo)
			countArgs = append(countArgs, *filter.DateTo)
		}
	}

	query += " ORDER BY al.created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	// Get total count
	var total int
	err := db.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// Get logs
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []ActivityLog
	for rows.Next() {
		var log ActivityLog
		var details []byte
		err := rows.Scan(
			&log.ID, &log.UserID, &log.Username, &log.Action, &log.TargetType, &log.TargetID,
			&details, &log.IPAddress, &log.UserAgent, &log.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}

		// Convert details to RawMessage if valid JSON
		if details != nil && json.Valid(details) {
			raw := json.RawMessage(details)
			log.Details = &raw
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetRecentActivity returns the most recent activity logs
func (db *DB) GetRecentActivity(limit int) ([]ActivityLog, error) {
	rows, err := db.Query(`
		SELECT al.id, al.user_id, u.username, al.action, al.target_type, al.target_id,
		       al.details, al.ip_address, al.user_agent, al.created_at
		FROM activity_logs al
		LEFT JOIN users u ON al.user_id = u.id
		ORDER BY al.created_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ActivityLog
	for rows.Next() {
		var log ActivityLog
		var details []byte
		err := rows.Scan(
			&log.ID, &log.UserID, &log.Username, &log.Action, &log.TargetType, &log.TargetID,
			&details, &log.IPAddress, &log.UserAgent, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if details != nil && json.Valid(details) {
			raw := json.RawMessage(details)
			log.Details = &raw
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// CleanupOldActivity removes activity logs older than the specified retention days
// If retentionDays is 0, no cleanup is performed
func (db *DB) CleanupOldActivity(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}

	result, err := db.Exec(`
		DELETE FROM activity_logs
		WHERE created_at < datetime('now', '-' || ? || ' days')
	`, retentionDays)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

// GetActivityByUser returns activity logs for a specific user
func (db *DB) GetActivityByUser(userID int, limit int) ([]ActivityLog, error) {
	rows, err := db.Query(`
		SELECT al.id, al.user_id, u.username, al.action, al.target_type, al.target_id,
		       al.details, al.ip_address, al.user_agent, al.created_at
		FROM activity_logs al
		LEFT JOIN users u ON al.user_id = u.id
		WHERE al.user_id = ?
		ORDER BY al.created_at DESC
		LIMIT ?
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []ActivityLog
	for rows.Next() {
		var log ActivityLog
		var details []byte
		err := rows.Scan(
			&log.ID, &log.UserID, &log.Username, &log.Action, &log.TargetType, &log.TargetID,
			&details, &log.IPAddress, &log.UserAgent, &log.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if details != nil && json.Valid(details) {
			raw := json.RawMessage(details)
			log.Details = &raw
		}

		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return logs, nil
}

// GetDistinctActions returns all distinct action types from the activity logs
func (db *DB) GetDistinctActions() ([]string, error) {
	rows, err := db.Query("SELECT DISTINCT action FROM activity_logs ORDER BY action")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var actions []string
	for rows.Next() {
		var action string
		if err := rows.Scan(&action); err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return actions, nil
}

// CountActivityToday returns the count of activity logs created today
func (db *DB) CountActivityToday() (int, error) {
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*)
		FROM activity_logs
		WHERE date(created_at) = date('now')
	`).Scan(&count)
	return count, err
}

// GetActivityCountByAction returns activity counts grouped by action for the last N days
func (db *DB) GetActivityCountByAction(days int) (map[string]int, error) {
	rows, err := db.Query(`
		SELECT action, COUNT(*) as count
		FROM activity_logs
		WHERE created_at >= datetime('now', '-' || ? || ' days')
		GROUP BY action
		ORDER BY count DESC
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var action string
		var count int
		if err := rows.Scan(&action, &count); err != nil {
			return nil, err
		}
		counts[action] = count
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return counts, nil
}

// GetUserActivitySummary returns a summary of user activity
type UserActivitySummary struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Count    int    `json:"count"`
}

func (db *DB) GetUserActivitySummary(days, limit int) ([]UserActivitySummary, error) {
	rows, err := db.Query(`
		SELECT al.user_id, u.username, COUNT(*) as count
		FROM activity_logs al
		JOIN users u ON al.user_id = u.id
		WHERE al.created_at >= datetime('now', '-' || ? || ' days')
		AND al.user_id IS NOT NULL
		GROUP BY al.user_id
		ORDER BY count DESC
		LIMIT ?
	`, days, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var summaries []UserActivitySummary
	for rows.Next() {
		var s UserActivitySummary
		var userID sql.NullInt64
		var username sql.NullString
		if err := rows.Scan(&userID, &username, &s.Count); err != nil {
			return nil, err
		}
		if userID.Valid {
			s.UserID = int(userID.Int64)
		}
		if username.Valid {
			s.Username = username.String
		}
		summaries = append(summaries, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return summaries, nil
}
