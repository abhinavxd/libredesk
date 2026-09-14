<script setup>
defineProps({
  text: { type: String, default: '' },
  colorClass: { type: String, default: '' }
})
</script>

<template>
  <div
    v-if="text"
    role="status"
    aria-live="polite"
    aria-atomic="true"
    :class="['px-4 py-2 text-center text-sm', colorClass]"
  >
    <div class="flex items-center justify-center gap-2">
      <div class="w-2 h-2 rounded-full bg-current animate-pulse"></div>
      {{ text }}
    </div>
  </div>
</template>
