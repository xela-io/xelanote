package api

import (
	"log/slog"
	"sync"
	"time"
)

const ipLockoutThreshold = 5

// AccountLockout tracks failed login attempts per account identifier (username/email)
// with hybrid IP-based and global lockout. Per-IP lockout triggers at ipLockoutThreshold
// attempts from the same IP. Global lockout triggers at maxAttempts from any IP combination.
type AccountLockout struct {
	mu           sync.RWMutex
	attempts     map[string]*lockoutEntry
	maxAttempts  int           // Global max failed attempts before lockout
	baseLockout  time.Duration // Initial lockout duration (doubles each time)
	maxLockout   time.Duration // Maximum lockout duration
	cleanupEvery time.Duration // How often to clean up expired entries
	logger       *slog.Logger  // Logger for security events
	nowFn        func() time.Time
}

type lockoutEntry struct {
	failedAttempts int       // Global failed attempts
	lockedUntil    time.Time // Global lockout
	lastAttempt    time.Time
	ipAttempts     map[string]*ipLockoutEntry // Per-IP tracking
}

type ipLockoutEntry struct {
	failedAttempts int
	lockedUntil    time.Time
	lastAttempt    time.Time
}

// NewAccountLockout creates a new account lockout tracker.
// maxAttempts is the global threshold (e.g. 10). Per-IP threshold is fixed at 5.
func NewAccountLockout(maxAttempts int, baseLockout, maxLockout time.Duration, logger *slog.Logger) *AccountLockout {
	al := &AccountLockout{
		attempts:     make(map[string]*lockoutEntry),
		maxAttempts:  maxAttempts,
		baseLockout:  baseLockout,
		maxLockout:   maxLockout,
		cleanupEvery: 5 * time.Minute,
		logger:       logger,
		nowFn:        time.Now,
	}

	// Start background cleanup goroutine
	go al.cleanupLoop()

	return al
}

func (al *AccountLockout) now() time.Time {
	if al.nowFn != nil {
		return al.nowFn()
	}
	return time.Now()
}

// IsLocked checks if an account is currently locked out for the given IP.
// Returns true if either the global lockout OR the per-IP lockout is active.
func (al *AccountLockout) IsLocked(identifier, ip string) (bool, time.Duration) {
	al.mu.RLock()
	defer al.mu.RUnlock()

	entry, exists := al.attempts[identifier]
	if !exists {
		return false, 0
	}

	now := al.now()

	// Check global lockout
	if now.Before(entry.lockedUntil) {
		return true, entry.lockedUntil.Sub(now)
	}

	// Check per-IP lockout
	if ipEntry, ok := entry.ipAttempts[ip]; ok {
		if now.Before(ipEntry.lockedUntil) {
			return true, ipEntry.lockedUntil.Sub(now)
		}
	}

	return false, 0
}

// RecordFailure records a failed login attempt for an account from the given IP.
// Returns (isNowLocked, lockoutDuration).
func (al *AccountLockout) RecordFailure(identifier, ip string) (bool, time.Duration) {
	al.mu.Lock()
	defer al.mu.Unlock()

	entry, exists := al.attempts[identifier]
	if !exists {
		entry = &lockoutEntry{
			ipAttempts: make(map[string]*ipLockoutEntry),
		}
		al.attempts[identifier] = entry
	}
	if entry.ipAttempts == nil {
		entry.ipAttempts = make(map[string]*ipLockoutEntry)
	}

	now := al.now()

	// Reset global counter if previous lockout has expired and cooldown passed (1 hour)
	if now.After(entry.lockedUntil) && now.Sub(entry.lastAttempt) > time.Hour {
		entry.failedAttempts = 0
		entry.ipAttempts = make(map[string]*ipLockoutEntry)
	}

	// Update global counter
	entry.failedAttempts++
	entry.lastAttempt = now

	// Update per-IP counter
	ipEntry, ok := entry.ipAttempts[ip]
	if !ok {
		ipEntry = &ipLockoutEntry{}
		entry.ipAttempts[ip] = ipEntry
	}

	// Reset per-IP counter if its lockout expired and cooldown passed
	if now.After(ipEntry.lockedUntil) && now.Sub(ipEntry.lastAttempt) > time.Hour {
		ipEntry.failedAttempts = 0
	}

	ipEntry.failedAttempts++
	ipEntry.lastAttempt = now

	// Check global lockout first (higher priority)
	if entry.failedAttempts >= al.maxAttempts {
		excessAttempts := entry.failedAttempts - al.maxAttempts
		lockoutDuration := al.baseLockout * (1 << excessAttempts)

		if lockoutDuration > al.maxLockout {
			lockoutDuration = al.maxLockout
		}

		entry.lockedUntil = now.Add(lockoutDuration)

		if al.logger != nil {
			al.logger.Warn("account_locked",
				slog.String("identifier_hash", hashIdentifier(identifier)),
				slog.String("event", "global_account_lockout"),
				slog.Int("failed_attempts", entry.failedAttempts),
				slog.Duration("lockout_duration", lockoutDuration))
		}

		return true, lockoutDuration
	}

	// Check per-IP lockout
	if ipEntry.failedAttempts >= ipLockoutThreshold {
		excessAttempts := ipEntry.failedAttempts - ipLockoutThreshold
		lockoutDuration := al.baseLockout * (1 << excessAttempts)

		if lockoutDuration > al.maxLockout {
			lockoutDuration = al.maxLockout
		}

		ipEntry.lockedUntil = now.Add(lockoutDuration)

		if al.logger != nil {
			al.logger.Warn("account_locked",
				slog.String("identifier_hash", hashIdentifier(identifier)),
				slog.String("event", "ip_account_lockout"),
				slog.Int("ip_failed_attempts", ipEntry.failedAttempts),
				slog.Int("global_failed_attempts", entry.failedAttempts),
				slog.Duration("lockout_duration", lockoutDuration))
		}

		return true, lockoutDuration
	}

	return false, 0
}

// RecordSuccess clears the failure counter for an account after successful login.
func (al *AccountLockout) RecordSuccess(identifier string) {
	al.mu.Lock()
	defer al.mu.Unlock()

	delete(al.attempts, identifier)
}

// GetRemainingAttempts returns how many global attempts remain before lockout.
func (al *AccountLockout) GetRemainingAttempts(identifier string) int {
	al.mu.RLock()
	defer al.mu.RUnlock()

	entry, exists := al.attempts[identifier]
	if !exists {
		return al.maxAttempts
	}

	remaining := al.maxAttempts - entry.failedAttempts
	if remaining < 0 {
		return 0
	}
	return remaining
}

// cleanupLoop periodically removes expired lockout entries to prevent memory leaks.
func (al *AccountLockout) cleanupLoop() {
	ticker := time.NewTicker(al.cleanupEvery)
	defer ticker.Stop()

	for range ticker.C {
		al.cleanup()
	}
}

func (al *AccountLockout) cleanup() {
	al.mu.Lock()
	defer al.mu.Unlock()

	now := al.now()
	for identifier, entry := range al.attempts {
		// Clean up expired per-IP entries
		for ip, ipEntry := range entry.ipAttempts {
			if now.After(ipEntry.lockedUntil) && now.Sub(ipEntry.lastAttempt) > 2*time.Hour {
				delete(entry.ipAttempts, ip)
			}
		}

		// Remove account entries that are not locked, have no IP entries, and no recent activity
		if now.After(entry.lockedUntil) && now.Sub(entry.lastAttempt) > 2*time.Hour && len(entry.ipAttempts) == 0 {
			delete(al.attempts, identifier)
		}
	}
}
