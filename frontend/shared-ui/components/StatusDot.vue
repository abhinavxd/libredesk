<template>
  <div
    class="rounded-full"
    :class="[sizeClass, statusClass]"
  />
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  status: {
    type: String,
    default: 'offline'
  },
  size: {
    type: String,
    default: 'md',
    validator: (value) => ['sm', 'md', 'lg'].includes(value)
  }
})

const sizeClass = computed(() => {
  const sizes = {
    sm: 'h-2 w-2',
    md: 'h-2.5 w-2.5',
    lg: 'h-3.5 w-3.5'
  }
  return sizes[props.size]
})

const statusClass = computed(() => {
  switch (props.status) {
    case 'online':
      return 'bg-green-500'
    case 'away':
    case 'away_manual':
    case 'away_and_reassigning':
      return 'bg-amber-500'
    default:
      return 'bg-gray-400'
  }
})
</script>
