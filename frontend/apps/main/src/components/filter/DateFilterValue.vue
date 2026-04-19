<template>
  <div @change.stop @input.stop>
    <template v-if="range">
      <div class="flex gap-2">
        <Popover v-model:open="startOpen">
          <PopoverTrigger as-child>
            <Button
              variant="outline"
              :class="
                cn(
                  'flex-1 justify-start text-left font-normal',
                  !start && 'text-muted-foreground'
                )
              "
            >
              <CalendarIcon class="mr-2 h-4 w-4" />
              {{ start ? formatDisplay(start) : t('globals.terms.pickDate') }}
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-auto p-0">
            <Calendar
              :model-value="toCalendarDate(start)"
              @update:model-value="pickStart"
            />
          </PopoverContent>
        </Popover>
        <Popover v-model:open="endOpen">
          <PopoverTrigger as-child>
            <Button
              variant="outline"
              :class="
                cn(
                  'flex-1 justify-start text-left font-normal',
                  !end && 'text-muted-foreground'
                )
              "
            >
              <CalendarIcon class="mr-2 h-4 w-4" />
              {{ end ? formatDisplay(end) : t('globals.terms.pickDate') }}
            </Button>
          </PopoverTrigger>
          <PopoverContent class="w-auto p-0">
            <Calendar
              :model-value="toCalendarDate(end)"
              @update:model-value="pickEnd"
            />
          </PopoverContent>
        </Popover>
      </div>
    </template>
    <template v-else>
      <Popover v-model:open="open">
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
    </template>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import { Calendar as CalendarIcon } from 'lucide-vue-next'
import { parseDate } from '@internationalized/date'
import { format } from 'date-fns'
import { useI18n } from 'vue-i18n'
import { cn } from '@shared-ui/lib/utils.js'
import { Button } from '@shared-ui/components/ui/button'
import { Popover, PopoverContent, PopoverTrigger } from '@shared-ui/components/ui/popover'
import { Calendar } from '@shared-ui/components/ui/calendar'

const { t } = useI18n()
const modelValue = defineModel({ type: String, default: '' })

const props = defineProps({
  range: { type: Boolean, default: false }
})

const open = ref(false)
const startOpen = ref(false)
const endOpen = ref(false)

const start = ref('')
const end = ref('')
let emittingRange = false

watch(
  [() => props.range, modelValue],
  ([isRange, value]) => {
    if (!isRange) return
    if (emittingRange) {
      emittingRange = false
      return
    }
    const [s = '', e = ''] = (value || '').split(',')
    start.value = s.trim()
    end.value = e.trim()
  },
  { immediate: true }
)

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

const handlePick = (v) => {
  modelValue.value = v ? v.toString() : ''
  open.value = false
}

const emitRange = () => {
  const next = start.value && end.value ? `${start.value},${end.value}` : ''
  if (next !== modelValue.value) {
    emittingRange = true
    modelValue.value = next
  }
}

const pickStart = (v) => {
  start.value = v ? v.toString() : ''
  emitRange()
  startOpen.value = false
}

const pickEnd = (v) => {
  end.value = v ? v.toString() : ''
  emitRange()
  endOpen.value = false
}
</script>
