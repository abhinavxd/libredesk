<template>
  <div class="w-full space-y-3">
    <div v-if="searchable" class="relative max-w-sm">
      <Search class="absolute left-2.5 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
      <Input
        v-model="globalFilter"
        :placeholder="searchPlaceholder || t('globals.terms.search')"
        class="pl-8"
      />
    </div>

    <div
      ref="scrollContainer"
      class="relative overflow-auto rounded-lg border border-border bg-card shadow-sm"
      :style="{ maxHeight }"
    >
      <table class="w-full caption-bottom text-sm">
        <TableHeader class="sticky top-0 z-10 bg-card">
          <TableRow
            v-for="headerGroup in table.getHeaderGroups()"
            :key="headerGroup.id"
            class="border-b border-border bg-muted/40 hover:bg-muted/40"
          >
            <TableHead
              v-for="header in headerGroup.headers"
              :key="header.id"
              class="h-11 px-4 text-center text-sm font-medium text-muted-foreground"
              :class="{
                'cursor-pointer select-none transition-colors hover:text-foreground':
                  header.column.getCanSort()
              }"
              @click="header.column.getToggleSortingHandler()?.($event)"
            >
              <div class="flex items-center justify-center gap-2">
                <FlexRender
                  v-if="!header.isPlaceholder"
                  :render="header.column.columnDef.header"
                  :props="header.getContext()"
                />
                <template v-if="header.column.getCanSort()">
                  <ChevronUp v-if="header.column.getIsSorted() === 'asc'" class="h-3.5 w-3.5" />
                  <ChevronDown
                    v-else-if="header.column.getIsSorted() === 'desc'"
                    class="h-3.5 w-3.5"
                  />
                  <ArrowUpDown
                    v-else
                    class="h-3.5 w-3.5 opacity-0 transition-opacity group-hover:opacity-50"
                  />
                </template>
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
              :data-index="virtualRow.index"
              :data-state="rows[virtualRow.index].getIsSelected() ? 'selected' : undefined"
              class="border-b border-border/50 transition-colors last:border-0 hover:bg-muted/30 data-[state=selected]:bg-muted"
            >
              <TableCell
                v-for="cell in rows[virtualRow.index].getVisibleCells()"
                :key="cell.id"
                class="px-4 py-3 text-center text-sm"
              >
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </TableCell>
            </TableRow>
            <tr v-if="paddingBottom > 0" :style="{ height: `${paddingBottom}px` }"></tr>
          </template>

          <template v-else-if="loading">
            <TableRow class="hover:bg-transparent">
              <TableCell :colspan="columns.length" class="h-32">
                <div class="flex items-center justify-center">
                  <p class="text-sm text-muted-foreground">{{ t('globals.terms.loading') }}</p>
                </div>
              </TableCell>
            </TableRow>
          </template>

          <template v-else>
            <TableRow class="hover:bg-transparent">
              <TableCell :colspan="columns.length" class="h-32">
                <div class="flex flex-col items-center justify-center gap-2 text-center">
                  <Ghost class="h-8 w-8 text-muted-foreground/50" />
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
import { useI18n } from 'vue-i18n'
import { computed, ref } from 'vue'
import { ArrowUpDown, ChevronDown, ChevronUp, Ghost, Search } from 'lucide-vue-next'
import {
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@shared-ui/components/ui/table'
import { Input } from '@shared-ui/components/ui/input'

const { t } = useI18n()

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
const globalFilter = ref('')

const emptyText = computed(() => props.emptyText || t('globals.messages.noResultsFound'))

const table = useVueTable({
  get data() {
    return props.data
  },
  get columns() {
    return props.columns
  },
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

const rowVirtualizer = useVirtualizer({
  get count() {
    return rows.value.length
  },
  getScrollElement: () => scrollContainer.value,
  estimateSize: () => props.estimatedRowHeight,
  overscan: 10
})

const virtualRows = computed(() => rowVirtualizer.value.getVirtualItems())
const totalSize = computed(() => rowVirtualizer.value.getTotalSize())
const paddingTop = computed(() => (virtualRows.value[0]?.start ?? 0))
const paddingBottom = computed(() => {
  const last = virtualRows.value[virtualRows.value.length - 1]
  return last ? totalSize.value - last.end : 0
})
</script>
