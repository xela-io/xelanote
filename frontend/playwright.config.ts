import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 60000,
  webServer: [
    {
      command: process.env.CI
        ? 'JWT_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef XELANOTE_DB=:memory: XELANOTE_ENV=test /tmp/xelanote-test'
        : 'cd ../backend && JWT_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef XELANOTE_DB=:memory: XELANOTE_ENV=test go run -tags "fts5" ./cmd/server',
      port: 8080,
      reuseExistingServer: !process.env.CI,
      timeout: 120000,
      stdout: 'pipe',
      stderr: 'pipe',
    },
    {
      command: 'VITE_API_BASE_URL=/api npm run dev -- --host localhost --port 4173',
      port: 4173,
      reuseExistingServer: !process.env.CI,
      stdout: 'pipe',
      stderr: 'pipe',
    },
  ],
  use: {
    baseURL: 'http://localhost:4173',
  },
});
