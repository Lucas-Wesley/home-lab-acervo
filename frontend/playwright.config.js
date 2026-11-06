import os from 'node:os'
import path from 'node:path'
import { defineConfig } from '@playwright/test'

const remoto = process.env.E2E_BASE_URL
// O teste local roda com yt-dlp e claude falsos (backend/testdata), em portas e dados só dele.
const dados = path.join(os.tmpdir(), 'acervo-e2e')
const testdata = path.resolve('../backend/testdata')

export default defineConfig({
  testDir: './e2e',
  workers: 1,
  timeout: 60000,
  use: {
    baseURL: remoto || 'http://localhost:5175',
    ignoreHTTPSErrors: true,
  },
  webServer: remoto
    ? undefined
    : [
        {
          command: `rm -rf "${dados}" && go run .`,
          cwd: '../backend',
          env: {
            DATA_DIR: dados,
            ADDR: ':8091',
            YTDLP_BIN: path.join(testdata, 'yt-dlp-falso'),
            CLAUDE_BIN: path.join(testdata, 'claude-falso'),
            REGRAS_DIR: path.resolve('../regras'),
          },
          url: 'http://localhost:8091/api/health',
        },
        {
          // A versão de produção: é a única que tem service worker e, portanto, offline.
          command: 'npm run build && npx vite preview --port 5175 --strictPort',
          env: { API_URL: 'http://localhost:8091' },
          url: 'http://localhost:5175',
          timeout: 180000,
        },
      ],
})
