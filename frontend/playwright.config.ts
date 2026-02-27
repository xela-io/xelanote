import { defineConfig, devices } from '@playwright/test';

const CI = !!process.env.CI;

export default defineConfig({
  testDir: './tests',
  outputDir: './tests/results',
  timeout: 60000,
  expect: {
    timeout: 10000,
    toHaveScreenshot: {
      maxDiffPixelRatio: 0.03,
      threshold: 0.2,
      animations: 'disabled',
    },
  },
  fullyParallel: true,
  forbidOnly: CI,
  retries: CI ? 2 : 0,
  workers: CI ? 2 : undefined,
  reporter: [
    ['list'],
    ['html', { outputFolder: 'tests/reports/html', open: 'never' }],
    ['json', { outputFile: 'tests/reports/results.json' }],
  ],
  use: {
    baseURL: 'http://localhost:4173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 30000,
    navigationTimeout: 60000,
  },
  projects: [
    // --- Default: E2E tests on Chromium (existing behavior) ---
    {
      name: 'e2e',
      testDir: './tests/e2e',
      use: { ...devices['Desktop Chrome'] },
    },

    // --- Visual regression: Desktop browsers ---
    {
      name: 'visual-desktop-chromium',
      testDir: './tests/visual',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1440, height: 900 },
      },
    },
    {
      name: 'visual-desktop-firefox',
      testDir: './tests/visual',
      use: {
        ...devices['Desktop Firefox'],
        viewport: { width: 1440, height: 900 },
      },
    },
    {
      name: 'visual-desktop-webkit',
      testDir: './tests/visual',
      use: {
        ...devices['Desktop Safari'],
        viewport: { width: 1440, height: 900 },
      },
    },

    // --- Visual regression: Tablet ---
    {
      name: 'visual-tablet-portrait',
      testDir: './tests/visual',
      use: {
        ...devices['iPad (gen 7)'],
      },
    },
    {
      name: 'visual-tablet-landscape',
      testDir: './tests/visual',
      use: {
        ...devices['iPad (gen 7) landscape'],
      },
    },

    // --- Visual regression: Mobile ---
    {
      name: 'visual-mobile-ios',
      testDir: './tests/visual',
      use: {
        ...devices['iPhone 14'],
      },
    },
    {
      name: 'visual-mobile-android',
      testDir: './tests/visual',
      use: {
        ...devices['Pixel 7'],
      },
    },

    // --- Functional tests ---
    {
      name: 'functional',
      testDir: './tests/functional',
      use: { ...devices['Desktop Chrome'] },
    },

    // --- Accessibility tests ---
    {
      name: 'accessibility',
      testDir: './tests/accessibility',
      use: { ...devices['Desktop Chrome'] },
    },

    // --- Design validation ---
    {
      name: 'design',
      testDir: './tests/design',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: [
    {
      command: CI
        ? 'JWT_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef XELANOTE_DB=:memory: XELANOTE_ENV=test /tmp/xelanote-test'
        : 'cd ../backend && JWT_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef XELANOTE_DB=:memory: XELANOTE_ENV=test go run -tags "fts5" ./cmd/server',
      port: 8080,
      reuseExistingServer: !CI,
      timeout: 120000,
      stdout: 'pipe',
      stderr: 'pipe',
    },
    {
      command: 'VITE_API_BASE_URL=/api npm run dev -- --host localhost --port 4173',
      port: 4173,
      reuseExistingServer: !CI,
      stdout: 'pipe',
      stderr: 'pipe',
    },
  ],
});
