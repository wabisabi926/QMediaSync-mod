import pluginVue from 'eslint-plugin-vue'
import vueTsEslintConfig from '@vue/eslint-config-typescript'
import skipFormatting from '@vue/eslint-config-prettier/skip-formatting'

export default [
  {
    name: 'app/files-to-lint',
    files: ['**/*.{mjs,ts,mts,tsx,vue}'],
  },

  {
    name: 'app/files-to-ignore',
    ignores: ['**/dist/**', '**/dist-ssr/**', '**/coverage/**'],
  },

  ...pluginVue.configs['flat/essential'],
  ...vueTsEslintConfig(),
  {
    name: 'app/project-rules',
    files: ['**/*.{mjs,ts,mts,tsx,vue}'],
    rules: {
      'no-alert': 'warn',
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'no-debugger': 'error',
      'no-duplicate-imports': 'error',
      eqeqeq: ['error', 'smart'],

      '@typescript-eslint/consistent-type-imports': [
        'error',
        { prefer: 'type-imports', fixStyle: 'inline-type-imports' },
      ],
      '@typescript-eslint/no-import-type-side-effects': 'error',

      'vue/html-button-has-type': 'warn',
      'vue/no-ref-object-reactivity-loss': 'error',
      'vue/no-setup-props-reactivity-loss': 'error',
      'vue/no-template-shadow': 'warn',
      'vue/no-template-target-blank': 'error',
      'vue/no-unused-refs': 'warn',
      'vue/no-useless-v-bind': 'warn',
      'vue/no-v-html': 'warn',
      'vue/prefer-use-template-ref': 'warn',
      'vue/require-typed-ref': 'warn',
    },
  },
  skipFormatting,
]
