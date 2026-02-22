<template>
  <span
    v-if="count > 0"
    class="bg-primary text-primary-foreground text-xs rounded-full min-w-[1.2rem] h-5 flex items-center justify-center font-bold"
  >
    {{ count }}
  </span>
</template>

<script setup>
defineProps({
  count: {
    type: Number,
    required: true
  }
})
</script>
