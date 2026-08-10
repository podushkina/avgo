import js from '@eslint/js';
import { defineConfig, globalIgnores } from 'eslint/config';
import globals from 'globals';

import {
  quality,
  stylistic,
  react,
  importX,
  typescript,
} from './eslint-presets/index.js';

export default defineConfig([
  globalIgnores(['.yarn/**', '**/build/**', '.pnp.*', 'public/**']),

  // Основные файлы исходного кода проекта
  {
    files: ['src/**/*.{ts,tsx,js,jsx}'],
    languageOptions: {
      ecmaVersion: 'latest',
      globals: globals.browser,
    },

    extends: [
      js.configs.recommended,
      quality,
      react,
      importX,
      typescript,
      stylistic,
    ],
  },

  // Компоненты, сгенерированные с помощью библиотеки shadcn
  {
    files: ['src/components/ui/**/*.{ts,tsx}'],
    rules: {
      'react-refresh/only-export-components': 'off',
      'react/no-multi-comp': 'off',
    },
  },

  // Конфиг файлы проекта
  {
    files: ['*.config.{js,ts}', 'eslint-presets/**/*.js'],
    languageOptions: {
      ecmaVersion: 'latest',
      globals: globals.node,
    },

    extends: [js.configs.recommended, quality, importX, stylistic],

    rules: {
      'import-x/no-commonjs': 'error',
      'import-x/no-import-module-exports': 'error',
      'no-restricted-globals': ['error', '__dirname', '__filename'],
    },
  },
]);
