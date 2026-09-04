import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createSSRApp, defineComponent, h } from 'vue'
import { renderToString } from 'vue/server-renderer'

const ensureTeamIDs = vi.fn()
const ensureUserIDs = vi.fn()
const ensureTagIDs = vi.fn()
const childProps = { modelValue: undefined }

const ChildStub = defineComponent({
  props: { modelValue: { type: [String, Number, Array], default: undefined } },
  setup (props) {
    childProps.modelValue = props.modelValue
    return () => h('div')
  }
})

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key) => key }) }))
vi.mock('@shared-ui/components/ui/select', () => ({ SelectTag: ChildStub }))
vi.mock('@/components/combobox/SelectCombobox.vue', () => ({ default: ChildStub }))
vi.mock('@/stores/team', () => ({
  useTeamStore: () => ({ options: [], fetchTeams: vi.fn(), searchTeams: vi.fn(), ensureTeamIDs })
}))
vi.mock('@/stores/users', () => ({
  useUsersStore: () => ({ options: [], fetchUsers: vi.fn(), searchUsers: vi.fn(), ensureUserIDs })
}))
vi.mock('@/stores/user', () => ({ useUserStore: () => ({ userID: 1 }) }))
vi.mock('@/stores/tag', () => ({
  useTagStore: () => ({
    tagOptions: [],
    tagNames: [],
    fetchTags: vi.fn(),
    searchTagNames: vi.fn(),
    searchTagOptions: vi.fn(),
    ensureTagIDs
  })
}))

const SelectTeamCombobox = (await import('./SelectTeamCombobox.vue')).default
const SelectAgentCombobox = (await import('./SelectAgentCombobox.vue')).default
const SelectTagCombobox = (await import('./SelectTagCombobox.vue')).default

const renderWithKebabModelValue = (component, props) =>
  renderToString(createSSRApp(defineComponent({ render: () => h(component, props) })))

describe('remote comboboxes with a kebab-case model-value', () => {
  beforeEach(() => {
    ensureTeamIDs.mockClear()
    ensureUserIDs.mockClear()
    ensureTagIDs.mockClear()
    childProps.modelValue = undefined
  })

  it('pins the selected team id so an out-of-page team keeps its label', async () => {
    await renderWithKebabModelValue(SelectTeamCombobox, { 'model-value': 7 })
    expect(ensureTeamIDs).toHaveBeenCalledWith([7])
    expect(childProps.modelValue).toBe(7)
  })

  it('pins the selected agent id', async () => {
    await renderWithKebabModelValue(SelectAgentCombobox, { 'model-value': 42 })
    expect(ensureUserIDs).toHaveBeenCalledWith([42])
    expect(childProps.modelValue).toBe(42)
  })

  it('pins the selected tag ids', async () => {
    await renderWithKebabModelValue(SelectTagCombobox, { 'model-value': [3, 9], 'value-field': 'id', multiple: true })
    expect(ensureTagIDs).toHaveBeenCalledWith([3, 9])
    expect(childProps.modelValue).toEqual([3, 9])
  })

  it('leaves the child model value untouched when no value is given', async () => {
    await renderWithKebabModelValue(SelectTeamCombobox, {})
    expect(childProps.modelValue).toBeUndefined()
  })
})
