import { describe, it, expect } from 'vitest'
import {
  emptyFilters,
  filtersFromQuery,
  queryFromFilters,
  hasActiveFilters,
  toFiltersJSON
} from './searchFilters'

describe('searchFilters', () => {
  it('round-trips filters through the route query', () => {
    const filters = {
      ...emptyFilters(),
      status: '2',
      address: '3',
      created: '2026-01-01,2026-01-31'
    }
    expect(filtersFromQuery(queryFromFilters(filters))).toEqual(filters)
  })

  it('ignores retired tag filters from old URLs', () => {
    expect(queryFromFilters(emptyFilters())).toEqual({})
    expect(filtersFromQuery({ tags: '4', status: '' })).toEqual(emptyFilters())
  })

  it('reports whether any filter is active', () => {
    expect(hasActiveFilters(emptyFilters())).toBe(false)
    expect(hasActiveFilters({ ...emptyFilters(), tags: ['1'] })).toBe(false)
    expect(hasActiveFilters({ ...emptyFilters(), address: '5' })).toBe(true)
  })

  it('serializes address filters and ignores retired assignment filters', () => {
    const json = toFiltersJSON({
      ...emptyFilters(),
      status: '1',
      address: '4',
      assignee: 'none',
      team: '9',
      tags: ['2', '5'],
      created: '2026-01-01,2026-01-31'
    })
    expect(JSON.parse(json)).toEqual([
      { model: 'conversations', field: 'status_id', operator: 'equals', value: '1' },
      { model: 'conversations', field: 'address_id', operator: 'equals', value: '4' },
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
