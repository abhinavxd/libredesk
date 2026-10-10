import { describe, expect, test } from 'vitest'
import { matchLanguage } from './language.js'

const codes = ['de-DE', 'en-US', 'pt-BR', 'zh-CN']

describe('matchLanguage', () => {
  test('returns an exact code, ignoring case and underscores', () => {
    expect(matchLanguage('zh-CN', codes)).toBe('zh-CN')
    expect(matchLanguage('pt_br', codes)).toBe('pt-BR')
    expect(matchLanguage(' EN-us ', codes)).toBe('en-US')
  })

  test('falls back to a code with the same primary subtag', () => {
    expect(matchLanguage('zh', codes)).toBe('zh-CN')
    expect(matchLanguage('de-AT', codes)).toBe('de-DE')
    expect(matchLanguage('en-GB', codes)).toBe('en-US')
  })

  test('returns an empty string when nothing matches or nothing was asked for', () => {
    expect(matchLanguage('ko-KR', codes)).toBe('')
    expect(matchLanguage('', codes)).toBe('')
    expect(matchLanguage(null, codes)).toBe('')
    expect(matchLanguage('zh-CN', undefined)).toBe('')
  })
})
