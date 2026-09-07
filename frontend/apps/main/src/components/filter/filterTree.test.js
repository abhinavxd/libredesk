import { describe, it, expect } from 'vitest'
import {
  createLeaf,
  createGroup,
  createRoot,
  isGroupNode,
  isStrictTwoLevel,
  collectLeaves,
  isPartialLeaf,
  normalizeToTwoLevel,
  serializeFilterTree,
  deserializeFilterTree
} from '@main/components/filter/filterTree'
import { FIELD_TYPE, LOGIC, OPERATOR } from '@main/constants/filterConfig'

const FIELDS = [
  { field: 'status_id', type: FIELD_TYPE.SELECT, model: 'conversations' },
  { field: 'priority_id', type: FIELD_TYPE.SELECT, model: 'conversations' },
  { field: 'tags', type: FIELD_TYPE.MULTI_SELECT, model: 'conversations' },
  { field: 'email', type: FIELD_TYPE.TEXT, model: 'users' }
]

const leaf = (over = {}) => ({
  model: 'conversations',
  field: 'status_id',
  operator: OPERATOR.EQUALS,
  value: '1',
  ...over
})

const stripIds = (node) => {
  if (isGroupNode(node)) {
    const { __id, ...rest } = node
    return { ...rest, rules: (node.rules || []).map(stripIds) }
  }
  const { __id, ...rest } = node
  return rest
}

describe('filterTree constructors', () => {
  it('builds a root holding one group holding one blank leaf', () => {
    const root = createRoot()

    expect(root.logic).toBe(LOGIC.AND)
    expect(root.rules).toHaveLength(1)
    expect(root.rules[0].rules).toHaveLength(1)
    expect(isStrictTwoLevel(root)).toBe(true)
    expect(collectLeaves(root)).toHaveLength(1)
  })

  it('gives every node a unique key', () => {
    const ids = [createRoot(), createGroup(), createLeaf(), createLeaf()].map((n) => n.__id)

    expect(new Set(ids).size).toBe(ids.length)
  })

  it('honours the requested logic', () => {
    expect(createRoot(LOGIC.OR).logic).toBe(LOGIC.OR)
    expect(createGroup(LOGIC.OR).logic).toBe(LOGIC.OR)
  })
})

describe('isStrictTwoLevel', () => {
  it('accepts a root of groups of leaves', () => {
    expect(
      isStrictTwoLevel({ logic: LOGIC.AND, rules: [{ logic: LOGIC.OR, rules: [leaf(), leaf()] }] })
    ).toBe(true)
  })

  it('rejects a three-level tree', () => {
    expect(
      isStrictTwoLevel({
        logic: LOGIC.AND,
        rules: [{ logic: LOGIC.AND, rules: [{ logic: LOGIC.OR, rules: [leaf()] }] }]
      })
    ).toBe(false)
  })

  it('rejects loose leaves at the top level', () => {
    expect(isStrictTwoLevel({ logic: LOGIC.AND, rules: [leaf()] })).toBe(false)
  })

  it('rejects an array, an empty root, null and undefined', () => {
    expect(isStrictTwoLevel([leaf()])).toBe(false)
    expect(isStrictTwoLevel({ logic: LOGIC.AND, rules: [] })).toBe(false)
    expect(isStrictTwoLevel(null)).toBe(false)
    expect(isStrictTwoLevel(undefined)).toBe(false)
  })
})

describe('normalizeToTwoLevel', () => {
  it('wraps a legacy flat array into one AND group', () => {
    const out = normalizeToTwoLevel([leaf(), leaf({ field: 'priority_id', value: '2' })])

    expect(isStrictTwoLevel(out)).toBe(true)
    expect(stripIds(out)).toEqual({
      logic: LOGIC.AND,
      rules: [
        {
          logic: LOGIC.AND,
          rules: [leaf(), leaf({ field: 'priority_id', value: '2' })]
        }
      ]
    })
  })

  it('leaves an already strict two-level tree semantically unchanged', () => {
    const input = {
      logic: LOGIC.OR,
      rules: [
        { logic: LOGIC.AND, rules: [leaf()] },
        { logic: LOGIC.OR, rules: [leaf({ field: 'priority_id', value: '2' })] }
      ]
    }

    expect(stripIds(normalizeToTwoLevel(input))).toEqual(input)
  })

  it('flattens a group nested inside a group instead of dropping its leaves', () => {
    const deep = {
      logic: LOGIC.AND,
      rules: [
        {
          logic: LOGIC.OR,
          rules: [leaf(), { logic: LOGIC.OR, rules: [leaf({ field: 'priority_id', value: '2' })] }]
        }
      ]
    }

    const out = normalizeToTwoLevel(deep)

    expect(isStrictTwoLevel(out)).toBe(true)
    expect(collectLeaves(out).map((l) => l.field)).toEqual(['status_id', 'priority_id'])
  })

  it('keeps loose top-level leaves as their own leading group, ahead of real groups', () => {
    const mixed = {
      logic: LOGIC.AND,
      rules: [
        leaf(),
        { logic: LOGIC.OR, rules: [leaf({ field: 'priority_id', value: '2' }), leaf({ field: 'priority_id', value: '3' })] }
      ]
    }

    const out = normalizeToTwoLevel(mixed)

    expect(isStrictTwoLevel(out)).toBe(true)
    expect(stripIds(out)).toEqual({
      logic: LOGIC.AND,
      rules: [
        { logic: LOGIC.AND, rules: [leaf()] },
        {
          logic: LOGIC.OR,
          rules: [leaf({ field: 'priority_id', value: '2' }), leaf({ field: 'priority_id', value: '3' })]
        }
      ]
    })
  })

  it('carries the outer logic onto the group it invents for loose leaves', () => {
    const out = normalizeToTwoLevel({ logic: LOGIC.OR, rules: [leaf(), leaf({ field: 'priority_id' })] })

    expect(out.logic).toBe(LOGIC.OR)
    expect(out.rules[0].logic).toBe(LOGIC.OR)
  })

  it('returns a usable blank tree for an empty root and junk input', () => {
    for (const input of [{ logic: LOGIC.AND, rules: [] }, null, undefined, 'nonsense', 42]) {
      const out = normalizeToTwoLevel(input)
      expect(isStrictTwoLevel(out), JSON.stringify(input)).toBe(true)
      expect(collectLeaves(out).length).toBeGreaterThan(0)
    }
  })

  // Every other empty input gets a blank leaf, this one does not.
  it('yields a group with no conditions for an empty legacy array', () => {
    const out = normalizeToTwoLevel([])

    expect(isStrictTwoLevel(out)).toBe(true)
    expect(out.rules).toHaveLength(1)
    expect(collectLeaves(out)).toEqual([])
  })

  it('defaults a missing logic to AND at both levels', () => {
    const out = normalizeToTwoLevel({ rules: [{ rules: [leaf()] }] })

    expect(out.logic).toBe(LOGIC.AND)
    expect(out.rules[0].logic).toBe(LOGIC.AND)
  })

  it('does not mutate the input', () => {
    const input = { logic: LOGIC.AND, rules: [{ logic: LOGIC.OR, rules: [leaf()] }] }
    const before = JSON.stringify(input)

    normalizeToTwoLevel(input)

    expect(JSON.stringify(input)).toBe(before)
  })
})

describe('collectLeaves', () => {
  it('returns leaves in order across groups', () => {
    const tree = {
      logic: LOGIC.AND,
      rules: [
        { logic: LOGIC.AND, rules: [leaf({ field: 'a' }), leaf({ field: 'b' })] },
        { logic: LOGIC.OR, rules: [leaf({ field: 'c' })] }
      ]
    }

    expect(collectLeaves(tree).map((l) => l.field)).toEqual(['a', 'b', 'c'])
  })

  it('returns an empty list for a group with no rules', () => {
    expect(collectLeaves({ logic: LOGIC.AND, rules: [] })).toEqual([])
  })
})

describe('isPartialLeaf', () => {
  it('accepts a fully filled leaf', () => {
    expect(isPartialLeaf(leaf())).toBe(false)
  })

  it('rejects a leaf with no field or no operator', () => {
    expect(isPartialLeaf(leaf({ field: '' }))).toBe(true)
    expect(isPartialLeaf(leaf({ operator: '' }))).toBe(true)
  })

  it('rejects a leaf whose value is empty, null, undefined or an empty array', () => {
    expect(isPartialLeaf(leaf({ value: '' }))).toBe(true)
    expect(isPartialLeaf(leaf({ value: null }))).toBe(true)
    expect(isPartialLeaf(leaf({ value: undefined }))).toBe(true)
    expect(isPartialLeaf(leaf({ value: [] }))).toBe(true)
  })

  it('accepts set and not-set with no value, since those operators take none', () => {
    expect(isPartialLeaf(leaf({ operator: OPERATOR.SET, value: '' }))).toBe(false)
    expect(isPartialLeaf(leaf({ operator: OPERATOR.NOT_SET, value: undefined }))).toBe(false)
  })

  it('accepts a zero and a false value', () => {
    expect(isPartialLeaf(leaf({ value: 0 }))).toBe(false)
    expect(isPartialLeaf(leaf({ value: false }))).toBe(false)
  })
})

describe('serializeFilterTree', () => {
  it('drops __id from every node and keeps the group logic', () => {
    const out = serializeFilterTree(
      deserializeFilterTree(normalizeToTwoLevel([leaf()]), FIELDS)
    )

    expect(JSON.stringify(out)).not.toContain('__id')
    expect(out).toEqual({
      logic: LOGIC.AND,
      rules: [{ logic: LOGIC.AND, rules: [leaf()] }]
    })
  })

  it('turns a multi-select array into a JSON string of numbers', () => {
    const out = serializeFilterTree({
      logic: LOGIC.AND,
      rules: [{ logic: LOGIC.AND, rules: [leaf({ field: 'tags', value: ['3', '7'] })] }]
    })

    expect(out.rules[0].rules[0].value).toBe('[3,7]')
  })

  it('keeps non-numeric multi-select values as strings', () => {
    const out = serializeFilterTree({
      logic: LOGIC.AND,
      rules: [{ logic: LOGIC.AND, rules: [leaf({ field: 'tags', value: ['billing', '7'] })] }]
    })

    expect(out.rules[0].rules[0].value).toBe('["billing",7]')
  })

  it('serializes an empty multi-select as an empty JSON array', () => {
    const out = serializeFilterTree(leaf({ field: 'tags', value: [] }))

    expect(out.value).toBe('[]')
  })

  it('leaves a scalar value untouched', () => {
    expect(serializeFilterTree(leaf({ value: '1' })).value).toBe('1')
    expect(serializeFilterTree(leaf({ value: 0 })).value).toBe(0)
  })

  it('emits only the four wire fields for a leaf', () => {
    const out = serializeFilterTree({ ...leaf(), __id: 'f9', label: 'Status', options: [1, 2] })

    expect(Object.keys(out).sort()).toEqual(['field', 'model', 'operator', 'value'])
  })

  it('defaults a missing group logic to AND', () => {
    expect(serializeFilterTree({ rules: [leaf()] }).logic).toBe(LOGIC.AND)
  })
})

describe('deserializeFilterTree', () => {
  it('restores a multi-select JSON string to an array of string ids', () => {
    const out = deserializeFilterTree(
      { logic: LOGIC.AND, rules: [{ logic: LOGIC.AND, rules: [leaf({ field: 'tags', value: '[3,7]' })] }] },
      FIELDS
    )

    expect(out.rules[0].rules[0].value).toEqual(['3', '7'])
  })

  it('keeps an already-array multi-select value as string ids', () => {
    const out = deserializeFilterTree(leaf({ field: 'tags', value: [3, 7] }), FIELDS)

    expect(out.value).toEqual(['3', '7'])
  })

  it('falls back to an empty array on an unparseable multi-select value', () => {
    expect(deserializeFilterTree(leaf({ field: 'tags', value: 'not json' }), FIELDS).value).toEqual([])
    expect(deserializeFilterTree(leaf({ field: 'tags', value: '{"a":1}' }), FIELDS).value).toEqual([])
    expect(deserializeFilterTree(leaf({ field: 'tags', value: null }), FIELDS).value).toEqual([])
  })

  it('leaves a non-multi-select value alone', () => {
    expect(deserializeFilterTree(leaf({ value: '1' }), FIELDS).value).toBe('1')
    expect(deserializeFilterTree(leaf({ field: 'email', value: 'a@b.com' }), FIELDS).value).toBe('a@b.com')
  })

  it('leaves the value alone when the field is not in the field list', () => {
    expect(deserializeFilterTree(leaf({ field: 'unknown', value: '[3,7]' }), FIELDS).value).toBe('[3,7]')
  })

  it('gives every node a key so list rendering stays stable', () => {
    const out = deserializeFilterTree(
      { logic: LOGIC.AND, rules: [{ logic: LOGIC.AND, rules: [leaf(), leaf()] }] },
      FIELDS
    )

    expect(out.__id).toBeTruthy()
    expect(out.rules[0].__id).toBeTruthy()
    expect(out.rules[0].rules.map((r) => r.__id).filter(Boolean)).toHaveLength(2)
  })

  it('preserves an existing key rather than reassigning it', () => {
    const out = deserializeFilterTree({ __id: 'keep-me', logic: LOGIC.AND, rules: [] }, FIELDS)

    expect(out.__id).toBe('keep-me')
  })
})

describe('round trip through the wire format', () => {
  const load = (stored) => deserializeFilterTree(normalizeToTwoLevel(stored), FIELDS)

  it('survives save, reload and save again unchanged', () => {
    const edited = {
      logic: LOGIC.OR,
      rules: [
        {
          logic: LOGIC.AND,
          rules: [leaf({ value: '1' }), leaf({ field: 'tags', value: ['3', '7'] })]
        },
        {
          logic: LOGIC.OR,
          rules: [leaf({ field: 'priority_id', value: '2' }), leaf({ field: 'email', operator: OPERATOR.SET, value: '' })]
        }
      ]
    }

    const saved = serializeFilterTree(edited)
    const savedAgain = serializeFilterTree(load(saved))

    expect(savedAgain).toEqual(saved)
  })

  it('keeps every group logic and leaf order across the round trip', () => {
    const saved = serializeFilterTree({
      logic: LOGIC.OR,
      rules: [
        { logic: LOGIC.OR, rules: [leaf({ field: 'a' }), leaf({ field: 'b' })] },
        { logic: LOGIC.AND, rules: [leaf({ field: 'c' })] }
      ]
    })

    const reloaded = serializeFilterTree(load(saved))

    expect(reloaded.logic).toBe(LOGIC.OR)
    expect(reloaded.rules.map((g) => g.logic)).toEqual([LOGIC.OR, LOGIC.AND])
    expect(reloaded.rules.map((g) => g.rules.map((r) => r.field))).toEqual([['a', 'b'], ['c']])
  })

  it('upgrades a legacy flat array and saves it as a two-level tree', () => {
    const saved = serializeFilterTree(load([leaf(), leaf({ field: 'priority_id', value: '2' })]))

    expect(saved).toEqual({
      logic: LOGIC.AND,
      rules: [{ logic: LOGIC.AND, rules: [leaf(), leaf({ field: 'priority_id', value: '2' })] }]
    })
  })

  it('does not double-encode a multi-select value on a second save', () => {
    const first = serializeFilterTree(load([leaf({ field: 'tags', value: ['3', '7'] })]))
    const second = serializeFilterTree(load(first))

    expect(first.rules[0].rules[0].value).toBe('[3,7]')
    expect(second.rules[0].rules[0].value).toBe('[3,7]')
  })
})
