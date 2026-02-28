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

func TestAdminService_GetEffectiveStorageLimitMB(t *testing.T) {
	svc, _, regularID := setupAdminTest(t)

	t.Run("falls back to global default when no per-user override", func(t *testing.T) {
		limit, err := svc.GetEffectiveStorageLimitMB(regularID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Global default is 0 (unlimited) unless changed
		if limit != 0 {
			t.Errorf("expected 0 (global default), got %d", limit)
		}
	})

	t.Run("returns per-user override when set to 0 (unlimited)", func(t *testing.T) {
		zero := 0
		if err := svc.SetUserStorageLimitMB(regularID, &zero); err != nil {
			t.Fatalf("failed to set limit: %v", err)
		}
		limit, err := svc.GetEffectiveStorageLimitMB(regularID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limit != 0 {
			t.Errorf("expected 0 (unlimited), got %d", limit)
		}
	})

	t.Run("returns per-user override when set to specific value", func(t *testing.T) {
		val := 500
		if err := svc.SetUserStorageLimitMB(regularID, &val); err != nil {
			t.Fatalf("failed to set limit: %v", err)
		}
		limit, err := svc.GetEffectiveStorageLimitMB(regularID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limit != 500 {
			t.Errorf("expected 500, got %d", limit)
		}
	})

	t.Run("falls back to global after clearing override", func(t *testing.T) {
		if err := svc.SetUserStorageLimitMB(regularID, nil); err != nil {
			t.Fatalf("failed to clear limit: %v", err)
		}
		limit, err := svc.GetEffectiveStorageLimitMB(regularID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if limit != 0 {
			t.Errorf("expected 0 (global default), got %d", limit)
		}
	})
}

func TestAdminService_SetUserStorageLimitMB(t *testing.T) {
	svc, _, regularID := setupAdminTest(t)

	t.Run("sets a positive limit", func(t *testing.T) {
		val := 200
		err := svc.SetUserStorageLimitMB(regularID, &val)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("clears limit with nil", func(t *testing.T) {
		err := svc.SetUserStorageLimitMB(regularID, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("rejects negative limit", func(t *testing.T) {
		neg := -1
		err := svc.SetUserStorageLimitMB(regularID, &neg)
		if err == nil {
			t.Fatal("expected error for negative limit")
		}
	})
}

func TestAdminService_GetUserStorageQuota(t *testing.T) {
	svc, _, regularID := setupAdminTest(t)

	t.Run("returns quota with global default", func(t *testing.T) {
		quota, err := svc.GetUserStorageQuota(regularID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if quota.IsCustom {
			t.Error("expected IsCustom=false for global default")
		}
	})

	t.Run("returns quota with per-user override", func(t *testing.T) {
		val := 1000
		if err := svc.SetUserStorageLimitMB(regularID, &val); err != nil {
			t.Fatalf("failed to set limit: %v", err)
		}
		quota, err := svc.GetUserStorageQuota(regularID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !quota.IsCustom {
			t.Error("expected IsCustom=true for per-user override")
		}
		if quota.LimitMB != 1000 {
			t.Errorf("expected LimitMB=1000, got %d", quota.LimitMB)
		}
	})

	t.Run("unlimited quota has zero percentage", func(t *testing.T) {
		zero := 0
		if err := svc.SetUserStorageLimitMB(regularID, &zero); err != nil {
			t.Fatalf("failed to set limit: %v", err)
		}
		quota, err := svc.GetUserStorageQuota(regularID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if quota.Percentage != 0 {
			t.Errorf("expected 0%% for unlimited, got %f", quota.Percentage)
		}
	})
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
