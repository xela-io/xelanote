package api

type rateLimitConfig struct {
	registerLimit       int
	loginLimit          int
	refreshLimit        int
	tfaLimit            int
	backupLimit         int
	recoveryLimit       int
	uploadLimit         int
	importLimit         int
	searchLimit         int
	passwordChangeLimit int
	emailChangeLimit    int
	recoveryKeyLimit    int
	llmLimit            int
	fido2Limit          int
	shareLimit          int
	userSearchLimit     int
	errorReportLimit    int
	perfMetricsLimit    int
	analyticsLimit      int
	lockoutAttempts     int
}

func buildRateLimitConfig() rateLimitConfig {
	// Defaults for production and development.
	config := rateLimitConfig{
		registerLimit:       5,
		loginLimit:          10,
		refreshLimit:        30,
		tfaLimit:            5,
		backupLimit:         3,
		recoveryLimit:       3,
		uploadLimit:         20,
		importLimit:         10,
		searchLimit:         120,
		passwordChangeLimit: 3,
		emailChangeLimit:    3,
		recoveryKeyLimit:    3,
		llmLimit:            10,
		fido2Limit:          10,
		shareLimit:          20,
		userSearchLimit:     30,
		errorReportLimit:    5,
		perfMetricsLimit:    30,
		analyticsLimit:      20,
		lockoutAttempts:     10,
	}

	// Use higher rate limits in test environment to allow E2E tests to run.
	if isTestEnv() {
		config.registerLimit = 1000
		config.loginLimit = 1000
		config.refreshLimit = 1000
		config.tfaLimit = 1000
		config.backupLimit = 1000
		config.recoveryLimit = 1000
		config.uploadLimit = 1000
		config.importLimit = 1000
		config.passwordChangeLimit = 1000
		config.emailChangeLimit = 1000
		config.recoveryKeyLimit = 1000
		config.fido2Limit = 1000
		config.searchLimit = 10000
		config.llmLimit = 10000
		config.shareLimit = 10000
		config.userSearchLimit = 10000
		config.errorReportLimit = 10000
		config.perfMetricsLimit = 10000
		config.analyticsLimit = 10000
		config.lockoutAttempts = 1000
	}

	return config
}
