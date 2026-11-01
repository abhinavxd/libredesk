import { defineConfig } from '@lingui/cli'
import { createVueExtractor } from '@lingui/extractor-vue'
import babel from '@lingui/cli/api/extractors/babel'
import { formatter } from '@lingui/format-po-gettext'
import { fileURLToPath } from 'node:url'
import { withGettextHelpers } from './scripts/gettext-extractor.mjs'

export default defineConfig({
  rootDir: fileURLToPath(new URL('..', import.meta.url)),
  sourceLocale: 'en-US',
  locales: ['en-US'], // The maintenance script discovers translated locales from PO files.
  catalogs: [{ path: '<rootDir>/locales/{locale}/LC_MESSAGES/frontend', include: ['<rootDir>/frontend/apps', '<rootDir>/frontend/shared-ui'] }],
  extractors: [withGettextHelpers(babel), withGettextHelpers(createVueExtractor())],
  compileNamespace: 'json',
  format: formatter({ lineNumbers: true, customHeaderAttributes: { 'Project-Id-Version': 'LibreDesk' } }),
})
