<template>
  <div class="w-full space-y-3">
    <SearchInput
      v-if="searchable"
      v-model="globalFilter"
      class="max-w-xs"
      :placeholder="searchPlaceholder || i18n._('Search')"
      :clear-label="i18n._('Clear search')"
    />

    <div
      ref="scrollContainer"
      class="relative overflow-auto rounded-lg border border-border bg-card shadow-sm"
      :style="{ maxHeight }"
      :aria-busy="loading"
    >
      <table class="w-full table-auto caption-bottom text-sm">
        <TableHeader class="sticky top-0 z-10 bg-card">
          <TableRow
            v-for="headerGroup in table.getHeaderGroups()"
            :key="headerGroup.id"
            class="border-b border-border bg-muted/40 hover:bg-muted/40"
          >
            <TableHead
              v-for="(header, headerIndex) in headerGroup.headers"
              :key="header.id"
              scope="col"
              :aria-sort="
                header.column.getCanSort()
                  ? header.column.getIsSorted() === 'asc'
                    ? 'ascending'
                    : header.column.getIsSorted() === 'desc'
                      ? 'descending'
                      : 'none'
                  : undefined
              "
              class="h-11 px-4 text-left text-sm font-medium text-muted-foreground"
              :class="{
                'max-sm:hidden': isHiddenOnPhone(header.column.id, headerIndex),
                'whitespace-nowrap': header.column.columnDef.meta?.compact || header.column.id === 'actions',
                'w-px': header.column.id === 'actions',
                'min-w-40': !header.column.columnDef.meta?.compact && header.column.id !== 'actions'
              }"
            >
              <Button
                v-if="header.column.getCanSort()"
                type="button"
                variant="ghost"
                size="sm"
                class="group h-auto justify-start gap-2 whitespace-normal px-0 py-2 text-left text-muted-foreground can-hover:hover:bg-transparent can-hover:hover:text-foreground"
                :class="{ 'whitespace-nowrap': header.column.columnDef.meta?.compact }"
                @click="header.column.getToggleSortingHandler()?.($event)"
              >
                <FlexRender
                  v-if="!header.isPlaceholder"
                  :render="header.column.columnDef.header"
                  :props="header.getContext()"
                />
                <ChevronUp
                  v-if="header.column.getIsSorted() === 'asc'"
                  class="h-3.5 w-3.5 shrink-0"
                  aria-hidden="true"
                />
                <ChevronDown
                  v-else-if="header.column.getIsSorted() === 'desc'"
                  class="h-3.5 w-3.5 shrink-0"
                  aria-hidden="true"
                />
                <ArrowUpDown
                  v-else
                  class="h-3.5 w-3.5 shrink-0 opacity-50 can-hover:opacity-0 can-hover:group-hover:opacity-50 group-focus-visible:opacity-50"
                  aria-hidden="true"
                />
              </Button>
              <div v-else class="flex items-center gap-2">
                <FlexRender
                  v-if="!header.isPlaceholder"
                  :render="header.column.columnDef.header"
                  :props="header.getContext()"
                />
              </div>
            </TableHead>
          </TableRow>
        </TableHeader>

        <TableBody>
          <template v-if="rows.length">
            <tr v-if="paddingTop > 0" :style="{ height: `${paddingTop}px` }"></tr>
            <TableRow
              v-for="virtualRow in virtualRows"
              :key="rows[virtualRow.index].id"
              :ref="measureRow"
              :data-index="virtualRow.index"
              :data-state="rows[virtualRow.index].getIsSelected() ? 'selected' : undefined"
              class="group/row border-b border-border/50 transition-colors last:border-0 hover:bg-muted/30 data-[state=selected]:bg-muted"
            >
              <TableCell
                v-for="(cell, cellIndex) in rows[virtualRow.index].getVisibleCells()"
                :key="cell.id"
                class="px-4 py-3 text-left text-sm [overflow-wrap:anywhere]"
                :class="[
                  cell.column.id === 'actions' ? actionCellClass : '',
                  cell.column.columnDef.meta?.compact && 'whitespace-nowrap',
                  isHiddenOnPhone(cell.column.id, cellIndex) && 'max-sm:hidden'
                ]"
              >
                <div
                  :class="{ 'line-clamp-3': !cell.column.columnDef.meta?.compact && cell.column.id !== 'actions' }"
                  :title="!cell.column.columnDef.meta?.compact && typeof cell.getValue() === 'string' ? cell.getValue() : undefined"
                >
                  <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
                </div>
              </TableCell>
            </TableRow>
            <tr v-if="paddingBottom > 0" :style="{ height: `${paddingBottom}px` }"></tr>
          </template>

          <template v-else-if="loading">
            <TableRow class="hover:bg-transparent">
              <TableCell :colspan="columns.length" class="h-32">
                <div class="flex items-center justify-center">
                  <Spinner
                    v-if="!overlayLoading"
                    :absolute="false"
                    :text="i18n._('Loading')"
                  />
                </div>
              </TableCell>
            </TableRow>
          </template>

          <template v-else>
            <TableRow class="hover:bg-transparent">
              <TableCell :colspan="columns.length" class="h-32">
                <div class="flex flex-col items-center justify-center gap-2 text-center">
                  <Ghost class="h-8 w-8 text-muted-foreground/50" aria-hidden="true" />
                  <p class="text-sm font-medium text-muted-foreground">{{ emptyText }}</p>
                </div>
              </TableCell>
            </TableRow>
          </template>
        </TableBody>
      </table>
    </div>
  </div>
</template>

<script setup>
import {
  FlexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  useVueTable
} from '@tanstack/vue-table'
import { useVirtualizer } from '@tanstack/vue-virtual'
import { useI18n } from '@shared-ui/i18n'
import { computed, inject, ref } from 'vue'
import { LOADING_OVERLAY_INJECTION_KEY } from '@main/components/layout/loadingOverlay'
import { ArrowUpDown, ChevronDown, ChevronUp, Ghost } from 'lucide-vue-next'
import SearchInput from '@shared-ui/components/SearchInput.vue'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@shared-ui/components/ui/table'

const i18n = useI18n()
const overlayLoading = inject(LOADING_OVERLAY_INJECTION_KEY, ref(false))

// Hidden until row-hover on pointer devices; always shown on touch, focus, and while the menu is open.
const actionCellClass =
  'w-px whitespace-nowrap transition-opacity duration-150 can-hover:opacity-0 can-hover:group-hover/row:opacity-100 focus-within:!opacity-100 [&:has([data-state=open])]:!opacity-100 max-md:[&_button]:size-11'

const isHiddenOnPhone = (columnId, index) =>
  (index > 1 && columnId !== 'actions') || ['created_at', 'updated_at'].includes(columnId)

const props = defineProps({
  columns: Array,
  data: Array,
  emptyText: {
    type: String,
    default: ''
  },
  loading: {
    type: Boolean,
    default: false
  },
  maxHeight: {
    type: String,
    default: '80vh'
  },
  estimatedRowHeight: {
    type: Number,
    default: 52
  },
  searchable: {
    type: Boolean,
    default: true
  },
  searchPlaceholder: {
    type: String,
    default: ''
  }
})

const sorting = ref([])
const globalFilter = defineModel('search', { type: String, default: '' })

const emptyText = computed(() => props.emptyText || i18n._('No results found'))

const table = useVueTable({
  get data() {
    return props.data
  },
  get columns() {
    return props.columns
  },
  getRowId: (row) => String(row.id),
  state: {
    get sorting() {
      return sorting.value
    },
    get globalFilter() {
      return globalFilter.value
    }
  },
  enableSortingRemoval: false,
  onSortingChange: (updaterOrValue) => {
    sorting.value =
      typeof updaterOrValue === 'function' ? updaterOrValue(sorting.value) : updaterOrValue
  },
  onGlobalFilterChange: (updaterOrValue) => {
    globalFilter.value =
      typeof updaterOrValue === 'function' ? updaterOrValue(globalFilter.value) : updaterOrValue
  },
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel(),
  getFilteredRowModel: getFilteredRowModel()
})

const scrollContainer = ref(null)
const rows = computed(() => table.getRowModel().rows)
const getRowKey = computed(() => {
  const currentRows = rows.value
  return (index) => currentRows[index].id
})

const rowVirtualizer = useVirtualizer({
  get count() {
    return rows.value.length
  },
  get getItemKey() {
    return getRowKey.value
  },
  getScrollElement: () => scrollContainer.value,
  estimateSize: () => props.estimatedRowHeight,
  overscan: 10
})

// Wrapped rows change height when content or available width changes.
const measureRow = (row) => rowVirtualizer.value.measureElement(row?.$el)

const virtualRows = computed(() => rowVirtualizer.value.getVirtualItems())
const totalSize = computed(() => rowVirtualizer.value.getTotalSize())
const paddingTop = computed(() => virtualRows.value[0]?.start ?? 0)
const paddingBottom = computed(() => {
  const last = virtualRows.value[virtualRows.value.length - 1]
  return last ? totalSize.value - last.end : 0
})
</script>
