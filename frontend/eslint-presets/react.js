/**
 * @import { Linter } from 'eslint'
 */
import react from 'eslint-plugin-react';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';

/**
 * React правила (со всеми рекомендованными конфигами)
 * @type {Linter.Config[]}
 */
const preset = [
  react.configs.flat.recommended,
  react.configs.flat['jsx-runtime'],
  reactRefresh.configs.recommended,
  reactHooks.configs.flat['recommended-latest'],
  {
    name: 'my-presets/react',
    settings: {
      react: { version: '19.2' },
    },
    rules: {
      'react/prop-types': 'off',
      'react/boolean-prop-naming': 'error',
      'react/hook-use-state': 'error',
      'react/no-children-prop': 'error',
      'react/no-multi-comp': 'error',
      'react/no-this-in-sfc': 'error',
      'react/no-typos': 'error',
      'react/no-unused-state': 'error',
      'react/jsx-boolean-value': 'error',
      'react/jsx-no-useless-fragment': 'error',
      'react/jsx-pascal-case': 'error',
      'react/prefer-stateless-function': 'error',
      'react/display-name': 'off',
      'react/self-closing-comp': 'error',
      'react-hooks/exhaustive-deps': 'warn',
    },
  },
];

export default preset;
