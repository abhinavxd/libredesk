// @vitest-environment jsdom
import { afterEach, describe, expect, it, vi } from 'vitest'
import { createApp, h, nextTick, ref } from 'vue'
import { createI18n } from 'vue-i18n'
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
    useVirtualizer: (options) =>
      computed(() => ({
        getVirtualItems: () =>
          Array.from({ length: options.count }, (_, index) => ({
            index,
            start: index * 52,
            end: (index + 1) * 52
          })),
        getTotalSize: () => options.count * 52
      }))
  }
})

let app
let container

function mountTable({
  overlay = false,
  pageLoading = false,
  tableLoading = false,
  toolbar = false
} = {}) {
  const search = ref('')
  const pageBusy = ref(pageLoading)
  const tableBusy = ref(tableLoading)
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
          data: tableBusy.value ? [] : [{ name: 'Customer A' }, { name: 'Customer B' }],
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
  return { search, pageBusy, tableBusy }
}

afterEach(() => {
  app?.unmount()
  container?.remove()
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
