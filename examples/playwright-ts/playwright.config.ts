import { resolve } from 'node:path'
import { defineConfig } from '@playwright/test'

const PORT = Number(process.env.PLAYGROUND_PORT ?? 7373)
const baseURL = `http://127.0.0.1:${PORT}`
const repoRoot = resolve(import.meta.dirname, '../..')

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0,
  reporter: process.env.CI ? 'github' : 'list',

  use: {
    baseURL,
    // Deterministic pages need no retries to be readable, so keep the trace
    // for whatever did fail.
    trace: 'retain-on-failure',
  },

  // Retries are off on purpose. A playground whose behaviour is fixed by a
  // seed should not need them, and this suite exists partly to prove that.
  webServer: {
    command: `go run ./cmd/playground serve --addr 127.0.0.1:${PORT} --log-level warn`,
    cwd: repoRoot,
    url: `${baseURL}/api/health`,
    reuseExistingServer: !process.env.CI,
    stdout: 'pipe',
    timeout: 120_000,
  },
})
