import { describe, it, expect } from 'vitest'
import { splitHighlight } from './highlight'

describe('splitHighlight', () => {
  it('marks every case-insensitive match', () => {
    expect(splitHighlight('Refund the refund', 'REFUND')).toEqual([
      { text: 'Refund', match: true },
      { text: ' the ', match: false },
      { text: 'refund', match: true }
    ])
  })

  it('returns the whole text unmarked when the term is blank or absent', () => {
    expect(splitHighlight('hello', '  ')).toEqual([{ text: 'hello', match: false }])
    expect(splitHighlight('hello', 'xyz')).toEqual([{ text: 'hello', match: false }])
    expect(splitHighlight('', 'a')).toEqual([])
  })
})
