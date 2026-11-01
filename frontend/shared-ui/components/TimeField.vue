<script setup>
import { computed } from 'vue'
import { useI18n } from '@shared-ui/i18n'
import { Time } from '@internationalized/date'
import { TimeFieldInput, TimeFieldRoot } from 'reka-ui'
import { cn } from '@shared-ui/lib/utils'

const props = defineProps({
  modelValue: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  class: { type: null, required: false }
})
const emit = defineEmits(['update:modelValue'])
const { locale } = useI18n()

const time = computed(() => {
  if (!props.modelValue) return null
  const [hour, minute] = props.modelValue.split(':').map(Number)
  return new Time(hour, minute)
})

const updateTime = (value) => {
  const formatted = value
    ? `${String(value.hour).padStart(2, '0')}:${String(value.minute).padStart(2, '0')}`
    : ''
  emit('update:modelValue', formatted)
}
</script>

<template>
  <TimeFieldRoot
    data-time-field
    v-slot="{ segments }"
    :model-value="time"
    :disabled="disabled"
    :locale="locale"
    :hour-cycle="24"
    granularity="minute"
    :class="
      cn(
        'flex h-9 w-full items-center rounded-md border border-input bg-transparent px-3 py-1 text-base sm:text-sm shadow-sm focus-within:ring-1 focus-within:ring-ring data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50',
        props.class
      )
    "
    @update:model-value="updateTime"
  >
    <TimeFieldInput
      v-for="segment in segments"
      :key="segment.part"
      :part="segment.part"
      class="rounded-sm tabular-nums outline-none focus:bg-accent focus:text-accent-foreground data-[disabled]:cursor-not-allowed"
    >
      {{ segment.value }}
    </TimeFieldInput>
  </TimeFieldRoot>
</template>
