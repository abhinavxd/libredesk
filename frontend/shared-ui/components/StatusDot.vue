<template>
  <div class="rounded-full" :class="sizeClass">
    <svg class="block !size-full" viewBox="0 0 16 16" aria-hidden="true">
      <template v-if="isOnline">
        <circle cx="8" cy="8" r="8" class="fill-success" />
        <path
          d="M4.3 8.4 6.8 11 11.7 5.3"
          class="stroke-white"
          stroke-width="2.5"
          stroke-linecap="round"
          stroke-linejoin="round"
          fill="none"
        />
      </template>
      <template v-else-if="isAway">
        <circle cx="8" cy="8" r="8" class="fill-warning" />
        <path
          d="M8 8V4.5 M8 8H11"
          class="stroke-white"
          stroke-width="2.5"
          stroke-linecap="round"
          fill="none"
        />
      </template>
      <circle
        v-else
        cx="8"
        cy="8"
        r="6.75"
        fill="none"
        class="stroke-muted-foreground"
        stroke-width="2.5"
      />
    </svg>
  </div>
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

const isOnline = computed(() => props.status === 'online')

const isAway = computed(() =>
  ['away', 'away_manual', 'away_and_reassigning'].includes(props.status)
)

const sizeClass = computed(() => {
  const sizes = {
    sm: 'h-2.5 w-2.5',
    md: 'h-3.5 w-3.5',
    lg: 'h-4 w-4'
  }
  return sizes[props.size]
})
</script>
