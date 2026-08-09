/**
 * @import { Linter } from 'eslint'
 */

/**
 * Правила качества кода
 * @type {Linter.Config}
 */
const preset = {
  name: 'my-presets/quality',
  rules: {
    camelcase: 'off',
    'consistent-return': 'warn',
    curly: ['error', 'all'],
    'dot-notation': 'error',
    eqeqeq: ['error', 'always'],
    'max-classes-per-file': 'error',
    'new-cap': ['error', { newIsCap: true }],
    'no-alert': 'error',
    'no-bitwise': 'error',
    'no-console': 'warn',
    'no-else-return': 'error',
    'no-implicit-coercion': 'error',
    'no-lonely-if': 'error',
    'no-multi-assign': 'error',
    'no-negated-condition': 'error',
    'no-param-reassign': 'error',
    'no-redeclare': 'error',
    'no-sequences': 'error',
    'no-shadow': 'off',
    'no-unneeded-ternary': 'error',
    'no-useless-return': 'error',
    'no-var': 'error',
    'no-warning-comments': 'warn',
    'prefer-arrow-callback': 'error',
    'prefer-const': 'error',
    yoda: 'error',
  },
};

export default preset;
