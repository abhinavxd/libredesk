// @vitest-environment jsdom
import { describe, it, expect } from 'vitest'
import { isValidTemplate } from '@shared-ui/utils/string.js'

const VARS = ['.Agent.FirstName', '.Agent.LastName', '.Agent.FullName', '.Inbox.Name']

describe('isValidTemplate', () => {
  describe('valid', () => {
    it('empty string', () => {
      expect(isValidTemplate('', VARS)).toBe(true)
    })

    it('plain text with no variables', () => {
      expect(isValidTemplate('Support team', VARS)).toBe(true)
    })

    it('single supported variable', () => {
      expect(isValidTemplate('{{ .Agent.FirstName }}', VARS)).toBe(true)
    })

    it('multiple variables mixed with text', () => {
      expect(isValidTemplate('{{ .Agent.FirstName }} at {{ .Inbox.Name }}', VARS)).toBe(true)
    })

    it('variable without surrounding spaces', () => {
      expect(isValidTemplate('{{.Agent.FullName}}', VARS)).toBe(true)
    })
  })

  describe('invalid', () => {
    it('unknown variable', () => {
      expect(isValidTemplate('{{ .Agent.Email }}', VARS)).toBe(false)
    })

    it('typo in variable name', () => {
      expect(isValidTemplate('{{ .Agent.Naem }}', VARS)).toBe(false)
    })

    it('missing closing brace', () => {
      expect(isValidTemplate('{{ .Agent.FirstName }', VARS)).toBe(false)
    })

    it('extra closing brace', () => {
      expect(isValidTemplate('{{ .Agent.FirstName }}}', VARS)).toBe(false)
    })

    it('stray opening braces after a valid token', () => {
      expect(isValidTemplate('{{ .Inbox.Name }} {{', VARS)).toBe(false)
    })
  })
})
