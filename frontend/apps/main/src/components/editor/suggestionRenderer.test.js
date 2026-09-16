import { describe, expect, it, vi } from 'vitest'
import { createSuggestionRenderer } from './suggestionRenderer'

vi.mock('@tiptap/vue-3', () => ({ VueRenderer: vi.fn() }))

describe('createSuggestionRenderer', () => {
  it('ignores key events before the suggestion list starts', () => {
    const renderer = createSuggestionRenderer({})

    expect(renderer.onKeyDown({ event: { key: 'Enter' } })).toBe(false)
  })
})
