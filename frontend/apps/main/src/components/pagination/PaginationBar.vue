<template>
  <div class="sticky bottom-0 bg-background pt-3 pb-4 mt-auto">
    <div
      class="flex flex-col sm:flex-row items-center justify-between gap-4 rounded-md border border-border shadow-sm px-4 py-3"
    >
      <div class="flex items-center gap-3">
        <span class="text-sm text-muted-foreground tabular-nums">
          {{ t('globals.messages.pageNofTotal', { page, total: totalPages }) }}
        </span>
        <Select :model-value="perPage" @update:model-value="handlePerPageChange">
          <SelectTrigger
            class="h-10 w-[70px] sm:h-8"
            :aria-label="t('globals.messages.resultsPerPage')"
          >
            <SelectValue :placeholder="String(perPage)" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem v-for="option in perPageOptions" :key="option" :value="option">
              {{ option }}
            </SelectItem>
          </SelectContent>
        </Select>
      </div>

      <div class="flex items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :class="EDGE_BUTTON_CLASS"
          :aria-label="t('globals.messages.firstPage')"
          :disabled="page <= 1"
          @click="goToPage(1)"
        >
          <ChevronsLeft :class="PAGINATION_ICON_CLASS" aria-hidden="true" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :class="STEP_BUTTON_CLASS"
          :aria-label="t('globals.messages.previousPage')"
          :disabled="page <= 1"
          @click="goToPage(page - 1)"
        >
          <ChevronLeft :class="PAGINATION_ICON_CLASS" aria-hidden="true" />
        </Button>

        <div class="flex items-center bg-muted rounded-lg p-1">
          <template v-for="pageNumber in visiblePages" :key="pageNumber">
            <span
              v-if="pageNumber === '...'"
              class="flex h-10 w-10 items-center justify-center text-sm text-muted-foreground select-none sm:h-7 sm:w-7"
            >
              ...
            </span>
            <button
              v-else
              type="button"
              @click="goToPage(pageNumber)"
              class="h-10 min-w-10 px-2 rounded-md text-sm font-medium transition-all duration-150 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring sm:h-7 sm:min-w-7"
              :class="
                pageNumber === page
                  ? 'bg-background text-foreground shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              "
            >
              {{ pageNumber }}
            </button>
          </template>
        </div>

        <Button
          type="button"
          variant="ghost"
          size="sm"
          :class="STEP_BUTTON_CLASS"
          :aria-label="t('globals.messages.nextPage')"
          :disabled="page >= totalPages"
          @click="goToPage(page + 1)"
        >
          <ChevronRight :class="PAGINATION_ICON_CLASS" aria-hidden="true" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :class="EDGE_BUTTON_CLASS"
          :aria-label="t('globals.messages.lastPage')"
          :disabled="page >= totalPages"
          @click="goToPage(totalPages)"
        >
          <ChevronsRight :class="PAGINATION_ICON_CLASS" aria-hidden="true" />
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Button } from '@shared-ui/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { ChevronsLeft, ChevronLeft, ChevronRight, ChevronsRight } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { getVisiblePages } from '@main/utils/pagination'

const EDGE_BUTTON_CLASS = 'hidden h-8 w-8 p-0 sm:inline-flex'
const PAGINATION_ICON_CLASS = 'h-4 w-4'
const STEP_BUTTON_CLASS = 'h-10 w-10 p-0 sm:h-8 sm:w-8'

const props = defineProps({
  page: { type: Number, required: true },
  perPage: { type: Number, required: true },
  totalPages: { type: Number, required: true },
  perPageOptions: { type: Array, default: () => [15, 30, 50, 100] }
})

const emit = defineEmits(['update:page', 'update:perPage', 'change'])

const { t } = useI18n()

const visiblePages = computed(() => getVisiblePages(props.page, props.totalPages))

function goToPage(p) {
  if (p >= 1 && p <= props.totalPages && p !== props.page) {
    emit('update:page', p)
    emit('change', { page: p, perPage: props.perPage })
  }
}

function handlePerPageChange(newPerPage) {
  emit('update:perPage', newPerPage)
  emit('update:page', 1)
  emit('change', { page: 1, perPage: newPerPage })
}
</script>
