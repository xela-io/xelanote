package api

import (
	"fmt"
	"io"
	"log/slog"
	"testing"
	"time"
)

type fakeClock struct {
	now time.Time
}

func (fc *fakeClock) Now() time.Time {
	return fc.now
}

func (fc *fakeClock) Advance(d time.Duration) {
	fc.now = fc.now.Add(d)
}

func TestAccountLockout_BasicFunctionality(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	lockout := NewAccountLockout(3, 100*time.Millisecond, 1*time.Second, logger)
	lockout.nowFn = clock.Now

	identifier := "testuser@example.com"
	ip := "127.0.0.1"

	// Initially not locked
	locked, _ := lockout.IsLocked(identifier, ip)
	if locked {
		t.Error("Account should not be locked initially")
	}

	// Should have max attempts remaining
	remaining := lockout.GetRemainingAttempts(identifier)
	if remaining != 3 {
		t.Errorf("Expected 3 remaining attempts, got %d", remaining)
	}

	// Record 2 failures - should not lock yet
	lockout.RecordFailure(identifier, ip)
	lockout.RecordFailure(identifier, ip)

	locked, _ = lockout.IsLocked(identifier, ip)
	if locked {
		t.Error("Account should not be locked after 2 failures")
	}

	remaining = lockout.GetRemainingAttempts(identifier)
	if remaining != 1 {
		t.Errorf("Expected 1 remaining attempt, got %d", remaining)
	}

	// 3rd failure should trigger global lockout (threshold=3)
	nowLocked, duration := lockout.RecordFailure(identifier, ip)
	if !nowLocked {
		t.Error("Account should be locked after 3 failures")
	}
	if duration < 100*time.Millisecond {
		t.Errorf("Lockout duration should be at least 100ms, got %v", duration)
	}

	// Verify account is locked
	locked, _ = lockout.IsLocked(identifier, ip)
	if !locked {
		t.Error("Account should be locked")
	}

	// Wait for lockout to expire
	clock.Advance(150 * time.Millisecond)

	// Should be unlocked now
	locked, _ = lockout.IsLocked(identifier, ip)
	if locked {
		t.Error("Account should be unlocked after lockout expires")
	}
}

func TestAccountLockout_ExponentialBackoff(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	lockout := NewAccountLockout(2, 50*time.Millisecond, 500*time.Millisecond, logger)
	lockout.nowFn = clock.Now

	identifier := "testuser2@example.com"
	ip := "127.0.0.1"

	// First lockout: 50ms
	lockout.RecordFailure(identifier, ip)
	_, duration1 := lockout.RecordFailure(identifier, ip)
	if duration1 < 50*time.Millisecond || duration1 > 100*time.Millisecond {
		t.Errorf("First lockout should be ~50ms, got %v", duration1)
	}

	// Wait for lockout to expire
	clock.Advance(60 * time.Millisecond)

	// Second lockout (3rd failure): should be ~100ms (doubled)
	_, duration2 := lockout.RecordFailure(identifier, ip)
	if duration2 < 100*time.Millisecond || duration2 > 150*time.Millisecond {
		t.Errorf("Second lockout should be ~100ms, got %v", duration2)
	}

	// Wait for lockout to expire
	clock.Advance(110 * time.Millisecond)

	// Third lockout (4th failure): should be ~200ms (doubled again)
	_, duration3 := lockout.RecordFailure(identifier, ip)
	if duration3 < 200*time.Millisecond || duration3 > 250*time.Millisecond {
		t.Errorf("Third lockout should be ~200ms, got %v", duration3)
	}
}

func TestAccountLockout_MaxLockoutDuration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	lockout := NewAccountLockout(1, 100*time.Millisecond, 200*time.Millisecond, logger)
	lockout.nowFn = clock.Now

	identifier := "testuser3@example.com"
	ip := "127.0.0.1"

	// First lockout: 100ms
	_, duration1 := lockout.RecordFailure(identifier, ip)
	if duration1 != 100*time.Millisecond {
		t.Errorf("First lockout should be 100ms, got %v", duration1)
	}

	clock.Advance(110 * time.Millisecond)

	// Second lockout would be 200ms, should be capped at max
	_, duration2 := lockout.RecordFailure(identifier, ip)
	if duration2 != 200*time.Millisecond {
		t.Errorf("Second lockout should be capped at 200ms, got %v", duration2)
	}

	clock.Advance(210 * time.Millisecond)

	// Third lockout would be 400ms, but should be capped at max 200ms
	_, duration3 := lockout.RecordFailure(identifier, ip)
	if duration3 != 200*time.Millisecond {
		t.Errorf("Third lockout should be capped at 200ms, got %v", duration3)
	}
}

func TestAccountLockout_SuccessResetsCounter(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	lockout := NewAccountLockout(3, 100*time.Millisecond, 1*time.Second, logger)
	lockout.nowFn = clock.Now

	identifier := "testuser4@example.com"
	ip := "127.0.0.1"

	// Record 2 failures
	lockout.RecordFailure(identifier, ip)
	lockout.RecordFailure(identifier, ip)

	remaining := lockout.GetRemainingAttempts(identifier)
	if remaining != 1 {
		t.Errorf("Expected 1 remaining attempt, got %d", remaining)
	}

	// Successful login should reset counter
	lockout.RecordSuccess(identifier)

	remaining = lockout.GetRemainingAttempts(identifier)
	if remaining != 3 {
		t.Errorf("Expected 3 remaining attempts after success, got %d", remaining)
	}

	// Should not be locked
	locked, _ := lockout.IsLocked(identifier, ip)
	if locked {
		t.Error("Account should not be locked after successful login")
	}
}

func TestAccountLockout_DifferentIdentifiers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	lockout := NewAccountLockout(2, 100*time.Millisecond, 1*time.Second, logger)
	lockout.nowFn = clock.Now

	user1 := "user1@example.com"
	user2 := "user2@example.com"
	ip := "127.0.0.1"

	// Lock user1
	lockout.RecordFailure(user1, ip)
	lockout.RecordFailure(user1, ip)

	locked1, _ := lockout.IsLocked(user1, ip)
	locked2, _ := lockout.IsLocked(user2, ip)

	if !locked1 {
		t.Error("user1 should be locked")
	}
	if locked2 {
		t.Error("user2 should not be locked")
	}
}

func TestPerIPLockout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	// Global threshold 10, per-IP threshold is 5
	lockout := NewAccountLockout(10, 100*time.Millisecond, 1*time.Second, logger)
	lockout.nowFn = clock.Now

	identifier := "victim@example.com"
	ip1 := "10.0.0.1"
	ip2 := "10.0.0.2"

	// 5 failures from ip1 -> should trigger per-IP lockout
	for i := 0; i < 4; i++ {
		locked, _ := lockout.RecordFailure(identifier, ip1)
		if locked {
			t.Errorf("Should not be locked after %d failures from ip1", i+1)
		}
	}
	locked, _ := lockout.RecordFailure(identifier, ip1)
	if !locked {
		t.Error("ip1 should be locked after 5 per-IP failures")
	}

	// ip1 should be locked
	isLocked, _ := lockout.IsLocked(identifier, ip1)
	if !isLocked {
		t.Error("ip1 should be locked for this account")
	}

	// ip2 should NOT be locked
	isLocked, _ = lockout.IsLocked(identifier, ip2)
	if isLocked {
		t.Error("ip2 should not be locked (different IP, no per-IP lockout)")
	}

	// ip2 can still make attempts
	locked, _ = lockout.RecordFailure(identifier, ip2)
	if locked {
		t.Error("ip2 should not be locked after 1 failure")
	}
}

func TestGlobalLockoutDistributed(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	// Global threshold 10, per-IP threshold is 5
	lockout := NewAccountLockout(10, 100*time.Millisecond, 1*time.Second, logger)
	lockout.nowFn = clock.Now

	identifier := "victim@example.com"

	// 10 different IPs with 1 failure each -> should trigger global lockout
	for i := 0; i < 9; i++ {
		ip := fmt.Sprintf("10.0.0.%d", i+1)
		locked, _ := lockout.RecordFailure(identifier, ip)
		if locked {
			t.Errorf("Should not be locked after %d distributed failures", i+1)
		}
	}

	// 10th failure triggers global lockout
	locked, _ := lockout.RecordFailure(identifier, "10.0.0.10")
	if !locked {
		t.Error("Should be globally locked after 10 distributed failures")
	}

	// ALL IPs should now be locked
	for i := 1; i <= 10; i++ {
		ip := fmt.Sprintf("10.0.0.%d", i)
		isLocked, _ := lockout.IsLocked(identifier, ip)
		if !isLocked {
			t.Errorf("IP %s should be locked due to global lockout", ip)
		}
	}

	// Even a new IP should be locked
	isLocked, _ := lockout.IsLocked(identifier, "192.168.1.1")
	if !isLocked {
		t.Error("Even new IPs should be locked due to global lockout")
	}
}

func TestPerIPDoesNotAffectGlobal(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	// Global threshold 10, per-IP threshold is 5
	lockout := NewAccountLockout(10, 100*time.Millisecond, 1*time.Second, logger)
	lockout.nowFn = clock.Now

	identifier := "victim@example.com"
	ip1 := "10.0.0.1"

	// 4 failures from ip1 -> no lockout at all
	for i := 0; i < 4; i++ {
		lockout.RecordFailure(identifier, ip1)
	}

	// Global should still have 6 remaining
	remaining := lockout.GetRemainingAttempts(identifier)
	if remaining != 6 {
		t.Errorf("Expected 6 global remaining attempts, got %d", remaining)
	}
}

func TestCleanupRemovesExpiredIPEntries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	lockout := NewAccountLockout(10, 50*time.Millisecond, 100*time.Millisecond, logger)
	lockout.nowFn = clock.Now

	identifier := "cleanup@example.com"
	ip := "10.0.0.1"

	// Record a failure
	lockout.RecordFailure(identifier, ip)

	// Manually set lastAttempt to 3 hours ago to trigger cleanup
	lockout.mu.Lock()
	entry := lockout.attempts[identifier]
	entry.lastAttempt = clock.Now().Add(-3 * time.Hour)
	if ipEntry, ok := entry.ipAttempts[ip]; ok {
		ipEntry.lastAttempt = clock.Now().Add(-3 * time.Hour)
	}
	lockout.mu.Unlock()

	// Run cleanup
	lockout.cleanup()

	// Entry should be removed
	lockout.mu.RLock()
	_, exists := lockout.attempts[identifier]
	lockout.mu.RUnlock()
	if exists {
		t.Error("Expired entry should have been cleaned up")
	}
}

func TestSafeLockoutDuration(t *testing.T) {
	baseLockout := 30 * time.Second
	maxLockout := 30 * time.Minute

	tests := []struct {
		name           string
		excessAttempts int
		wantPositive   bool
		wantCapped     bool
	}{
		{"zero excess", 0, true, false},
		{"small excess", 3, true, false},
		{"capped by maxLockout", 10, true, true},
		{"near overflow boundary", 29, true, true},
		{"at overflow boundary", 30, true, true},
		{"well past overflow", 64, true, true},
		{"extreme value", 1000, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := safeLockoutDuration(baseLockout, tt.excessAttempts, maxLockout)
			if tt.wantPositive && d <= 0 {
				t.Errorf("safeLockoutDuration(%d) = %v, want positive duration", tt.excessAttempts, d)
			}
			if tt.wantCapped && d != maxLockout {
				t.Errorf("safeLockoutDuration(%d) = %v, want capped at %v", tt.excessAttempts, d, maxLockout)
			}
		})
	}
}

func TestAccountLockout_OverflowProtection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	lockout := NewAccountLockout(5, 30*time.Second, 30*time.Minute, logger)
	lockout.nowFn = clock.Now

	identifier := "overflow@example.com"
	ip := "127.0.0.1"

	// Simulate 50 failed attempts (well past overflow boundary of ~29 excess)
	for i := 0; i < 50; i++ {
		_, duration := lockout.RecordFailure(identifier, ip)
		if duration < 0 {
			t.Fatalf("Lockout duration became negative after %d attempts: %v", i+1, duration)
		}
		// Advance past lockout to allow next attempt
		clock.Advance(31 * time.Minute)
	}

	// After many attempts, lockout should still work (capped at maxLockout)
	locked, duration := lockout.RecordFailure(identifier, ip)
	if !locked {
		t.Error("Account should still be locked after 51 failures")
	}
	if duration != 30*time.Minute {
		t.Errorf("Lockout duration should be capped at 30min, got %v", duration)
	}
}

func TestAccountLockout_CooldownResetsCounters(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	clock := &fakeClock{now: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)}
	lockout := NewAccountLockout(3, 100*time.Millisecond, 1*time.Second, logger)
	lockout.nowFn = clock.Now

	identifier := "cooldown@example.com"
	ip := "127.0.0.1"

	lockout.RecordFailure(identifier, ip)
	lockout.RecordFailure(identifier, ip)

	remaining := lockout.GetRemainingAttempts(identifier)
	if remaining != 1 {
		t.Errorf("Expected 1 remaining attempt, got %d", remaining)
	}

	// Advance beyond cooldown window to reset counters
	clock.Advance(2 * time.Hour)

	lockout.RecordFailure(identifier, ip)
	remaining = lockout.GetRemainingAttempts(identifier)
	if remaining != 2 {
		t.Errorf("Expected remaining attempts to reset after cooldown, got %d", remaining)
	}
}
