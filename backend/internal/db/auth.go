package db

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
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
		return nil, fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get user insert id: %w", err)
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
		return nil, fmt.Errorf("query user by id: %w", err)
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
		return nil, fmt.Errorf("query user by username or email: %w", err)
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
	familyID, err := generateTokenFamilyID()
	if err != nil {
		return fmt.Errorf("generate token family id: %w", err)
	}
	_, err = db.Exec(`
		INSERT INTO refresh_tokens (user_id, token, expires_at, family_id)
		VALUES (?, ?, ?, ?)
	`, userID, tokenHash, expiresAt, familyID)
	if err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}
	return nil
}

// ValidateRefreshToken checks if a refresh token exists and is not expired
// Returns the associated user_id if valid
func (db *DB) ValidateRefreshToken(token string) (int, error) {
	var userID int
	var expiresAt string
	var consumedAt, revokedAt sql.NullString
	tokenHash := hashRefreshToken(token)

	err := db.QueryRow(`
		SELECT user_id, expires_at, consumed_at, revoked_at FROM refresh_tokens
		WHERE token = ?
	`, tokenHash).Scan(&userID, &expiresAt, &consumedAt, &revokedAt)

	if err == sql.ErrNoRows {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("query refresh token: %w", err)
	}

	// Check expiration
	expiresTime, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, fmt.Errorf("parse refresh token expiry: %w", err)
	}

	if consumedAt.Valid || revokedAt.Valid {
		return 0, ErrRefreshTokenReuse
	}

	if time.Now().After(expiresTime) {
		_, _ = db.Exec(`UPDATE refresh_tokens SET revoked_at = datetime('now') WHERE token = ?`, tokenHash)
		return 0, errors.New("refresh token expired")
	}

	return userID, nil
}

// RotateRefreshToken deletes the old token and creates a new one atomically
// This is a security best practice to prevent token theft
func (db *DB) RotateRefreshToken(oldToken string, userID int, newToken string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin token rotation tx: %w", err)
	}
	defer tx.Rollback()

	oldTokenHash := hashRefreshToken(oldToken)
	newTokenHash := hashRefreshToken(newToken)

	var dbUserID int
	var familyID, expiresAt string
	var consumedAt, revokedAt sql.NullString
	err = tx.QueryRow(`
		SELECT user_id, family_id, expires_at, consumed_at, revoked_at
		FROM refresh_tokens
		WHERE token = ?
	`, oldTokenHash).Scan(&dbUserID, &familyID, &expiresAt, &consumedAt, &revokedAt)
	if err == sql.ErrNoRows {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("query old refresh token: %w", err)
	}

	if dbUserID != userID {
		return ErrNotFound
	}
	if familyID == "" {
		familyID, err = generateTokenFamilyID()
		if err != nil {
			return fmt.Errorf("generate token family id: %w", err)
		}
		if _, err := tx.Exec(`UPDATE refresh_tokens SET family_id = ? WHERE token = ?`, familyID, oldTokenHash); err != nil {
			return fmt.Errorf("set token family id: %w", err)
		}
	}

	if consumedAt.Valid || revokedAt.Valid {
		if _, err := tx.Exec(`UPDATE refresh_tokens SET revoked_at = datetime('now') WHERE family_id = ? AND revoked_at IS NULL`, familyID); err != nil {
			return fmt.Errorf("revoke token family: %w", err)
		}
		return ErrRefreshTokenReuse
	}

	expiry, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return fmt.Errorf("parse token expiry: %w", err)
	}
	if time.Now().After(expiry) {
		if _, err := tx.Exec(`UPDATE refresh_tokens SET revoked_at = datetime('now') WHERE token = ?`, oldTokenHash); err != nil {
			return fmt.Errorf("revoke expired token: %w", err)
		}
		return errors.New("refresh token expired")
	}

	result, err := tx.Exec(`
		UPDATE refresh_tokens
		SET consumed_at = datetime('now'), replaced_by = ?
		WHERE token = ? AND consumed_at IS NULL AND revoked_at IS NULL
	`, newTokenHash, oldTokenHash)
	if err != nil {
		return fmt.Errorf("consume old token: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check token consumption rows: %w", err)
	}
	if rows == 0 {
		if _, err := tx.Exec(`UPDATE refresh_tokens SET revoked_at = datetime('now') WHERE family_id = ? AND revoked_at IS NULL`, familyID); err != nil {
			return fmt.Errorf("revoke token family after failed rotation: %w", err)
		}
		return ErrRefreshTokenReuse
	}

	newExpiresAt := time.Now().Add(auth.RefreshTokenDuration).Format(time.RFC3339)
	_, err = tx.Exec(`
		INSERT INTO refresh_tokens (user_id, token, expires_at, family_id)
		VALUES (?, ?, ?, ?)
	`, userID, newTokenHash, newExpiresAt, familyID)
	if err != nil {
		return fmt.Errorf("insert rotated token: %w", err)
	}

	return tx.Commit()
}

// DeleteRefreshToken removes a refresh token (used for logout)
func (db *DB) DeleteRefreshToken(token string) error {
	tokenHash := hashRefreshToken(token)
	if _, err := db.Exec(`DELETE FROM refresh_tokens WHERE token = ?`, tokenHash); err != nil {
		return fmt.Errorf("delete refresh token: %w", err)
	}
	return nil
}

// CleanupExpiredRefreshTokens removes expired and old revoked tokens from the database.
// Returns the number of deleted rows.
func (db *DB) CleanupExpiredRefreshTokens() (int64, error) {
	now := time.Now().Format(time.RFC3339)
	res, err := db.Exec(`
		DELETE FROM refresh_tokens
		WHERE expires_at < ?
		   OR (revoked_at IS NOT NULL AND revoked_at < datetime('now', '-7 days'))
	`, now)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired tokens: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get cleanup rows affected: %w", err)
	}
	return n, nil
}

func hashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func generateTokenFamilyID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate random bytes: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// RevokeRefreshTokenFamilyByToken revokes all refresh tokens in the family of token.
func (db *DB) RevokeRefreshTokenFamilyByToken(token string) error {
	tokenHash := hashRefreshToken(token)
	if _, err := db.Exec(`
		UPDATE refresh_tokens
		SET revoked_at = datetime('now')
		WHERE family_id = (
			SELECT family_id FROM refresh_tokens WHERE token = ?
		)
		AND revoked_at IS NULL
	`, tokenHash); err != nil {
		return fmt.Errorf("revoke token family: %w", err)
	}
	return nil
}

// DeleteAllUserRefreshTokensExcept deletes all refresh tokens for a user except the current one
// This is used when changing email or password to invalidate other sessions
// The currentRawToken is hashed internally (consistent with other functions)
func (db *DB) DeleteAllUserRefreshTokensExcept(userID int, currentRawToken string) error {
	hashedToken := hashRefreshToken(currentRawToken)
	if _, err := db.Exec(`DELETE FROM refresh_tokens WHERE user_id = ? AND token != ?`, userID, hashedToken); err != nil {
		return fmt.Errorf("delete other refresh tokens: %w", err)
	}
	return nil
}

// DeleteAllUserRefreshTokens deletes all refresh tokens for a user (e.g., after password reset)
func (db *DB) DeleteAllUserRefreshTokens(userID int) error {
	if _, err := db.Exec(`DELETE FROM refresh_tokens WHERE user_id = ?`, userID); err != nil {
		return fmt.Errorf("delete all user refresh tokens: %w", err)
	}
	return nil
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
		return fmt.Errorf("update user email: %w", err)
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
		return fmt.Errorf("update user password: %w", err)
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
		return fmt.Errorf("update user password in tx: %w", err)
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
	if err != nil {
		return nil, fmt.Errorf("query user by email: %w", err)
	}
	return &user, nil
}

// SetUserEncryptionSalt stores or updates a user's encryption salt
// The salt is used for Argon2id key derivation on the client-side
func (db *DB) SetUserEncryptionSalt(userID int, salt []byte) error {
	if _, err := db.Exec(`
		UPDATE users
		SET encryption_salt = ?
		WHERE id = ?
	`, salt, userID); err != nil {
		return fmt.Errorf("set encryption salt: %w", err)
	}
	return nil
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
		return nil, fmt.Errorf("query encryption salt: %w", err)
	}

	// Check if salt is NULL (not yet generated)
	if salt == nil {
		return nil, ErrNotFound
	}

	return salt, nil
}
