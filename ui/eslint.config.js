import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import globals from 'globals'

export default [
  js.configs.recommended,
  ...pluginVue.configs['flat/essential'],
  {
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
    rules: {
      // In Vuex and UI callback code, arg signatures document the API (e.g. `(state, payload)`
      // in mutations, `(newVal, oldVal)` in watchers). Linting unused args here produces
      // tons of noise and encourages dropping documentation for no real benefit — so we
      // only flag unused locals, imports, and caught errors.
      'no-unused-vars': ['error', { args: 'none', varsIgnorePattern: '^_', caughtErrorsIgnorePattern: '^_' }],
      // Page-level components (Scenes, List, Filters) are legitimately single-word.
      'vue/multi-word-component-names': 'off',
      // The codebase uses the legacy Vue 2 pattern of mutating props passed from
      // Vuex-backed parents. A proper fix requires moving each button/list to
      // emit-based updates plus reworking parent wiring — tracked separately.
      'vue/no-mutating-props': 'off',
      // Several computed properties dispatch store actions or chain promises
      // to populate themselves. Migrating them to watchers is the right fix but
      // is out of scope for the lint cleanup pass.
      'vue/no-side-effects-in-computed-properties': 'off',
      'vue/no-async-in-computed-properties': 'off',
    },
  },
]
