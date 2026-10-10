<template>
  <div
    class="inline-flex w-fit items-center px-3 py-2 text-sm text-muted-foreground"
    :class="bubble ? 'rounded-2xl bg-muted' : ''"
  >
    <div class="flex space-x-1">
      <div
        v-for="delay in [0, 150, 300]"
        :key="delay"
        class="h-2 w-2 rounded-full bg-current motion-reduce:animate-none"
        :class="bubble ? 'typing-dot' : 'animate-bounce'"
        :style="{ animationDelay: `${delay}ms` }"
      ></div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  bubble: { type: Boolean, default: false }
})
</script>

<style scoped>
.typing-dot {
  animation: typing-dot 1s cubic-bezier(0.77, 0, 0.175, 1) infinite;
}

@keyframes typing-dot {
  0%,
  60%,
  100% {
    transform: translateY(0);
  }
  30% {
    transform: translateY(-50%);
  }
}

@media (prefers-reduced-motion: reduce) {
  .typing-dot {
    animation: none;
  }
}
</style>
