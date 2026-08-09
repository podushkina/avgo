/**
 * @import { Linter } from 'eslint'
 */
import typescriptEslint from 'typescript-eslint';

/**
 * TypeScript правила (со всеми рекомендованными конфигами)
 * @type {Linter.Config[]}
 */
const preset = [
  typescriptEslint.configs.recommendedTypeChecked,
  typescriptEslint.configs.stylisticTypeChecked,
  {
    name: 'my-presets/typescript',
    languageOptions: {
      parserOptions: {
        projectService: true,
      },
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/interface-name-prefix': 'off',
      '@typescript-eslint/no-unused-vars': 'error',
      '@typescript-eslint/camelcase': 'off',
      '@typescript-eslint/no-var-requires': 'off',
      '@typescript-eslint/ban-ts-comment': 'off',
      '@typescript-eslint/no-empty-function': 'off',
      '@typescript-eslint/no-shadow': 'error',
      '@typescript-eslint/consistent-type-definitions': 'off',
      '@typescript-eslint/consistent-indexed-object-style': 'off',
      '@typescript-eslint/no-floating-promises': [
        'error',
        { ignoreVoid: true },
      ],
      '@typescript-eslint/naming-convention': [
        'error',
        {
          selector: ['parameter'],
          format: ['camelCase', 'PascalCase'],
        },
        {
          selector: ['objectLiteralProperty', 'typeProperty'],
          modifiers: ['requiresQuotes'],
          format: null,
        },
        {
          selector: ['typeProperty', 'objectLiteralProperty'],
          format: ['camelCase', 'snake_case', 'PascalCase', 'UPPER_CASE'],
          leadingUnderscore: 'allow',
        },
        {
          selector: ['memberLike'],
          format: ['camelCase', 'UPPER_CASE'],
        },
        {
          selector: ['memberLike'],
          modifiers: ['private'],
          format: ['camelCase'],
          leadingUnderscore: 'require',
        },
        {
          selector: ['memberLike'],
          modifiers: ['protected'],
          format: ['camelCase'],
          leadingUnderscore: 'require',
        },
        {
          selector: ['classProperty'],
          modifiers: ['readonly'],
          types: ['boolean', 'number', 'string', 'array'],
          format: ['camelCase', 'UPPER_CASE'],
        },
        {
          selector: ['classProperty'],
          modifiers: ['private', 'readonly'],
          types: ['boolean', 'number', 'string', 'array'],
          format: ['camelCase', 'UPPER_CASE'],
          leadingUnderscore: 'require',
        },
        {
          selector: ['classProperty'],
          modifiers: ['protected', 'readonly'],
          types: ['boolean', 'number', 'string', 'array'],
          format: ['camelCase', 'UPPER_CASE'],
          leadingUnderscore: 'require',
        },
        {
          selector: ['variable'],
          format: ['camelCase', 'UPPER_CASE', 'PascalCase'],
        },
        {
          selector: ['typeLike'],
          format: ['PascalCase'],
        },
        {
          selector: ['import'],
          format: ['PascalCase', 'camelCase'],
        },
        {
          selector: 'function',
          format: ['camelCase', 'PascalCase'],
          leadingUnderscore: 'allow',
        },
        {
          selector: ['default'],
          format: ['camelCase'],
        },
      ],
    },
  },
];

export default preset;
