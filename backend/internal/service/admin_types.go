package service

import "github.com/xela-io/xelanote/internal/db"

// Type aliases for admin/stats-related DB types.
// Allows the API layer to reference these types without importing db directly.

type DailyCount = db.DailyCount
type DailyFloat = db.DailyFloat
type ActivityLog = db.ActivityLog
type ActivityFilter = db.ActivityFilter
