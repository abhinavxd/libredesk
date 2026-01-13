<template>
  <div class="w-full">
    <div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
      <Table>
        <TableHeader>
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
          <template v-if="table.getRowModel().rows?.length">
            <TableRow
              v-for="row in table.getRowModel().rows"
              :key="row.id"
              :data-state="row.getIsSelected() ? 'selected' : undefined"
              class="border-b border-border/50 transition-colors last:border-0 hover:bg-muted/30 data-[state=selected]:bg-muted"
            >
              <TableCell
                v-for="cell in row.getVisibleCells()"
                :key="cell.id"
                class="px-4 py-3 text-center text-sm"
              >
                <FlexRender :render="cell.column.columnDef.cell" :props="cell.getContext()" />
              </TableCell>
            </TableRow>
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
      </Table>
    </div>
  </div>
</template>

<script setup>
import { FlexRender, getCoreRowModel, getSortedRowModel, useVueTable } from '@tanstack/vue-table'
import { useI18n } from 'vue-i18n'
import { computed, ref } from 'vue'
import { ArrowUpDown, ChevronDown, ChevronUp, Ghost } from 'lucide-vue-next'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow
} from '@shared-ui/components/ui/table'

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
  }
})

const emptyText = computed(
  () =>
    props.emptyText ||
    t('globals.messages.noResults', {
      name: t('globals.terms.result', 2).toLowerCase()
    })
)

const sorting = ref([])

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
    }
  },
  enableSortingRemoval: false,
  onSortingChange: (updaterOrValue) => {
    sorting.value =
      typeof updaterOrValue === 'function' ? updaterOrValue(sorting.value) : updaterOrValue
  },
  getCoreRowModel: getCoreRowModel(),
  getSortedRowModel: getSortedRowModel()
})
</script>
