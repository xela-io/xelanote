import { sveltekit } from '@sveltejs/kit/vite';
import { visualizer } from 'rollup-plugin-visualizer';
import type { PluginOption } from 'vite';
import { VitePWA } from 'vite-plugin-pwa';
import topLevelAwait from 'vite-plugin-top-level-await';
import wasm from 'vite-plugin-wasm';
import { defineConfig } from 'vitest/config';

const plugins: PluginOption[] = [wasm(), topLevelAwait(), sveltekit()];

// ✅ PWA Plugin in allen Umgebungen aktivieren (ES Worker kompatibel)
// Security-First: Prompted updates + cache purging on logout
plugins.push(
  VitePWA({
    strategies: 'generateSW',
    registerType: 'prompt', // ← User controls when to update
    injectRegister: false, // Manual registration in +layout.svelte

    manifest: {
      name: 'xelanote - Encrypted Notes',
      short_name: 'xelanote',
      description:
        'Secure personal note-taking with end-to-end encryption, wikilinks, and graph view',
      id: '/',
      theme_color: '#458588', // Gruvbox teal — matches meta theme-color for light mode
      background_color: '#282828',
      display: 'standalone',
      orientation: 'any',
      start_url: '/',
      scope: '/',
      categories: ['productivity', 'utilities'],
      icons: [
        {
          src: '/icon-192.png',
          sizes: '192x192',
          type: 'image/png',
          purpose: 'any',
        },
        {
          src: '/icon-512.png',
          sizes: '512x512',
          type: 'image/png',
          purpose: 'any',
        },
        {
          src: '/icon-192-maskable.png',
          sizes: '192x192',
          type: 'image/png',
          purpose: 'maskable',
        },
        {
          src: '/icon-512-maskable.png',
          sizes: '512x512',
          type: 'image/png',
          purpose: 'maskable',
        },
      ],
      shortcuts: [
        {
          name: 'New Note',
          short_name: 'New',
          url: '/?action=new-note',
          icons: [{ src: '/icon-192.png', sizes: '192x192', type: 'image/png' }],
        },
        {
          name: 'Search',
          short_name: 'Search',
          url: '/search',
          icons: [{ src: '/icon-192.png', sizes: '192x192', type: 'image/png' }],
        },
        {
          name: 'Journal',
          short_name: 'Journal',
          url: '/journal',
          icons: [{ src: '/icon-192.png', sizes: '192x192', type: 'image/png' }],
        },
        {
          name: 'Due Dates',
          short_name: 'Due',
          url: '/due-dates',
          icons: [{ src: '/icon-192.png', sizes: '192x192', type: 'image/png' }],
        },
      ],
      share_target: {
        action: '/',
        method: 'GET',
        params: { title: 'title', text: 'text', url: 'url' },
      },
    },

    workbox: {
      globPatterns: ['**/*.{js,css,html,ico,png,svg,woff,woff2}'],
      globIgnores: ['**/splash/**'], // Splash screens are only fetched by iOS via <link> media queries
      navigateFallback: '/', // ← Enable SPA routing for offline deep links
      navigateFallbackDenylist: [/^\/api/],
      maximumFileSizeToCacheInBytes: 2 * 1024 * 1024, // 2MB - largest chunk is ~931KB

      // NOTE: No runtime caching for /api/notes, /api/notes/:id, /api/folders.
      // These are user-specific endpoints — caching them in a shared SW cache
      // risks stale data leaking across sessions (timeout, crash, account switch).
      // The app's offline queue handles offline writes; reads use in-memory state.
      runtimeCaching: [
        {
          urlPattern: ({ url }) => url.pathname.startsWith('/api/uploads/'),
          handler: 'CacheFirst',
          options: {
            cacheName: 'uploads', // Cleared on logout in auth.svelte.ts
            expiration: {
              maxEntries: 100,
              maxAgeSeconds: 86400 * 30, // 30 days
            },
          },
        },
      ],

      cleanupOutdatedCaches: true,
      skipWaiting: false, // ← Wait for user confirmation
      clientsClaim: false, // ← Don't auto-claim clients
    },

    devOptions: {
      enabled: false, // Disabled in dev to avoid caching issues during development
      type: 'module',
      navigateFallback: '/', // ← Fix: SvelteKit uses '/' not 'index.html'
    },
  })
);

if (process.env.ANALYZE === 'true') {
  plugins.push(
    visualizer({
      filename: 'bundle-stats.html',
      open: true,
      gzipSize: true,
      brotliSize: true,
    }) as PluginOption
  );
}

export default defineConfig({
  plugins,
  resolve: {
    conditions: ['browser', 'module', 'import', 'default'],
  },
  ssr: {
    // No special SSR externals needed
  },
  optimizeDeps: {
    // No special optimizations needed
  },
  server: {
    port: 5173,
    strictPort: false, // Allow fallback ports for web dev
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        ws: true,
      },
    },
  },
  build: {
    target: 'esnext',
    // Chunk-size warning increased (we know CodeMirror is large)
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        format: 'es',
        manualChunks: (id) => {
          // Only apply manual chunks for client build (not SSR)
          if (id.includes('node_modules')) {
            // CodeMirror - ALL packages together (better compression)
            if (
              id.includes('@codemirror/state') ||
              id.includes('@codemirror/view') ||
              id.includes('@codemirror/commands') ||
              id.includes('@codemirror/lang-markdown') ||
              id.includes('@codemirror/language') ||
              id.includes('@codemirror/autocomplete') ||
              id.includes('@lezer/markdown')
            ) {
              return 'codemirror';
            }

            // Force-graph
            if (id.includes('force-graph')) {
              return 'force-graph';
            }

            // Crypto
            if (id.includes('libsodium-wrappers') || id.includes('@noble/hashes')) {
              return 'crypto';
            }

            // Markdown-Rendering
            if (id.includes('markdown-it') && !id.includes('node_modules/@types')) {
              return 'markdown';
            }

            // Icons
            if (id.includes('lucide-svelte')) {
              return 'icons';
            }

            // UI vendors (rest)
            if (id.includes('bits-ui')) {
              return 'vendor';
            }
          }
        },
      },
    },
  },
  // Fix for Web Worker format issue (IIFE not supported with code-splitting)
  worker: {
    format: 'es',
    rollupOptions: {
      output: {
        format: 'es',
      },
    },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    include: ['src/**/*.test.ts'],
  },
});
