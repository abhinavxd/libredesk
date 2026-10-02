import { describe, it, expect } from 'vitest'
import {
  emptyFilters,
  filtersFromQuery,
  queryFromFilters,
  hasActiveFilters,
  toFiltersJSON,
  UNASSIGNED
} from './searchFilters'

describe('searchFilters', () => {
  it('round-trips filters through the route query', () => {
    const filters = {
      ...emptyFilters(),
      status: '2',
      tags: ['3', '7'],
      created: '2026-01-01,2026-01-31'
    }
    expect(filtersFromQuery(queryFromFilters(filters))).toEqual(filters)
  })

  it('drops empty values and junk tag ids from the query', () => {
    expect(queryFromFilters(emptyFilters())).toEqual({})
    expect(filtersFromQuery({ tags: 'a,0,-1,4,', status: '' }).tags).toEqual(['4'])
  })

  it('reports whether any filter is active', () => {
    expect(hasActiveFilters(emptyFilters())).toBe(false)
    expect(hasActiveFilters({ ...emptyFilters(), tags: ['1'] })).toBe(true)
    expect(hasActiveFilters({ ...emptyFilters(), inbox: '5' })).toBe(true)
  })

  it('serializes to the conversation list filter JSON', () => {
    const json = toFiltersJSON({
      ...emptyFilters(),
      status: '1',
      assignee: UNASSIGNED,
      team: '9',
      tags: ['2', '5'],
      created: '2026-01-01,2026-01-31'
    })
    expect(JSON.parse(json)).toEqual([
      { model: 'conversations', field: 'status_id', operator: 'equals', value: '1' },
      { model: 'conversations', field: 'assigned_user_id', operator: 'not set', value: '' },
      { model: 'conversations', field: 'assigned_team_id', operator: 'equals', value: '9' },
      { model: 'conversations', field: 'tags', operator: 'contains', value: '[2,5]' },
      {
        model: 'conversations',
        field: 'created_at',
        operator: 'between',
        value: '2026-01-01,2026-01-31'
      }
    ])
  })

  it('returns an empty string when nothing is set', () => {
    expect(toFiltersJSON(emptyFilters())).toBe('')
  })
})
