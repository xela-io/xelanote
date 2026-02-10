import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  preprocess: vitePreprocess({ style: false }),

  kit: {
    adapter: adapter({
      pages: 'build',
      assets: 'build',
      fallback: 'index.html',
      precompress: false,
      strict: true,
    }),
    // Use relative paths for Electron/Tauri desktop builds (file:// protocol)
    paths: {
      base: '',
      relative: true,
    },
    alias: {
      $lib: './src/lib',
      $components: './src/lib/components',
    },
  },
};

export default config;
