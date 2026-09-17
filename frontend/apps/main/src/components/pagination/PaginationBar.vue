<template>
  <div class="sticky bottom-0 bg-background border-t border-border px-4 py-3 mt-auto">
    <div class="flex flex-col sm:flex-row items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <span class="text-sm text-muted-foreground tabular-nums">
          {{ t('globals.messages.pageNofTotal', { page, total: totalPages }) }}
        </span>
        <Select :model-value="perPage" @update:model-value="handlePerPageChange">
          <SelectTrigger class="h-8 w-[70px]">
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
          variant="ghost"
          size="sm"
          class="h-8 w-8 p-0"
          :disabled="page <= 1"
          @click="goToPage(1)"
        >
          <ChevronsLeft class="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          class="h-8 w-8 p-0"
          :disabled="page <= 1"
          @click="goToPage(page - 1)"
        >
          <ChevronLeft class="h-4 w-4" />
        </Button>

        <div class="flex items-center bg-muted rounded-lg p-1">
          <template v-for="pageNumber in visiblePages" :key="pageNumber">
            <span
              v-if="pageNumber === '...'"
              class="flex items-center justify-center h-7 w-7 text-sm text-muted-foreground select-none"
            >
              ...
            </span>
            <button
              v-else
              @click="goToPage(pageNumber)"
              class="h-7 min-w-7 px-2 rounded-md text-sm font-medium transition-all duration-150 focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
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
          variant="ghost"
          size="sm"
          class="h-8 w-8 p-0"
          :disabled="page >= totalPages"
          @click="goToPage(page + 1)"
        >
          <ChevronRight class="h-4 w-4" />
        </Button>
        <Button
          variant="ghost"
          size="sm"
          class="h-8 w-8 p-0"
          :disabled="page >= totalPages"
          @click="goToPage(totalPages)"
        >
          <ChevronsRight class="h-4 w-4" />
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

const props = defineProps({
  page: { type: Number, required: true },
  perPage: { type: Number, required: true },
  totalPages: { type: Number, required: true },
  perPageOptions: { type: Array, default: () => [15, 30, 50, 100] }
})

const emit = defineEmits(['update:page', 'update:perPage'])

const { t } = useI18n()

const visiblePages = computed(() => getVisiblePages(props.page, props.totalPages))

function goToPage(p) {
  if (p >= 1 && p <= props.totalPages && p !== props.page) {
    emit('update:page', p)
  }
}

function handlePerPageChange(newPerPage) {
  emit('update:perPage', newPerPage)
  emit('update:page', 1)
}
</script>
