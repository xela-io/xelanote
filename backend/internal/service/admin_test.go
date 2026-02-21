package service

import (
	"testing"
)

func setupAdminTest(t *testing.T) (*AdminService, int, int) {
	t.Helper()

	database := setupTestDB(t)
	svc := NewAdminService(database, t.TempDir())

	// Create admin user
	admin := createTestUser(t, database, "admin")
	if err := database.SetUserAdmin(admin.ID, true); err != nil {
		t.Fatalf("failed to set admin: %v", err)
	}

	// Create regular user
	regular := createTestUser(t, database, "regular")

	return svc, admin.ID, regular.ID
}

func TestAdminService_SetUserAdmin(t *testing.T) {
	svc, adminID, regularID := setupAdminTest(t)

	t.Run("promotes user to admin", func(t *testing.T) {
		err := svc.SetUserAdmin(adminID, regularID, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		isAdmin, err := svc.IsUserAdmin(regularID)
		if err != nil {
			t.Fatalf("failed to check admin: %v", err)
		}
		if !isAdmin {
			t.Error("expected user to be admin")
		}
	})

	t.Run("demotes user from admin", func(t *testing.T) {
		err := svc.SetUserAdmin(adminID, regularID, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		isAdmin, err := svc.IsUserAdmin(regularID)
		if err != nil {
			t.Fatalf("failed to check admin: %v", err)
		}
		if isAdmin {
			t.Error("expected user to not be admin")
		}
	})

	t.Run("prevents self-demotion", func(t *testing.T) {
		err := svc.SetUserAdmin(adminID, adminID, false)
		if err != ErrSelfDemotion {
			t.Errorf("expected ErrSelfDemotion, got: %v", err)
		}
	})

	t.Run("allows self-promotion (no-op)", func(t *testing.T) {
		err := svc.SetUserAdmin(adminID, adminID, true)
		if err != nil {
			t.Fatalf("self-promotion should not error: %v", err)
		}
	})
}

func TestAdminService_DeleteUser(t *testing.T) {
	svc, adminID, regularID := setupAdminTest(t)

	t.Run("prevents self-deletion", func(t *testing.T) {
		err := svc.DeleteUser(adminID, adminID)
		if err != ErrSelfDeletion {
			t.Errorf("expected ErrSelfDeletion, got: %v", err)
		}
	})

	t.Run("deletes another user", func(t *testing.T) {
		err := svc.DeleteUser(adminID, regularID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAdminService_IsUserAdmin(t *testing.T) {
	svc, adminID, regularID := setupAdminTest(t)

	t.Run("returns true for admin", func(t *testing.T) {
		isAdmin, err := svc.IsUserAdmin(adminID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !isAdmin {
			t.Error("expected true for admin user")
		}
	})

	t.Run("returns false for regular user", func(t *testing.T) {
		isAdmin, err := svc.IsUserAdmin(regularID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if isAdmin {
			t.Error("expected false for regular user")
		}
	})
}

func TestAdminService_CountUsers(t *testing.T) {
	svc, _, _ := setupAdminTest(t)

	count, err := svc.CountUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// setupAdminTest creates 2 users (admin + regular)
	if count < 2 {
		t.Errorf("expected at least 2 users, got %d", count)
	}
}

func TestAdminService_GetSystemStats(t *testing.T) {
	svc, _, _ := setupAdminTest(t)

	stats, err := svc.GetSystemStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.TotalUsers < 2 {
		t.Errorf("expected at least 2 users in stats, got %d", stats.TotalUsers)
	}
}

func TestAdminService_GetAllUsers(t *testing.T) {
	svc, _, _ := setupAdminTest(t)

	users, err := svc.GetAllUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) < 2 {
		t.Errorf("expected at least 2 users, got %d", len(users))
	}
}

func TestAdminService_GetUserDetails(t *testing.T) {
	svc, adminID, _ := setupAdminTest(t)

	user, err := svc.GetUserDetails(adminID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected non-nil user details")
	}
	if user.Username != "admin" {
		t.Errorf("expected username 'admin', got %q", user.Username)
	}
}

func TestAdminService_StorageCalculation(t *testing.T) {
	svc, _, _ := setupAdminTest(t)

	// With empty temp dir, storage should be 0
	mb := svc.GetUserStorageMB(999)
	if mb != 0 {
		t.Errorf("expected 0 MB for nonexistent user dir, got %f", mb)
	}
}

func TestAdminService_CacheInvalidation(t *testing.T) {
	svc, adminID, regularID := setupAdminTest(t)

	// First call populates cache
	count1, err := svc.CountUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Delete a user (should invalidate cache)
	if err := svc.DeleteUser(adminID, regularID); err != nil {
		t.Fatalf("failed to delete user: %v", err)
	}

	// Second call should reflect the deletion
	count2, err := svc.CountUsers()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count2 >= count1 {
		t.Errorf("expected fewer users after deletion, got %d (was %d)", count2, count1)
	}
}
