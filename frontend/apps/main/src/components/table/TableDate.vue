<script setup>
import { computed } from 'vue'
import { format, isValid } from 'date-fns'

const props = defineProps({
  value: { type: [String, Number, Date], default: null }
})

const date = computed(() => {
  if (props.value === null || props.value === '') return null
  const parsed = new Date(props.value)
  return isValid(parsed) ? parsed : null
})
const fullTimestamp = computed(() => date.value ? format(date.value, 'd MMM yyyy, h:mm:ss a OOOO') : '')
</script>

<template>
  <time
    v-if="date"
    :datetime="date.toISOString()"
    :title="fullTimestamp"
    :aria-label="fullTimestamp"
    class="whitespace-nowrap"
  >{{ format(date, 'd MMM yyyy, h:mm a') }}</time>
</template>
