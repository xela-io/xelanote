package api

import "os"

// xelanoteEnv returns the current XELANOTE_ENV value.
func xelanoteEnv() string {
	return os.Getenv("XELANOTE_ENV")
}

// isDevelopment reports whether the app is running in development or test mode.
// These environments run without TLS, so Secure cookies must be disabled.
func isDevelopment() bool {
	env := xelanoteEnv()
	return env == "development" || env == "test" || env == "testing"
}

// isTestEnv reports whether the app is running in a test environment.
// Test environments use relaxed rate limits for E2E tests.
func isTestEnv() bool {
	env := xelanoteEnv()
	return env == "test" || env == "testing"
}
