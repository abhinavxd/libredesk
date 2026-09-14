<template>
  <svg :width="size" :height="size" viewBox="0 0 16 16" xmlns="http://www.w3.org/2000/svg">
    <template v-if="level">
      <rect
        v-for="bar in 3"
        :key="bar"
        :x="1 + (bar - 1) * 5"
        :y="14 - bar * 4"
        width="3"
        :height="bar * 4"
        rx="0.8"
        :class="bar <= level ? 'fill-current' : 'fill-border'"
      />
    </template>
    <circle v-else cx="8" cy="8" r="6.5" fill="none" stroke="currentColor" stroke-width="1.5" />
  </svg>
</template>

<script setup>
defineProps({
  level: {
    type: Number,
    default: 0
  },
  size: {
    type: [Number, String],
    default: 16
  }
})
</script>
