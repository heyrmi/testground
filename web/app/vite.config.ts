import { resolve } from 'node:path'
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// Output filenames are fixed rather than content-hashed. The build output is
// committed so `go install` produces a working binary, and stable names keep
// those diffs readable. The Go server sends no-cache for /assets, so there is
// no stale-bundle hazard to hash around.
export default defineConfig({
  base: '/',
  plugins: [react(), tailwindcss()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'es2022',
    rollupOptions: {
      input: {
        app: resolve(import.meta.dirname, 'index.html'),
        wc: resolve(import.meta.dirname, 'src/wc/main.ts'),
      },
      output: {
        entryFileNames: 'assets/[name].js',
        chunkFileNames: 'assets/[name].js',
        assetFileNames: 'assets/[name][extname]',
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://127.0.0.1:7373',
      '/static': 'http://127.0.0.1:7373',
    },
  },
})
