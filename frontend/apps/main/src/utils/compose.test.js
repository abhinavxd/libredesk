// @vitest-environment jsdom
import { describe, expect, it } from 'vitest'
import { composeProblems, defaultFromAddress, joinRecipients, splitRecipients } from './compose'

const addresses = [
  { id: 1, address: 'support@example.test', enabled: true },
  { id: 2, address: 'old@example.test', enabled: false },
  { id: 3, address: 'billing@example.test', enabled: true }
]

describe('compose helpers', () => {
  it('splits and joins recipient lists', () => {
    expect(splitRecipients(' a@example.test, b@example.test;\nc@example.test,, ')).toEqual([
      'a@example.test',
      'b@example.test',
      'c@example.test'
    ])
    expect(joinRecipients(['a@example.test', 'b@example.test'])).toBe('a@example.test, b@example.test')
    expect(joinRecipients(null)).toBe('')
  })

  it('starts from the viewed address when it can send, else the first usable one', () => {
    expect(defaultFromAddress(addresses, '3')).toBe(3)
    expect(defaultFromAddress(addresses, 2)).toBe(1)
    expect(defaultFromAddress(addresses, undefined)).toBe(1)
    expect(defaultFromAddress([{ id: 2, enabled: false }], 2)).toBeNull()
  })

  it('reports every reason the email cannot be sent', () => {
    expect(composeProblems({ addressID: 1, to: 'a@example.test', cc: '', bcc: '', subject: 'Hi', hasBody: true })).toEqual([])
    const problems = composeProblems({ addressID: null, to: '', cc: 'nope', bcc: '', subject: ' ', hasBody: false })
    expect(problems.map((problem) => problem.key)).toEqual([
      'compose.fromRequired',
      'compose.toRequired',
      'compose.invalidRecipients',
      'compose.subjectRequired',
      'compose.bodyRequired'
    ])
    expect(problems[2].values).toEqual({ field: 'CC', list: 'nope' })
  })
})
