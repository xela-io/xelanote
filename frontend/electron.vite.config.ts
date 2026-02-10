/**
 * Electron Vite Configuration
 *
 * Configures the build for Electron main process and preload.
 * The renderer uses the SvelteKit build from `build/` directory.
 */

import { defineConfig, externalizeDepsPlugin } from 'electron-vite';
import { resolve } from 'path';

export default defineConfig({
  // Main process configuration
  main: {
    plugins: [externalizeDepsPlugin()],
    build: {
      outDir: 'dist-electron/main',
      rollupOptions: {
        input: {
          index: resolve(__dirname, 'src-electron/main.ts'),
        },
      },
    },
  },

  // Preload script configuration
  preload: {
    plugins: [externalizeDepsPlugin()],
    build: {
      outDir: 'dist-electron/preload',
      rollupOptions: {
        input: {
          index: resolve(__dirname, 'src-electron/preload.ts'),
        },
        output: {
          // Preload scripts MUST be CommonJS with .cjs extension
          // because package.json has "type": "module"
          format: 'cjs',
          entryFileNames: '[name].cjs',
        },
      },
    },
  },

  // Renderer: Omitted completely - we use SvelteKit's dev server on port 5173 instead
  // electron-vite will skip the renderer and only build/watch main + preload
});
