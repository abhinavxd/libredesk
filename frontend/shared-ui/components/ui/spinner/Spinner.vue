<template>
  <div role="status" class="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2 z-50">
    <svg :class="spinnerClass" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle
        class="opacity-10"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="4"
      ></circle>
      <path
        :class="['opacity-75', fillColor]"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      ></path>
    </svg>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  size: {
    type: String,
    default: 'w-5 h-5'
  },
  color: {
    type: String,
    default: 'text-secondary'
  },
  fillColor: {
    type: String,
    default: 'fill-primary'
  }
})

const spinnerClass = computed(() => `${props.size} ${props.color} animate-spin animate-bounce-in`)
</script>
