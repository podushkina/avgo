/**
 * @import { Linter } from 'eslint'
 */
import { createTypeScriptImportResolver } from 'eslint-import-resolver-typescript';
import importXPlugin, { createNodeResolver } from 'eslint-plugin-import-x';

/**
 * Import правила (с рекомендованным конфигом)
 * @type {Linter.Config[]}
 */
const preset = [
  importXPlugin.flatConfigs.recommended,
  {
    name: 'my-presets/import',
    settings: {
      'import-x/resolver-next': [
        createTypeScriptImportResolver(),
        createNodeResolver(),
      ],
    },
    rules: {
      'import-x/no-named-as-default': 'error',
      'import-x/no-relative-packages': 'error',
      'import-x/no-useless-path-segments': 'error',
      'import-x/no-self-import': 'error',
      'import-x/first': 'error',
      'import-x/newline-after-import': 'error',
      'import-x/order': [
        'error',
        {
          groups: [
            'builtin',
            'external',
            'internal',
            'parent',
            'sibling',
            'index',
          ],
          alphabetize: {
            order: 'asc',
          },
          'newlines-between': 'always',
        },
      ],
    },
  },
];

export default preset;
