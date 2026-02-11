import { dirname } from 'node:path';
import { fileURLToPath } from 'node:url';

import js from '@eslint/js';
import prettier from 'eslint-config-prettier';
import importSort from 'eslint-plugin-simple-import-sort';
import svelte from 'eslint-plugin-svelte';
import globals from 'globals';
import svelteParser from 'svelte-eslint-parser';
import tseslint from 'typescript-eslint';

const __dirname = dirname(fileURLToPath(import.meta.url));
const sharedGlobals = {
  ...globals.browser,
  ...globals.node,
};

export default [
  {
    ignores: [
      'node_modules/**',
      'build/**',
      'dist/**',
      'dist-electron/**',
      'dev-dist/**',
      '.playwright-browsers/**',
      'src-tauri/target/**',
      '.svelte-kit/**',
      '.vite/**',
      'coverage/**',
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...svelte.configs['flat/recommended'],
  prettier,
  {
    plugins: {
      'simple-import-sort': importSort,
    },
    rules: {
      'simple-import-sort/imports': 'error',
      'simple-import-sort/exports': 'error',
    },
  },

  // Global rule overrides: downgrade noisy rules to warnings
  {
    rules: {
      // Disabled: SPA app, no SSR preloading needed
      'svelte/no-navigation-without-resolve': 'off',
      // Disabled: all {@html} usage is sanitized i18n content
      'svelte/no-at-html-tags': 'off',

      // Svelte 5 rules - enforce
      'svelte/require-each-key': 'error',
      'svelte/prefer-svelte-reactivity': 'error',
      'svelte/prefer-writable-derived': 'error',

      // TypeScript
      '@typescript-eslint/no-explicit-any': 'error',

      // Common patterns
      'prefer-const': ['error', { destructuring: 'all' }],
      'no-useless-escape': 'error',
    },
  },

  // Svelte files
  {
    files: ['**/*.svelte'],
    languageOptions: {
      parser: svelteParser,
      parserOptions: {
        parser: tseslint.parser,
        ecmaVersion: 'latest',
        sourceType: 'module',
        extraFileExtensions: ['.svelte'],
        tsconfigRootDir: __dirname,
      },
      globals: sharedGlobals,
    },
    rules: {
      'no-unused-vars': 'off',
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
      ],
    },
  },

  // TypeScript and JavaScript files
  {
    files: ['**/*.{ts,js}'],
    languageOptions: {
      parser: tseslint.parser,
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module',
        tsconfigRootDir: __dirname,
      },
      globals: sharedGlobals,
    },
    rules: {
      'no-unused-vars': 'off',
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unused-vars': [
        'error',
        { argsIgnorePattern: '^_', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' },
      ],
    },
  },

  // CommonJS config files (prettier.config.cjs etc.)
  {
    files: ['**/*.cjs'],
    languageOptions: {
      globals: {
        ...globals.node,
        module: 'readonly',
        require: 'readonly',
      },
    },
  },
];
