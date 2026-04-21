<template>
  <div @change.stop @input.stop>
    <Popover v-if="range" v-model:open="rangeOpen">
      <PopoverTrigger as-child>
        <Button
          variant="outline"
          :class="
            cn(
              'w-full justify-start text-left font-normal',
              !rangeLabel && 'text-muted-foreground'
            )
          "
        >
          <CalendarIcon class="mr-2 h-4 w-4" />
          {{ rangeLabel || t('globals.terms.pickDate') }}
        </Button>
      </PopoverTrigger>
      <PopoverContent class="w-auto p-0">
        <RangeCalendar
          :model-value="dateRange"
          :number-of-months="2"
          @update:model-value="handleRangePick"
        />
      </PopoverContent>
    </Popover>
    <Popover v-else v-model:open="open">
      <PopoverTrigger as-child>
        <Button
          variant="outline"
          :class="
            cn(
              'w-full justify-start text-left font-normal',
              !modelValue && 'text-muted-foreground'
            )
          "
        >
          <CalendarIcon class="mr-2 h-4 w-4" />
          {{ modelValue ? formatDisplay(modelValue) : t('globals.terms.pickDate') }}
        </Button>
      </PopoverTrigger>
      <PopoverContent class="w-auto p-0">
        <Calendar
          :model-value="toCalendarDate(modelValue)"
          @update:model-value="handlePick"
        />
      </PopoverContent>
    </Popover>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { Calendar as CalendarIcon } from 'lucide-vue-next'
import { parseDate } from '@internationalized/date'
import { format } from 'date-fns'
import { useI18n } from 'vue-i18n'
import { cn } from '@shared-ui/lib/utils.js'
import { Button } from '@shared-ui/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { Calendar } from '@shared-ui/components/ui/calendar'
import { RangeCalendar } from '@shared-ui/components/ui/range-calendar'

const { t } = useI18n()
const modelValue = defineModel({ type: String, default: '' })

defineProps({
  range: { type: Boolean, default: false }
})

const open = ref(false)
const rangeOpen = ref(false)

const toCalendarDate = (v) => {
  if (!v) return undefined
  try {
    return parseDate(v)
  } catch {
    return undefined
  }
}

const formatDisplay = (v) => {
  try {
    return format(new Date(v), 'MMM dd, yyyy')
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

const handleRangePick = (v) => {
  if (!v) {
    modelValue.value = ''
    return
  }
  const start = v.start ? v.start.toString() : ''
  const end = v.end ? v.end.toString() : ''
  modelValue.value = start && end ? `${start},${end}` : ''
  if (start && end) rangeOpen.value = false
}
</script>
