// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'
import { createI18n } from 'vue-i18n'
import { useVirtualizer } from '@tanstack/vue-virtual'
import DataTable from './DataTable.vue'
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import DataTableActionTrigger from './DataTableActionTrigger.vue'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem
} from '@shared-ui/components/ui/dropdown-menu'

vi.mock('@tanstack/vue-virtual', async () => {
  const { computed } = await import('vue')
  return {
    useVirtualizer: vi.fn((options) =>
      computed(() => ({
        getVirtualItems: () =>
          Array.from({ length: options.count }, (_, index) => ({
            index,
            start: index * 52,
            end: (index + 1) * 52
          })),
        getTotalSize: () => options.count * 52,
        measureElement: vi.fn()
      })))
  }
})

let app
let container

function mountTable({
  overlay = false,
  pageLoading = false,
  tableLoading = false,
  toolbar = false,
  records = [{ id: 1, name: 'Customer A' }, { id: 2, name: 'Customer B' }]
} = {}) {
  const search = ref('')
  const pageBusy = ref(pageLoading)
  const tableBusy = ref(tableLoading)
  const tableData = ref(records)
  container = document.createElement('div')
  document.body.append(container)
  app = createApp({
    setup() {
      const content = () => [
        toolbar &&
          h(AdminToolbar, {
            searchable: true,
            search: search.value,
            'onUpdate:search': (value) => {
              search.value = value
            }
          }),
        h(DataTable, {
          columns: [{ accessorKey: 'name', header: 'Name' }],
          data: tableBusy.value ? [] : tableData.value,
          loading: tableBusy.value,
          searchable: !toolbar,
          search: search.value,
          'onUpdate:search': (value) => {
            search.value = value
          }
        })
      ]
      return () =>
        overlay
          ? h(LoadingOverlay, { loading: pageBusy.value }, { default: content })
          : h('div', content())
    }
  })
  app.use(
    createI18n({
      legacy: false,
      locale: 'en',
      messages: {
        en: {
          'globals.terms.search': 'Search',
          'globals.terms.loading': 'Loading',
          'globals.messages.clearSearch': 'Clear search',
          'globals.messages.noResultsFound': 'No results found'
        }
      }
    })
  )
  app.mount(container)
  return { search, pageBusy, tableBusy, tableData }
}

afterEach(() => {
  app?.unmount()
  container?.remove()
})

describe('DataTable row identity', () => {
  it('keeps record identities through deletion, replacement, sorting and filtering', async () => {
    const { tableData, search } = mountTable({
      records: [{ id: 0, name: 'Customer B' }, { id: 42, name: 'Customer A' }]
    })
    const options = useVirtualizer.mock.lastCall[0]
    const rowKeys = () => Array.from({ length: options.count }, (_, index) => options.getItemKey(index))
    expect(rowKeys()).toEqual(['0', '42'])
    const retainedRow = container.querySelectorAll('tbody tr[data-index]')[1]

    tableData.value = [{ id: 42, name: 'Customer A' }]
    await nextTick()
    expect(rowKeys()).toEqual(['42'])
    expect(container.querySelector('tbody tr[data-index]')).toBe(retainedRow)

    tableData.value = [{ id: 99, name: 'Customer C' }, { id: 42, name: 'Customer A' }]
    await nextTick()
    expect(rowKeys()).toEqual(['99', '42'])
    expect(container.querySelectorAll('tbody tr[data-index]')[1]).toBe(retainedRow)

    container.querySelector('thead button').click()
    await nextTick()
    expect(rowKeys()).toEqual(['42', '99'])

    search.value = 'Customer C'
    await nextTick()
    expect(rowKeys()).toEqual(['99'])

    tableData.value = []
    await nextTick()
    expect(rowKeys()).toEqual([])
  })
})

describe('DataTable toolbar search', () => {
  it('filters through the external toolbar and clears through its button', async () => {
    mountTable({ toolbar: true })
    expect(container.querySelectorAll('input')).toHaveLength(1)
    const input = container.querySelector('input')
    input.value = 'Customer B'
    input.dispatchEvent(new Event('input', { bubbles: true }))
    await nextTick()
    await nextTick()
    expect(container.querySelector('tbody').textContent).toContain('Customer B')
    expect(container.querySelector('tbody').textContent).not.toContain('Customer A')
    container.querySelector('button[aria-label="Clear search"]').click()
    await nextTick()
    await nextTick()
    expect(container.querySelector('tbody').textContent).toContain('Customer A')
    expect(container.querySelector('tbody').textContent).toContain('Customer B')
  })
})

describe('DataTable loading', () => {
  it('shows one spinner when the page and table load together', async () => {
    const state = mountTable({ overlay: true, pageLoading: true, tableLoading: true })
    expect(container.querySelectorAll('.animate-spin')).toHaveLength(1)
    expect(container.querySelector('tbody').textContent).not.toContain('No results found')
    state.pageBusy.value = false
    await nextTick()
    expect(container.querySelectorAll('.animate-spin')).toHaveLength(1)
    state.tableBusy.value = false
    await nextTick()
    expect(container.querySelectorAll('.animate-spin')).toHaveLength(0)
    expect(container.querySelector('tbody').textContent).toContain('Customer A')
  })

  it('keeps a standalone table spinner', () => {
    mountTable({ tableLoading: true })
    expect(container.querySelectorAll('.animate-spin')).toHaveLength(1)
  })
})

describe('DataTable row actions', () => {
  it('opens the shared action menu from the keyboard', async () => {
    container = document.createElement('div')
    document.body.append(container)
    app = createApp({
      render: () =>
        h(
          DropdownMenu,
          {},
          {
            default: () => [
              h(DataTableActionTrigger),
              h(
                DropdownMenuContent,
                {},
                {
                  default: () => h(DropdownMenuItem, {}, { default: () => 'Edit' })
                }
              )
            ]
          }
        )
    })
    app.config.globalProperties.$t = () => 'More actions'
    app.mount(container)
    const trigger = container.querySelector('button')
    expect(trigger.getAttribute('aria-label')).toBe('More actions')
    expect(trigger.type).toBe('button')
    trigger.focus()
    trigger.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await nextTick()
    await nextTick()
    expect(trigger.getAttribute('aria-expanded')).toBe('true')
    expect(document.querySelector('[role="menuitem"]').textContent).toBe('Edit')
  })
})
