package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	"github.com/xela-io/xelanote/internal/auth"
)

// User represents a registered user in the system
type User struct {
	ID             int
	Username       string
	Email          string
	PasswordHash   string
	IsAdmin        bool
	CreatedAt      string
	UpdatedAt      string
	EncryptionSalt *string // Base64-encoded encryption salt (optional)
}

// CreateUser inserts a new user into the database
func (db *DB) CreateUser(username, email, passwordHash string) (*User, error) {
	result, err := db.Exec(`
		INSERT INTO users (username, email, password_hash)
		VALUES (?, ?, ?)
	`, username, email, passwordHash)

	if err != nil {
		// Check for unique constraint violation
		if err.Error() == "UNIQUE constraint failed: users.username" ||
			err.Error() == "UNIQUE constraint failed: users.email" {
			return nil, ErrDuplicate
		}
		return nil, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	userID, err := validateLastInsertID(id, "user id")
	if err != nil {
		return nil, err
	}
	return db.GetUserByID(userID)
}

// GetUserByID retrieves a user by their ID
func (db *DB) GetUserByID(id int) (*User, error) {
	var user User
	var encryptionSaltBytes []byte

	err := db.QueryRow(`
		SELECT id, username, email, password_hash, is_admin, created_at, updated_at, encryption_salt
		FROM users
		WHERE id = ?
	`, id).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt, &encryptionSaltBytes)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Convert encryption_salt BLOB to base64 string if present
	if len(encryptionSaltBytes) > 0 {
		encoded := base64.StdEncoding.EncodeToString(encryptionSaltBytes)
		user.EncryptionSalt = &encoded
	}

	return &user, nil
}

// GetUserByUsernameOrEmail finds a user by username or email
// This is used for login where users can provide either
func (db *DB) GetUserByUsernameOrEmail(usernameOrEmail string) (*User, error) {
	var user User
	var encryptionSaltBytes []byte

	err := db.QueryRow(`
		SELECT id, username, email, password_hash, is_admin, created_at, updated_at, encryption_salt
		FROM users
		WHERE username = ? OR email = ?
	`, usernameOrEmail, usernameOrEmail).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt, &encryptionSaltBytes)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Convert encryption_salt BLOB to base64 string if present
	if len(encryptionSaltBytes) > 0 {
		encoded := base64.StdEncoding.EncodeToString(encryptionSaltBytes)
		user.EncryptionSalt = &encoded
	}

	return &user, nil
}

// CreateRefreshToken stores a refresh token in the database with expiration
func (db *DB) CreateRefreshToken(userID int, token string) error {
	expiresAt := time.Now().Add(auth.RefreshTokenDuration).Format(time.RFC3339)
	tokenHash := hashRefreshToken(token)
	_, err := db.Exec(`
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES (?, ?, ?)
	`, userID, tokenHash, expiresAt)
	return err
}

// ValidateRefreshToken checks if a refresh token exists and is not expired
// Returns the associated user_id if valid
func (db *DB) ValidateRefreshToken(token string) (int, error) {
	var userID int
	var expiresAt string
	tokenHash := hashRefreshToken(token)

	err := db.QueryRow(`
		SELECT user_id, expires_at FROM refresh_tokens
		WHERE token = ?
	`, tokenHash).Scan(&userID, &expiresAt)

	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}

	// Check expiration
	expiresTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, err
	}

	if time.Now().After(expiresTime) {
		// Token expired, clean it up
		db.DeleteRefreshToken(token)
		return 0, errors.New("refresh token expired")
	}

	return userID, nil
}

// RotateRefreshToken deletes the old token and creates a new one atomically
// This is a security best practice to prevent token theft
func (db *DB) RotateRefreshToken(oldToken string, userID int, newToken string) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	oldTokenHash := hashRefreshToken(oldToken)
	// Delete old token
	_, err = tx.Exec(`DELETE FROM refresh_tokens WHERE token = ?`, oldTokenHash)
	if err != nil {
		return err
	}

	// Create new token
	expiresAt := time.Now().Add(auth.RefreshTokenDuration).Format(time.RFC3339)
	newTokenHash := hashRefreshToken(newToken)
	_, err = tx.Exec(`
		INSERT INTO refresh_tokens (user_id, token, expires_at)
		VALUES (?, ?, ?)
	`, userID, newTokenHash, expiresAt)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// DeleteRefreshToken removes a refresh token (used for logout)
func (db *DB) DeleteRefreshToken(token string) error {
	tokenHash := hashRefreshToken(token)
	_, err := db.Exec(`DELETE FROM refresh_tokens WHERE token = ?`, tokenHash)
	return err
}

// CleanupExpiredRefreshTokens removes all expired tokens from the database
// This should be called periodically (e.g., daily cron job)
func (db *DB) CleanupExpiredRefreshTokens() error {
	now := time.Now().Format(time.RFC3339)
	_, err := db.Exec(`DELETE FROM refresh_tokens WHERE expires_at < ?`, now)
	return err
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// DeleteAllUserRefreshTokensExcept deletes all refresh tokens for a user except the current one
// This is used when changing email or password to invalidate other sessions
// The currentRawToken is hashed internally (consistent with other functions)
func (db *DB) DeleteAllUserRefreshTokensExcept(userID int, currentRawToken string) error {
	hashedToken := hashRefreshToken(currentRawToken)
	_, err := db.Exec(`DELETE FROM refresh_tokens WHERE user_id = ? AND token != ?`, userID, hashedToken)
	return err
}

// DeleteAllUserRefreshTokens deletes all refresh tokens for a user (e.g., after password reset)
func (db *DB) DeleteAllUserRefreshTokens(userID int) error {
	_, err := db.Exec(`DELETE FROM refresh_tokens WHERE user_id = ?`, userID)
	return err
}

// UpdateUserEmail updates a user's email address
func (db *DB) UpdateUserEmail(userID int, newEmail string) error {
	result, err := db.Exec(`
		UPDATE users
		SET email = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newEmail, userID)
	if err != nil {
		// Check for unique constraint violation
		if err.Error() == "UNIQUE constraint failed: users.email" {
			return ErrDuplicate
		}
		return err
	}

	return ensureRowsAffected(result)
}

// UpdateUserPassword updates a user's password hash
func (db *DB) UpdateUserPassword(userID int, newPasswordHash string) error {
	result, err := db.Exec(`
		UPDATE users
		SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newPasswordHash, userID)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

// UpdateUserPasswordTx updates a user's password hash within a transaction.
// Use this when atomically updating password along with other operations.
func (tx *Tx) UpdateUserPasswordTx(userID int, newPasswordHash string) error {
	result, err := tx.Exec(`
		UPDATE users
		SET password_hash = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, newPasswordHash, userID)
	if err != nil {
		return err
	}

	return ensureRowsAffected(result)
}

// GetUserByEmail retrieves a user by their email address
func (db *DB) GetUserByEmail(email string) (*User, error) {
	var user User
	err := db.QueryRow(`
		SELECT id, username, email, password_hash, is_admin, created_at, updated_at
		FROM users
		WHERE email = ?
	`, email).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.IsAdmin, &user.CreatedAt, &user.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &user, err
}

// SetUserEncryptionSalt stores or updates a user's encryption salt
// The salt is used for Argon2id key derivation on the client-side
func (db *DB) SetUserEncryptionSalt(userID int, salt []byte) error {
	_, err := db.Exec(`
		UPDATE users
		SET encryption_salt = ?
		WHERE id = ?
	`, salt, userID)
	return err
}

// GetUserEncryptionSalt retrieves a user's encryption salt
// Returns ErrNotFound if the user doesn't exist or has no salt
func (db *DB) GetUserEncryptionSalt(userID int) ([]byte, error) {
	var salt []byte
	err := db.QueryRow(`
		SELECT encryption_salt
		FROM users
		WHERE id = ?
	`, userID).Scan(&salt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Check if salt is NULL (not yet generated)
	if salt == nil {
		return nil, ErrNotFound
	}

	return salt, nil
}
