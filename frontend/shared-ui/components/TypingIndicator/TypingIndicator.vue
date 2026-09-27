<template>
  <div
    class="inline-flex w-fit items-center px-3 py-2 text-sm text-muted-foreground"
    :class="bubble ? 'rounded-2xl bg-muted' : ''"
  >
    <div class="flex space-x-1">
      <div class="w-2 h-2 bg-current rounded-full animate-bounce" style="animation-delay: 0ms"></div>
      <div class="w-2 h-2 bg-current rounded-full animate-bounce" style="animation-delay: 150ms"></div>
      <div class="w-2 h-2 bg-current rounded-full animate-bounce" style="animation-delay: 300ms"></div>
    </div>
  </div>
</template>

<script setup>
defineProps({
  bubble: { type: Boolean, default: false }
})
</script>
