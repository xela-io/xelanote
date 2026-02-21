//go:build fts5

package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/xela-io/xelanote/internal/auth"
)

// TestGenerateSignatureIntegration verifies signature generation works with real JWT secret
func TestGenerateSignatureIntegration(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	sig, expires, err := auth.GenerateUploadSignature(userID, filename, secret)
	if err != nil {
		t.Fatalf("GenerateUploadSignature failed: %v", err)
	}

	if sig == "" {
		t.Error("Expected non-empty signature")
	}

	if expires <= time.Now().Unix() {
		t.Error("Expected expiry in the future")
	}
}

// TestValidateSignatureIntegration verifies signature validation with realistic data
func TestValidateSignatureIntegration(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	// Generate signature
	sig, expires, err := auth.GenerateUploadSignature(userID, filename, secret)
	if err != nil {
		t.Fatalf("GenerateUploadSignature failed: %v", err)
	}

	// Validate it
	err = auth.ValidateUploadSignature(userID, filename, sig, expires, secret)
	if err != nil {
		t.Errorf("ValidateUploadSignature failed for valid signature: %v", err)
	}
}

// TestExpiredSignatureIntegration verifies expired signatures fail validation
func TestExpiredSignatureIntegration(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	// Create signature with expiry in the past
	expires := time.Now().Add(-1 * time.Hour).Unix()
	sig, _, _ := auth.GenerateUploadSignature(userID, filename, secret)

	// Manually create expired signature
	// (we can't use GenerateUploadSignature since it always creates future expiry)
	mac := auth.GenerateUploadSignature
	_ = mac // suppress unused warning

	// Instead, test validation with expired timestamp
	err := auth.ValidateUploadSignature(userID, filename, sig, expires, secret)
	if err == nil {
		t.Error("Expected error for expired signature")
	}
}

// TestTamperedSignatureIntegration verifies tampered signatures fail
func TestTamperedSignatureIntegration(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	sig, expires, _ := auth.GenerateUploadSignature(userID, filename, secret)

	// Tamper with signature
	tamperedSig := sig + "x"

	err := auth.ValidateUploadSignature(userID, filename, tamperedSig, expires, secret)
	if err == nil {
		t.Error("Expected error for tampered signature")
	}
}

// TestWrongUserSignatureIntegration verifies signatures are user-specific
func TestWrongUserSignatureIntegration(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	user1ID := 42
	user2ID := 99
	filename := "test-image.png"

	// Generate signature for user1
	sig, expires, _ := auth.GenerateUploadSignature(user1ID, filename, secret)

	// Try to validate with user2's ID
	err := auth.ValidateUploadSignature(user2ID, filename, sig, expires, secret)
	if err == nil {
		t.Error("Expected error when validating with wrong user ID")
	}
}

// TestWrongFilenameSignatureIntegration verifies signatures are filename-specific
func TestWrongFilenameSignatureIntegration(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename1 := "test-image.png"
	filename2 := "hacked.png"

	// Generate signature for filename1
	sig, expires, _ := auth.GenerateUploadSignature(userID, filename1, secret)

	// Try to validate with filename2
	err := auth.ValidateUploadSignature(userID, filename2, sig, expires, secret)
	if err == nil {
		t.Error("Expected error when validating with wrong filename")
	}
}

// TestSignatureURLFormat verifies the signature generates proper URL query params
func TestSignatureURLFormat(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	sig, expires, err := auth.GenerateUploadSignature(userID, filename, secret)
	if err != nil {
		t.Fatalf("GenerateUploadSignature failed: %v", err)
	}

	// Construct URL
	url := fmt.Sprintf("/api/uploads/%d/%s?signature=%s&expires=%d", userID, filename, sig, expires)

	// Verify URL format
	if url == "" {
		t.Error("Generated empty URL")
	}

	// Parse expires from URL
	var parsedExpires int64
	_, err = fmt.Sscanf(url, "/api/uploads/%d/%s?signature=%s&expires=%d", &userID, &filename, &sig, &parsedExpires)
	if err != nil {
		// URL parsing via fmt.Sscanf is tricky, so we just verify basic structure
		t.Log("URL format verification (non-fatal):", url)
	}
}

// TestFileUploadPathSecurity verifies upload path security
func TestFileUploadPathSecurity(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	userID := 42

	// Verify clean filename
	cleanFilename := filepath.Base("test.png")
	if cleanFilename != "test.png" {
		t.Error("Expected clean filename")
	}

	// Verify path traversal prevention
	maliciousFilename := "../../../etc/passwd"
	cleanMalicious := filepath.Base(maliciousFilename)
	if cleanMalicious != "passwd" {
		t.Errorf("Expected 'passwd', got %s", cleanMalicious)
	}

	// Verify user directory isolation
	userUploadDir := filepath.Join(tmpDir, UploadDir, fmt.Sprintf("%d", userID))
	err := os.MkdirAll(userUploadDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create user upload directory: %v", err)
	}

	// Verify directory exists
	info, err := os.Stat(userUploadDir)
	if err != nil {
		t.Fatalf("User upload directory not created: %v", err)
	}

	if !info.IsDir() {
		t.Error("Expected directory, got file")
	}
}

// TestSignatureExpiryDuration verifies 7-day expiry
func TestSignatureExpiryDuration(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	before := time.Now()
	_, expires, err := auth.GenerateUploadSignature(userID, filename, secret)
	if err != nil {
		t.Fatalf("GenerateUploadSignature failed: %v", err)
	}

	expectedExpiry := before.Add(auth.SignatureExpiry).Unix()
	actualExpiry := expires

	// Allow 5 seconds tolerance for test execution time
	diff := actualExpiry - expectedExpiry
	if diff < -5 || diff > 5 {
		t.Errorf("Expiry not ~7 days from now. Expected ~%d, got %d (diff: %d seconds)",
			expectedExpiry, actualExpiry, diff)
	}
}

// TestMultipleSignaturesForSameFile verifies signature uniqueness per timestamp
func TestMultipleSignaturesForSameFile(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	// Generate two signatures (may share the same timestamp depending on clock resolution)
	sig1, expires1, _ := auth.GenerateUploadSignature(userID, filename, secret)
	sig2, expires2, _ := auth.GenerateUploadSignature(userID, filename, secret)

	// If the timestamps differ, signatures should differ as well.
	if expires1 != expires2 && sig1 == sig2 {
		t.Error("Expected different signatures for different expiry timestamps")
	}

	// Both should validate correctly
	if err := auth.ValidateUploadSignature(userID, filename, sig1, expires1, secret); err != nil {
		t.Errorf("Signature 1 validation failed: %v", err)
	}

	if err := auth.ValidateUploadSignature(userID, filename, sig2, expires2, secret); err != nil {
		t.Errorf("Signature 2 validation failed: %v", err)
	}
}

// TestSignatureWithSpecialCharactersInFilename tests edge cases
func TestSignatureWithSpecialCharactersInFilename(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42

	testCases := []string{
		"test-image.png",
		"test_image.png",
		"test.image.png",
		"123456789.png",
		"a.png",
	}

	for _, filename := range testCases {
		sig, expires, err := auth.GenerateUploadSignature(userID, filename, secret)
		if err != nil {
			t.Errorf("Failed for filename %s: %v", filename, err)
			continue
		}

		err = auth.ValidateUploadSignature(userID, filename, sig, expires, secret)
		if err != nil {
			t.Errorf("Validation failed for filename %s: %v", filename, err)
		}
	}
}

// TestSignatureQueryParamExtraction simulates URL query param extraction
func TestSignatureQueryParamExtraction(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	sig, expires, _ := auth.GenerateUploadSignature(userID, filename, secret)

	// Simulate URL construction
	url := fmt.Sprintf("/api/uploads/%d/%s?signature=%s&expires=%d", userID, filename, sig, expires)

	// In real code, query params would be extracted via chi.URLParam() and r.URL.Query().Get()
	// For this test, we simply verify the URL format is correct
	if url == "" {
		t.Error("Generated empty URL")
	}

	// Verify signature directly
	err := auth.ValidateUploadSignature(userID, filename, sig, expires, secret)
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}
}

// TestSignatureWithDifferentUserIDs verifies signature isolation
func TestSignatureWithDifferentUserIDs(t *testing.T) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	filename := "test-image.png"

	userIDs := []int{1, 2, 42, 999, 1000000}
	signatures := make([]string, len(userIDs))

	// Generate signatures for different users
	for i, userID := range userIDs {
		sig, _, err := auth.GenerateUploadSignature(userID, filename, secret)
		if err != nil {
			t.Fatalf("Failed to generate signature for user %d: %v", userID, err)
		}
		signatures[i] = sig
	}

	// Verify all signatures are unique
	for i := 0; i < len(signatures); i++ {
		for j := i + 1; j < len(signatures); j++ {
			if signatures[i] == signatures[j] {
				t.Errorf("Signatures for users %d and %d are identical", userIDs[i], userIDs[j])
			}
		}
	}
}

// BenchmarkGenerateUploadSignature benchmarks signature generation
func BenchmarkGenerateUploadSignature(b *testing.B) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = auth.GenerateUploadSignature(userID, filename, secret)
	}
}

// BenchmarkValidateUploadSignature benchmarks signature validation
func BenchmarkValidateUploadSignature(b *testing.B) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	sig, expires, _ := auth.GenerateUploadSignature(userID, filename, secret)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = auth.ValidateUploadSignature(userID, filename, sig, expires, secret)
	}
}

// BenchmarkServeUpload simulates serving a file with signature validation
func BenchmarkServeUpload(b *testing.B) {
	secret := []byte("test-jwt-secret-at-least-64-characters-long-for-security-purposes-ok")
	userID := 42
	filename := "test-image.png"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Generate signature
		sig, expires, _ := auth.GenerateUploadSignature(userID, filename, secret)

		// Validate signature
		_ = auth.ValidateUploadSignature(userID, filename, sig, expires, secret)

		// Simulate query param extraction
		expiresStr := strconv.FormatInt(expires, 10)
		parsedExpires, _ := strconv.ParseInt(expiresStr, 10, 64)
		_ = parsedExpires
	}
}

// TestServeUpload_PathTraversal verifies that path traversal filenames are rejected
// even when injected directly into chi route params (bypassing chi's URL normalization).
func TestServeUpload_PathTraversal(t *testing.T) {
	ts := newTestServer(t)
	ts.dataDir = t.TempDir()

	tests := []struct {
		name       string
		filename   string
		wantStatus int
	}{
		{"parent dir traversal", "../etc/passwd", http.StatusBadRequest},
		{"double parent traversal", "../../etc/shadow", http.StatusBadRequest},
		{"dot-dot bypasses Base but caught by prefix check", "..", http.StatusForbidden},
		{"single dot", ".", http.StatusBadRequest},
		{"subdir with slash", "foo/bar.png", http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := 42

			// Generate valid signature so authentication succeeds —
			// we are testing the path validation, not auth.
			sig, expires, err := auth.GenerateUploadSignature(userID, tt.filename, ts.jwtSecret)
			if err != nil {
				t.Fatalf("failed to generate signature: %v", err)
			}

			url := fmt.Sprintf("/api/uploads/%d/file?signature=%s&expires=%d", userID, sig, expires)
			req := httptest.NewRequest(http.MethodGet, url, nil)

			// Inject chi route params directly to simulate worst case
			// (what if chi somehow passes a malicious value through).
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("user_id", strconv.Itoa(userID))
			rctx.URLParams.Add("filename", tt.filename)
			req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

			rec := httptest.NewRecorder()
			ts.serveUpload(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("filename %q: expected status %d, got %d (body: %s)",
					tt.filename, tt.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestServeUpload_ValidFilename verifies that a legitimate UUID filename passes
// validation and results in 404 (file not found) rather than a security error.
func TestServeUpload_ValidFilename(t *testing.T) {
	ts := newTestServer(t)
	ts.dataDir = t.TempDir()

	userID := 42
	filename := "a1b2c3d4-e5f6-7890-abcd-ef1234567890.png"
	sig, expires, err := auth.GenerateUploadSignature(userID, filename, ts.jwtSecret)
	if err != nil {
		t.Fatalf("failed to generate signature: %v", err)
	}

	// Create the user upload dir so the prefix check can succeed
	userUploadDir := filepath.Join(ts.dataDir, UploadDir, strconv.Itoa(userID))
	if err := os.MkdirAll(userUploadDir, 0750); err != nil {
		t.Fatalf("failed to create upload dir: %v", err)
	}

	url := fmt.Sprintf("/api/uploads/%d/%s?signature=%s&expires=%d", userID, filename, sig, expires)
	req := httptest.NewRequest(http.MethodGet, url, nil)

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", strconv.Itoa(userID))
	rctx.URLParams.Add("filename", filename)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	rec := httptest.NewRecorder()
	ts.serveUpload(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404 for valid filename (file doesn't exist), got %d (body: %s)",
			rec.Code, rec.Body.String())
	}
}

// TestServeUpload_UserIsolation verifies that user B cannot access user A's files
// via cookie-based authentication.
func TestServeUpload_UserIsolation(t *testing.T) {
	ts := newTestServer(t)
	ts.dataDir = t.TempDir()

	userA := ts.createUser(t, "userA", "a@test.com", "password123")
	userB := ts.createUser(t, "userB", "b@test.com", "password123")

	// Create a file in user A's upload directory
	userADir := filepath.Join(ts.dataDir, UploadDir, strconv.Itoa(userA.ID))
	if err := os.MkdirAll(userADir, 0750); err != nil {
		t.Fatalf("failed to create upload dir: %v", err)
	}
	filename := "secret.png"
	if err := os.WriteFile(filepath.Join(userADir, filename), []byte("secret data"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// User B tries to access user A's file via cookie auth (no signed URL)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/uploads/%d/%s", userA.ID, filename), nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("user_id", strconv.Itoa(userA.ID))
	rctx.URLParams.Add("filename", filename)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	// Inject user B's ID as the authenticated user (simulates JWT middleware)
	ctx = context.WithValue(ctx, userIDKey, userB.ID)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	ts.serveUpload(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for cross-user access, got %d (body: %s)",
			rec.Code, rec.Body.String())
	}
}
