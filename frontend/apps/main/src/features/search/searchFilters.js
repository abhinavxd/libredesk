export const FILTER_KEYS = ['status', 'priority', 'inbox', 'assignee', 'team', 'tags', 'created']

export const UNASSIGNED = 'none'

export const emptyFilters = () => ({
  status: '',
  priority: '',
  inbox: '',
  assignee: '',
  team: '',
  tags: [],
  created: ''
})

export const filtersFromQuery = (query) => {
  const filters = emptyFilters()
  for (const key of FILTER_KEYS) {
    const raw = query[key]
    if (raw === undefined || raw === null || raw === '') continue
    if (key === 'tags') {
      filters.tags = String(raw)
        .split(',')
        .filter((v) => /^[1-9]\d*$/.test(v))
    } else {
      filters[key] = String(raw)
    }
  }
  return filters
}

export const queryFromFilters = (filters) => {
  const query = {}
  for (const key of FILTER_KEYS) {
    const value = filters[key]
    if (key === 'tags') {
      if (value?.length) query.tags = value.join(',')
    } else if (value) {
      query[key] = value
    }
  }
  return query
}

export const hasActiveFilters = (filters) =>
  FILTER_KEYS.some((key) => (key === 'tags' ? filters.tags?.length > 0 : Boolean(filters[key])))

const leaf = (field, operator, value = '') => ({
  model: 'conversations',
  field,
  operator,
  value: String(value)
})

const assignmentLeaf = (field, value) =>
  value === UNASSIGNED ? leaf(field, 'not set') : leaf(field, 'equals', value)

export const toFiltersJSON = (filters) => {
  const rules = []
  if (filters.status) rules.push(leaf('status_id', 'equals', filters.status))
  if (filters.priority) rules.push(leaf('priority_id', 'equals', filters.priority))
  if (filters.inbox) rules.push(leaf('inbox_id', 'equals', filters.inbox))
  if (filters.assignee) rules.push(assignmentLeaf('assigned_user_id', filters.assignee))
  if (filters.team) rules.push(assignmentLeaf('assigned_team_id', filters.team))
  if (filters.tags?.length) rules.push(leaf('tags', 'contains', JSON.stringify(filters.tags.map(Number))))
  if (filters.created) rules.push(leaf('created_at', 'between', filters.created))
  return rules.length ? JSON.stringify(rules) : ''
}
