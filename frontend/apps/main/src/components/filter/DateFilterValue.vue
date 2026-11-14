<template>
  <div @change.stop @input.stop>
    <Popover v-if="range" v-model:open="rangeOpen">
      <PopoverTrigger as-child>
        <Button
          type="button"
          variant="outline"
          :class="
            cn(
              'w-full justify-start text-left font-normal',
              !rangeLabel && 'text-muted-foreground'
            )
          "
        >
          <CalendarIcon class="mr-2 h-4 w-4" aria-hidden="true" />
          {{ rangeLabel || placeholder || t('globals.terms.pickDate') }}
        </Button>
      </PopoverTrigger>
      <PopoverContent class="w-auto p-0">
        <RangeCalendar
          :model-value="draftRange || dateRange"
          :number-of-months="2"
          @update:model-value="handleRangePick"
        />
        <div v-if="modelValue" class="flex justify-end border-t border-border p-2">
          <Button type="button" variant="ghost" size="sm" @click="clearRange">
            {{ t('globals.terms.clear') }}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
    <Popover v-else v-model:open="open">
      <PopoverTrigger as-child>
        <Button
          type="button"
          variant="outline"
          :class="
            cn(
              'w-full justify-start text-left font-normal',
              !modelValue && 'text-muted-foreground'
            )
          "
        >
          <CalendarIcon class="mr-2 h-4 w-4" aria-hidden="true" />
          {{ modelValue ? formatDisplay(modelValue) : placeholder || t('globals.terms.pickDate') }}
        </Button>
      </PopoverTrigger>
      <PopoverContent class="w-auto p-0">
        <Calendar
          :model-value="dateValue"
          @update:model-value="handlePick"
        />
      </PopoverContent>
    </Popover>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { Calendar as CalendarIcon } from 'lucide-vue-next'
import { parseDate } from '@internationalized/date'
import { format, parseISO } from 'date-fns'
import { useI18n } from 'vue-i18n'
import { cn } from '@shared-ui/lib/utils.js'
import { Button } from '@shared-ui/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { Calendar } from '@shared-ui/components/ui/calendar'
import { RangeCalendar } from '@shared-ui/components/ui/range-calendar'

const { t } = useI18n()
const modelValue = defineModel({ type: String, default: '' })

defineProps({
  range: { type: Boolean, default: false },
  placeholder: { type: String, default: '' }
})

const open = ref(false)
const rangeOpen = ref(false)
const draftRange = ref(null)

watch(rangeOpen, (isOpen) => {
  if (!isOpen) draftRange.value = null
})

const toCalendarDate = (v) => {
  if (!v) return undefined
  try {
    return parseDate(v)
  } catch {
    return undefined
  }
}

const dateValue = computed(() => toCalendarDate(modelValue.value))

const formatDisplay = (v) => {
  try {
    // parseISO reads "2026-01-01" as local midnight. new Date() reads it as UTC and shows the previous day west of UTC.
    return format(parseISO(v), 'MMM dd, yyyy')
  } catch {
    return v
  }
}

const dateRange = computed(() => {
  const [s = '', e = ''] = (modelValue.value || '').split(',')
  return {
    start: toCalendarDate(s.trim()),
    end: toCalendarDate(e.trim())
  }
})

const rangeLabel = computed(() => {
  const { start, end } = dateRange.value
  if (!start && !end) return ''
  if (start && end) return `${formatDisplay(start.toString())} - ${formatDisplay(end.toString())}`
  if (start) return formatDisplay(start.toString())
  return formatDisplay(end.toString())
})

const handlePick = (v) => {
  modelValue.value = v ? v.toString() : ''
  open.value = false
}

// A half-picked range stays local so the applied range is kept until the end date is chosen.
const handleRangePick = (v) => {
  const start = v?.start ? v.start.toString() : ''
  const end = v?.end ? v.end.toString() : ''
  if (!start || !end) {
    draftRange.value = v || { start: undefined, end: undefined }
    return
  }
  draftRange.value = null
  modelValue.value = `${start},${end}`
  rangeOpen.value = false
}

const clearRange = () => {
  draftRange.value = null
  modelValue.value = ''
  rangeOpen.value = false
}
</script>
