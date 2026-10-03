<script setup>
// The iOS activity indicator: eight spokes that appear clockwise from twelve
// o'clock as the list is pulled, then turn in steps once refreshing starts.
const SPOKES = 8

const props = defineProps({
  // 0 to 1: how far the pull is toward starting a refresh.
  progress: { type: Number, default: 0 },
  spinning: { type: Boolean, default: false }
})

// Spinning, the spokes trail off behind the brightest one. Pulling, each spoke
// fades in over its own slice of the pull.
const spokeOpacity = (index) =>
  props.spinning
    ? 0.3 + (0.7 * (index + 1)) / SPOKES
    : Math.min(1, Math.max(0, props.progress * SPOKES - index))
</script>

<template>
  <svg
    viewBox="0 0 20 20"
    class="h-5 w-5 text-muted-foreground"
    :class="{ 'refresh-spinner-turning': spinning }"
    aria-hidden="true"
  >
    <line
      v-for="index in SPOKES"
      :key="index"
      x1="10"
      y1="1.75"
      x2="10"
      y2="5.75"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      :transform="`rotate(${(index - 1) * (360 / SPOKES)} 10 10)`"
      :opacity="spokeOpacity(index - 1)"
    />
  </svg>
</template>

<style scoped>
.refresh-spinner-turning {
  animation: refresh-spinner-turn 1s steps(8) infinite;
}

@keyframes refresh-spinner-turn {
  to {
    transform: rotate(360deg);
  }
}
</style>
