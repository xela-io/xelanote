package api

import (
	"log/slog"
	"sync"
	"time"

	"github.com/xela-io/xelanote/internal/service"
)

// LockoutStore abstracts the DB methods used for persistent lockout state.
// Satisfied by *db.DB. Defined here so the API layer doesn't import db directly.
type LockoutStore interface {
	UpsertLockout(rec service.LockoutRecord) error
	DeleteLockout(identifierHash string) error
	LoadActiveLockouts() ([]service.LockoutRecord, error)
}

const ipLockoutThreshold = 5

// maxExponentShift caps the bit-shift exponent in exponential backoff to prevent
// integer overflow. With baseLockout=30s and int64, overflow occurs at shift≈29.
// Capping at 20 allows up to ~30s * 2^20 ≈ 364 days before the maxLockout cap applies.
const maxExponentShift = 20

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
	database     LockoutStore // Optional: persists lockout state across restarts (F2-06)
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

// SetDB enables persistent lockout storage (F2-06).
// Must be called before any lockout operations. Loads existing state from DB.
func (al *AccountLockout) SetDB(database LockoutStore) {
	al.mu.Lock()
	defer al.mu.Unlock()

	al.database = database
	al.loadFromDB()
}

// loadFromDB restores lockout state from the database on startup.
// Must be called while holding al.mu.
func (al *AccountLockout) loadFromDB() {
	if al.database == nil {
		return
	}

	records, err := al.database.LoadActiveLockouts()
	if err != nil {
		if al.logger != nil {
			al.logger.Warn("failed to load lockout state from DB", slog.String("error", err.Error()))
		}
		return
	}

	for _, rec := range records {
		entry, exists := al.attempts[rec.IdentifierHash]
		if !exists {
			entry = &lockoutEntry{
				ipAttempts: make(map[string]*ipLockoutEntry),
			}
			al.attempts[rec.IdentifierHash] = entry
		}

		if rec.IP == "" {
			// Global entry
			entry.failedAttempts = rec.FailedAttempts
			entry.lockedUntil = rec.LockedUntil
			entry.lastAttempt = rec.LastAttempt
		} else {
			// Per-IP entry
			entry.ipAttempts[rec.IP] = &ipLockoutEntry{
				failedAttempts: rec.FailedAttempts,
				lockedUntil:    rec.LockedUntil,
				lastAttempt:    rec.LastAttempt,
			}
		}
	}

	if al.logger != nil && len(records) > 0 {
		al.logger.Info("restored lockout state from DB", slog.Int("records", len(records)))
	}
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
		lockoutDuration := safeLockoutDuration(al.baseLockout, excessAttempts, al.maxLockout)

		entry.lockedUntil = now.Add(lockoutDuration)

		if al.logger != nil {
			al.logger.Warn("account_locked",
				slog.String("identifier_hash", hashIdentifier(identifier)),
				slog.String("event", "global_account_lockout"),
				slog.Int("failed_attempts", entry.failedAttempts),
				slog.Duration("lockout_duration", lockoutDuration))
		}

		al.persistEntry(identifier, "", entry.failedAttempts, entry.lockedUntil, entry.lastAttempt)
		return true, lockoutDuration
	}

	// Check per-IP lockout
	if ipEntry.failedAttempts >= ipLockoutThreshold {
		excessAttempts := ipEntry.failedAttempts - ipLockoutThreshold
		lockoutDuration := safeLockoutDuration(al.baseLockout, excessAttempts, al.maxLockout)

		ipEntry.lockedUntil = now.Add(lockoutDuration)

		if al.logger != nil {
			al.logger.Warn("account_locked",
				slog.String("identifier_hash", hashIdentifier(identifier)),
				slog.String("event", "ip_account_lockout"),
				slog.Int("ip_failed_attempts", ipEntry.failedAttempts),
				slog.Int("global_failed_attempts", entry.failedAttempts),
				slog.Duration("lockout_duration", lockoutDuration))
		}

		al.persistEntry(identifier, ip, ipEntry.failedAttempts, ipEntry.lockedUntil, ipEntry.lastAttempt)
		return true, lockoutDuration
	}

	return false, 0
}

// persistEntry writes a lockout entry to the database (best-effort, non-blocking).
func (al *AccountLockout) persistEntry(identifier, ip string, failedAttempts int, lockedUntil, lastAttempt time.Time) {
	if al.database == nil {
		return
	}
	err := al.database.UpsertLockout(service.LockoutRecord{
		IdentifierHash: hashIdentifier(identifier),
		IP:             ip,
		FailedAttempts: failedAttempts,
		LockedUntil:    lockedUntil,
		LastAttempt:    lastAttempt,
	})
	if err != nil && al.logger != nil {
		al.logger.Warn("failed to persist lockout to DB", slog.String("error", err.Error()))
	}
}

// RecordSuccess clears the failure counter for an account after successful login.
func (al *AccountLockout) RecordSuccess(identifier string) {
	al.mu.Lock()
	defer al.mu.Unlock()

	delete(al.attempts, identifier)

	// Persist: remove from DB
	if al.database != nil {
		if err := al.database.DeleteLockout(hashIdentifier(identifier)); err != nil && al.logger != nil {
			al.logger.Warn("failed to delete lockout from DB", slog.String("error", err.Error()))
		}
	}
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

// safeLockoutDuration calculates exponential backoff duration without integer overflow.
// Caps the shift exponent at maxExponentShift before computing baseLockout * 2^excessAttempts,
// then clamps the result to maxLockout.
func safeLockoutDuration(baseLockout time.Duration, excessAttempts int, maxLockout time.Duration) time.Duration {
	if excessAttempts > maxExponentShift {
		excessAttempts = maxExponentShift
	}
	lockoutDuration := baseLockout * (1 << excessAttempts)
	if lockoutDuration > maxLockout {
		lockoutDuration = maxLockout
	}
	return lockoutDuration
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
