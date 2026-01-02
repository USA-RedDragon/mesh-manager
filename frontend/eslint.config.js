import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import pluginVitest from '@vitest/eslint-plugin'
import pluginCypress from 'eslint-plugin-cypress/flat'
import tseslint from 'typescript-eslint'
import { defineConfigWithVueTs, vueTsConfigs } from '@vue/eslint-config-typescript'

export default defineConfigWithVueTs([
  {
    name: 'app/files-to-lint',
    files: ['**/*.{js,mjs,vue,ts}'],
  },

  {
    name: 'app/files-to-ignore',
    ignores: ['**/dist/**', '**/dist-ssr/**', '**/coverage/**', '**/reports/**'],
  },

  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...pluginVue.configs['flat/essential'],
  vueTsConfigs.recommended,

  {
    ...pluginVitest.configs.recommended,
    files: ['src/**/__tests__/*'],
  },

  {
    ...pluginCypress.configs.recommended,
    files: ['**/__tests__/*.{cy,spec}.{js,ts,jsx,tsx}', 'tests/e2e/**/*.{cy,spec}.{js,ts,jsx,tsx}'],
  },

  {
    rules: {
      'object-curly-spacing': ['error', 'always'],
      'require-jsdoc': 'off',
      indent: ['error', 2, { SwitchCase: 1 }],
      'max-len': ['error', 120],
      '@typescript-eslint/no-unused-vars': [
        'error',
        { varsIgnorePattern: '^_', argsIgnorePattern: '^_' },
      ],
      'vue/multi-word-component-names': 'off',
      'valid-jsdoc': 'off',
    },
  },
])
