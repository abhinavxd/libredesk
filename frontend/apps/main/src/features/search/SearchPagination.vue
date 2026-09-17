<template>
  <div class="sticky bottom-0 mt-auto bg-background pb-4 pt-3">
    <div
      class="flex items-center justify-between gap-4 rounded-md border border-border px-4 py-3 shadow-sm"
    >
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

      <div class="flex items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :class="STEP_BUTTON_CLASS"
          :aria-label="t('globals.messages.previousPage')"
          :disabled="page <= 1"
          @click="goToPage(page - 1)"
        >
          <ChevronLeft class="h-4 w-4" aria-hidden="true" />
        </Button>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :class="STEP_BUTTON_CLASS"
          :aria-label="t('globals.messages.nextPage')"
          :disabled="!hasMore"
          @click="goToPage(page + 1)"
        >
          <ChevronRight class="h-4 w-4" aria-hidden="true" />
        </Button>
      </div>
    </div>
  </div>
</template>

<script setup>
const STEP_BUTTON_CLASS = 'h-10 w-10 p-0 sm:h-8 sm:w-8'

import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { Button } from '@shared-ui/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'

const props = defineProps({
  page: { type: Number, required: true },
  perPage: { type: Number, required: true },
  hasMore: { type: Boolean, required: true },
  perPageOptions: { type: Array, default: () => [15, 30, 50, 100] }
})

const emit = defineEmits(['change'])

const { t } = useI18n()

function goToPage(page) {
  if (page < 1 || (page > props.page && !props.hasMore)) return
  emit('change', { page, perPage: props.perPage })
}

function handlePerPageChange(perPage) {
  emit('change', { page: 1, perPage })
}
</script>
