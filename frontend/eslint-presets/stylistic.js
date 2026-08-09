/**
 * @import { Linter } from 'eslint'
 */
import stylistic from '@stylistic/eslint-plugin';
import prettierConfig from 'eslint-config-prettier';
import prettierPlugin from 'eslint-plugin-prettier';

/**
 * Стилистические правила
 * @type {Linter.Config}
 */
const preset = [
  prettierConfig,
  {
    name: 'my-presets/stylistic',
    plugins: {
      '@stylistic': stylistic,
      prettier: prettierPlugin,
    },

    rules: {
      'prettier/prettier': 'error',

      '@stylistic/max-statements-per-line': ['error', { max: 2 }],

      '@stylistic/lines-between-class-members': [
        'error',
        'always',
        { exceptAfterSingleLine: true },
      ],

      '@stylistic/newline-per-chained-call': [
        'error',
        { ignoreChainWithDepth: 3 },
      ],

      '@stylistic/object-property-newline': [
        'warn',
        { allowAllPropertiesOnSameLine: true },
      ],

      '@stylistic/padding-line-between-statements': [
        'error',
        {
          blankLine: 'always',
          prev: '*',
          next: [
            'block',
            'block-like',
            'return',
            'class',
            'export',
            'for',
            'while',
            'if',
          ],
        },
        {
          blankLine: 'always',
          prev: ['block', 'block-like', 'const', 'let', 'var'],
          next: '*',
        },
        {
          blankLine: 'any',
          prev: ['const', 'let', 'var'],
          next: ['const', 'let', 'var'],
        },
      ],

      '@stylistic/line-comment-position': ['warn', { position: 'above' }],

      '@stylistic/lines-around-comment': [
        'warn',
        {
          beforeBlockComment: true,
          allowBlockStart: true,
          allowArrayStart: true,
          allowObjectStart: true,
          allowClassStart: true,
          allowEnumStart: true,
          allowInterfaceStart: true,
          allowTypeStart: true,
          allowModuleStart: true,
        },
      ],
    },
  },
];

export default preset;
