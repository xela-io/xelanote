package api

import (
	"log/slog"
	"net/http"

	"golang.org/x/crypto/bcrypt"
)

// TwoFactorSetupRequest is the request body for starting 2FA setup
type TwoFactorSetupRequest struct{}

// TwoFactorSetupResponse contains TOTP setup data
type TwoFactorSetupResponse struct {
	Secret      string   `json:"secret"`
	QRCodeURL   string   `json:"qr_code_url"`
	BackupCodes []string `json:"backup_codes"`
}

// TwoFactorVerifyRequest is the request body for verifying TOTP code
type TwoFactorVerifyRequest struct {
	Code string `json:"code"`
}

// TwoFactorDisableRequest is the request body for disabling 2FA
type TwoFactorDisableRequest struct {
	Password   string `json:"password"`
	TOTPCode   string `json:"totp_code"`
	BackupCode string `json:"backup_code"`
}

// TwoFactorStatusResponse contains 2FA status
type TwoFactorStatusResponse struct {
	Enabled           bool   `json:"enabled"`
	TOTPEnabled       bool   `json:"totp_enabled"`
	FIDO2Enabled      bool   `json:"fido2_enabled"`
	FIDO2KeyCount     int    `json:"fido2_key_count"`
	VerifiedAt        string `json:"verified_at,omitempty"`
	UnusedBackupCodes int    `json:"unused_backup_codes"`
}

// RegenerateBackupCodesRequest is the request body for regenerating backup codes
type RegenerateBackupCodesRequest struct {
	Password string `json:"password"`
}

// RegenerateBackupCodesResponse contains new backup codes
type RegenerateBackupCodesResponse struct {
	BackupCodes []string `json:"backup_codes"`
}

// setupTwoFactor handles POST /api/2fa/setup
func (s *Server) setupTwoFactor(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Get user email for QR code
	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user details")
		return
	}

	// Generate TOTP setup
	setup, err := s.tfaService.GenerateTOTPSetup(userID, user.Email)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, TwoFactorSetupResponse{
		Secret:      setup.Secret,
		QRCodeURL:   setup.QRCodeURL,
		BackupCodes: setup.BackupCodes,
	})
}

// verifyTwoFactor handles POST /api/2fa/verify
func (s *Server) verifyTwoFactor(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req TwoFactorVerifyRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Code == "" {
		respondError(w, http.StatusBadRequest, "Code is required")
		return
	}

	// Verify TOTP code and enable 2FA
	if err := s.tfaService.VerifyAndEnableTOTP(userID, req.Code); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log 2FA activation
	s.logger().Info("2fa_enabled",
		slog.Int("user_id", userID),
		slog.String("event", "2fa_activation"),
		slog.String("method", "totp"),
		slog.String("remote_ip", getClientIPSafe(r)))

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "2FA enabled successfully",
	})
}

// disableTwoFactor handles DELETE /api/2fa
func (s *Server) disableTwoFactor(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req TwoFactorDisableRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" {
		respondError(w, http.StatusBadRequest, "Password is required")
		return
	}

	// Get user to verify password
	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user details")
		return
	}

	// SEC-005: Use generic error message to prevent credential enumeration
	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Check if 2FA is enabled
	status, err := s.tfaService.GetTwoFactorStatus(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// If 2FA is enabled, require TOTP code or backup code
	if status.Enabled {
		if req.TOTPCode == "" && req.BackupCode == "" {
			respondError(w, http.StatusBadRequest, "TOTP code or backup code required")
			return
		}

		// Apply backup code rate limiter if backup code is provided
		if req.BackupCode != "" {
			clientIP := getClientIPSafe(r)
			if !s.backupCodeLimiter.Allow(clientIP) {
				respondError(w, http.StatusTooManyRequests, "Too many backup code attempts, please try again later")
				return
			}
		}

		// SEC-005: Use generic error message for TOTP/backup code verification
		// Verify TOTP code or backup code
		if req.TOTPCode != "" {
			if err := s.tfaService.VerifyTOTP(userID, req.TOTPCode); err != nil {
				respondError(w, http.StatusUnauthorized, "Invalid credentials")
				return
			}
		} else if req.BackupCode != "" {
			if err := s.tfaService.VerifyBackupCode(userID, req.BackupCode); err != nil {
				respondError(w, http.StatusUnauthorized, "Invalid credentials")
				return
			}
		}
	}

	// Disable 2FA
	if err := s.tfaService.DisableTwoFactor(userID); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Log 2FA deactivation
	s.logger().Warn("2fa_disabled",
		slog.Int("user_id", userID),
		slog.String("event", "2fa_deactivation"),
		slog.String("remote_ip", getClientIPSafe(r)))

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "2FA disabled successfully",
	})
}

// getTwoFactorStatus handles GET /api/2fa/status
func (s *Server) getTwoFactorStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	status, err := s.tfaService.GetTwoFactorStatus(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Check FIDO2 status
	fido2Count := 0
	fido2Enabled := false
	if s.fido2Service != nil {
		creds, err := s.fido2Service.ListCredentials(userID)
		if err == nil {
			fido2Count = len(creds)
			fido2Enabled = fido2Count > 0
		}
	}

	respondJSON(w, http.StatusOK, TwoFactorStatusResponse{
		Enabled:           status.Enabled || fido2Enabled,
		TOTPEnabled:       status.Enabled,
		FIDO2Enabled:      fido2Enabled,
		FIDO2KeyCount:     fido2Count,
		VerifiedAt:        status.VerifiedAt,
		UnusedBackupCodes: status.UnusedBackupCodes,
	})
}

// regenerateBackupCodes handles POST /api/2fa/backup-codes/regenerate
// SEC-009: Requires password re-authentication to prevent session-based attacks
func (s *Server) regenerateBackupCodes(w http.ResponseWriter, r *http.Request) {
	userID, ok := getUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// SEC-009: Parse request body to get password
	var req RegenerateBackupCodesRequest
	if err := decodeJSON(w, r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Password == "" {
		respondError(w, http.StatusBadRequest, "Password is required")
		return
	}

	// SEC-009: Verify password before regenerating codes
	user, err := s.authService.GetUserByID(userID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to get user details")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Regenerate backup codes
	codes, err := s.tfaService.RegenerateBackupCodes(userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Log backup code regeneration
	s.logger().Info("backup_codes_regenerated",
		slog.Int("user_id", userID),
		slog.String("event", "backup_codes_regeneration"),
		slog.String("remote_ip", getClientIPSafe(r)))

	respondJSON(w, http.StatusOK, RegenerateBackupCodesResponse{
		BackupCodes: codes,
	})
}
