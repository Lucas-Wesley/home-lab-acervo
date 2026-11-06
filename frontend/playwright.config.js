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
            BIBLIOTECA_URL: 'http://localhost:8093',
            BIBLIOTECA_LINK: 'https://biblioteca.lan',
            FLASHCARDS_URL: 'http://localhost:8097',
            FLASHCARDS_LINK: 'https://flashcards.lan',
            ACERVO_LINK: 'https://acervo.lan',
          },
          url: 'http://localhost:8091/api/health',
        },
        {
          command: 'node e2e/biblioteca-falsa.mjs',
          env: { PORT: '8093' },
          url: 'http://localhost:8093/api/books',
        },
        {
          command: 'node e2e/flashcards-falso.mjs',
          env: { PORT: '8097' },
          url: 'http://localhost:8097/api/baralhos',
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
