package db

import (
	"testing"
)

func TestLogActivity_Success(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)

	userID := 1
	action := "login"
	targetType := "session"
	targetID := "sess-123"
	ip := "127.0.0.1"
	ua := "TestAgent/1.0"

	err := db.LogActivity(&userID, action, &targetType, &targetID,
		map[string]string{"method": "password"}, &ip, &ua)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLogActivity_NilFields(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Should handle all nil optional fields
	err := db.LogActivity(nil, "system_startup", nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error with nil fields: %v", err)
	}
}

func TestGetActivityLogs_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	logs, total, err := db.GetActivityLogs(10, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}
	if len(logs) != 0 {
		t.Errorf("expected 0 logs, got %d", len(logs))
	}
}

func TestGetActivityLogs_WithEntries(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	userID := 1

	for _, action := range []string{"login", "logout", "note_create"} {
		if err := db.LogActivity(&userID, action, nil, nil, nil, nil, nil); err != nil {
			t.Fatalf("failed to log activity %q: %v", action, err)
		}
	}

	logs, total, err := db.GetActivityLogs(10, 0, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Errorf("expected 3 total, got %d", total)
	}
	if len(logs) != 3 {
		t.Errorf("expected 3 logs, got %d", len(logs))
	}
}

func TestGetActivityLogs_WithFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	userID := 1

	if err := db.LogActivity(&userID, "login", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to log: %v", err)
	}
	if err := db.LogActivity(&userID, "note_create", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to log: %v", err)
	}

	action := "login"
	logs, total, err := db.GetActivityLogs(10, 0, &ActivityFilter{Action: &action})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 login entry, got %d", total)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}
}

func TestGetRecentActivity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	userID := 1

	if err := db.LogActivity(&userID, "login", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to log: %v", err)
	}

	logs, err := db.GetRecentActivity(10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 recent log, got %d", len(logs))
	}
}

func TestGetActivityByUser(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	createTestUser2(t, db, 2)
	userID1 := 1
	userID2 := 2

	if err := db.LogActivity(&userID1, "login", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to log: %v", err)
	}
	if err := db.LogActivity(&userID2, "login", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to log: %v", err)
	}

	logs, err := db.GetActivityByUser(1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log for user 1, got %d", len(logs))
	}
}

func TestCleanupOldActivity(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// No-op for 0 days
	count, err := db.CleanupOldActivity(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 cleaned up, got %d", count)
	}

	// With entries (all recent, so cleanup shouldn't delete them)
	createTestUser(t, db, 1)
	userID := 1
	if err := db.LogActivity(&userID, "login", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to log: %v", err)
	}

	count, err = db.CleanupOldActivity(30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 cleaned (all recent), got %d", count)
	}
}

func TestGetDistinctActions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	userID := 1

	for _, action := range []string{"login", "logout", "login"} {
		if err := db.LogActivity(&userID, action, nil, nil, nil, nil, nil); err != nil {
			t.Fatalf("failed to log: %v", err)
		}
	}

	actions, err := db.GetDistinctActions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 2 {
		t.Errorf("expected 2 distinct actions, got %d", len(actions))
	}
}

func TestCountActivityToday(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	userID := 1

	if err := db.LogActivity(&userID, "login", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to log: %v", err)
	}

	count, err := db.CountActivityToday()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 activity today, got %d", count)
	}
}

func TestGetActivityCountByAction(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	userID := 1

	if err := db.LogActivity(&userID, "login", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to log: %v", err)
	}
	if err := db.LogActivity(&userID, "login", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to log: %v", err)
	}
	if err := db.LogActivity(&userID, "logout", nil, nil, nil, nil, nil); err != nil {
		t.Fatalf("failed to log: %v", err)
	}

	counts, err := db.GetActivityCountByAction(30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if counts["login"] != 2 {
		t.Errorf("expected 2 logins, got %d", counts["login"])
	}
	if counts["logout"] != 1 {
		t.Errorf("expected 1 logout, got %d", counts["logout"])
	}
}

func TestLogActivity_LongUserAgent(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	createTestUser(t, db, 1)
	userID := 1

	// Create a user agent longer than maxUserAgentLength (512)
	longUA := make([]byte, 1000)
	for i := range longUA {
		longUA[i] = 'A'
	}
	ua := string(longUA)

	err := db.LogActivity(&userID, "login", nil, nil, nil, nil, &ua)
	if err != nil {
		t.Fatalf("unexpected error with long user agent: %v", err)
	}
}
