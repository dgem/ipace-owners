import eslint from '@eslint/js';
import globals from 'globals';

export default [
  {
    ignores: ['_site/**', 'node_modules/**', 'firebase-preview.json'],
  },
  eslint.configs.recommended,
  {
    rules: {
      'no-useless-assignment': 'off',
    },
  },
  {
    files: ['src/assets/js/**/*.js'],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'script',
      globals: globals.browser,
    },
  },
  {
    files: ['.eleventy.js', 'scripts/**/*.{js,cjs,mjs}', 'test/**/*.js'],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: globals.node,
    },
  },
  {
    files: ['.eleventy.js', 'scripts/**/*.cjs'],
    languageOptions: {
      sourceType: 'commonjs',
    },
  },
];
