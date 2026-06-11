import { FIELD_TYPE, LOGIC, OPERATOR } from '@/constants/filterConfig'

let _seq = 0
const uid = () => `f${++_seq}`

export const createLeaf = () => ({ __id: uid(), model: '', field: '', operator: '', value: '' })

export const createGroup = (logic = LOGIC.AND) => ({ __id: uid(), logic, rules: [createLeaf()] })

export const createRoot = (logic = LOGIC.AND) => ({ __id: uid(), logic, rules: [createGroup()] })

export const isGroupNode = (node) =>
  !!node && (Array.isArray(node.rules) || typeof node.logic === 'string')

// Strict two-level shape: an object root holding only groups, each group holding only leaves.
export const isStrictTwoLevel = (node) =>
  !!node &&
  typeof node === 'object' &&
  !Array.isArray(node) &&
  Array.isArray(node.rules) &&
  node.rules.length > 0 &&
  node.rules.every(
    (g) => isGroupNode(g) && Array.isArray(g.rules) && g.rules.every((r) => !isGroupNode(r))
  )

export const collectLeaves = (node) =>
  isGroupNode(node) ? (node.rules || []).flatMap(collectLeaves) : [node]

export const isPartialLeaf = (leaf) =>
  !leaf.field ||
  !leaf.operator ||
  (![OPERATOR.SET, OPERATOR.NOT_SET].includes(leaf.operator) &&
    (leaf.value === undefined ||
      leaf.value === null ||
      leaf.value === '' ||
      (Array.isArray(leaf.value) && leaf.value.length === 0)))

const keyed = (node) => ({ __id: node.__id || uid(), ...node })

const toStringIdArray = (value) => {
  if (Array.isArray(value)) return value.map((v) => String(v))
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      return Array.isArray(parsed) ? parsed.map((v) => String(v)) : []
    } catch {
      return []
    }
  }
  return []
}

const withIds = (node) => {
  if (isGroupNode(node)) {
    return { __id: node.__id || uid(), logic: node.logic || LOGIC.AND, rules: (node.rules || []).map(withIds) }
  }
  return keyed(node)
}

// normalizeToTwoLevel coerces any stored shape (legacy flat array, old nested tree, new two-level)
// into a strict { logic, rules: [ {logic, rules:[leaf,...]}, ... ] }: the top level holds only groups,
// and each group holds only leaves. Existing groups are preserved; loose leaves are collected into one
// group so semantics (e.g. status AND (high OR low)) survive the coercion.
export const normalizeToTwoLevel = (filters) => {
  if (Array.isArray(filters)) {
    return withIds({ logic: LOGIC.AND, rules: [{ logic: LOGIC.AND, rules: filters.map((f) => ({ ...f })) }] })
  }
  if (!isGroupNode(filters)) return createRoot()

  const outerLogic = filters.logic || LOGIC.AND
  const topRules = filters.rules || []
  const groups = []
  const looseLeaves = []
  for (const r of topRules) {
    if (isGroupNode(r)) {
      groups.push({ logic: r.logic || LOGIC.AND, rules: collectLeaves(r).map((l) => ({ ...l })) })
    } else {
      looseLeaves.push({ ...r })
    }
  }
  if (looseLeaves.length) groups.unshift({ logic: outerLogic, rules: looseLeaves })
  if (groups.length === 0) groups.push({ logic: LOGIC.AND, rules: [createLeaf()] })
  return withIds({ logic: outerLogic, rules: groups })
}

// serializeFilterTree drops UI-only __id and converts multi-select array values to JSON strings of numeric IDs.
export const serializeFilterTree = (node) => {
  if (isGroupNode(node)) {
    return { logic: node.logic || LOGIC.AND, rules: (node.rules || []).map(serializeFilterTree) }
  }
  const leaf = { model: node.model, field: node.field, operator: node.operator, value: node.value }
  if (Array.isArray(leaf.value)) {
    leaf.value = JSON.stringify(
      leaf.value.map((v) => {
        const num = Number(v)
        return isNaN(num) ? v : num
      })
    )
  }
  return leaf
}

// deserializeFilterTree restores multi-select string values to string-ID arrays and keeps __id for stable keys.
export const deserializeFilterTree = (node, fields) => {
  if (isGroupNode(node)) {
    return { __id: node.__id || uid(), logic: node.logic || LOGIC.AND, rules: (node.rules || []).map((n) => deserializeFilterTree(n, fields)) }
  }
  const field = fields.find((f) => f.field === node.field)
  if (field?.type === FIELD_TYPE.MULTI_SELECT) {
    return keyed({ ...node, value: toStringIdArray(node.value) })
  }
  return keyed(node)
}
