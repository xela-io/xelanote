import { defineConfig, devices } from '@playwright/test';

const CI = !!process.env.CI;

export default defineConfig({
  testDir: './tests/pwa',
  outputDir: './tests/results',
  timeout: 60000,
  expect: { timeout: 15000 },
  fullyParallel: false, // SW-Tests brauchen sequenzielle Ausführung
  forbidOnly: CI,
  retries: CI ? 2 : 0,
  workers: 1, // Ein Worker — SW-State ist global pro Browser
  reporter: [
    ['list'],
    ['html', { outputFolder: 'tests/reports/pwa-html', open: 'never' }],
    ['json', { outputFile: 'tests/reports/pwa-results.json' }],
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
    {
      name: 'pwa',
      use: {
        ...devices['Desktop Chrome'],
        serviceWorkers: 'allow',
      },
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
      command: CI
        ? 'npx vite preview --port 4173' // Build bereits in CI-Step passiert
        : 'npm run build && npx vite preview --port 4173',
      port: 4173,
      reuseExistingServer: !CI,
      timeout: 180000, // Build kann 30-60s dauern
      stdout: 'pipe',
      stderr: 'pipe',
    },
  ],
});
