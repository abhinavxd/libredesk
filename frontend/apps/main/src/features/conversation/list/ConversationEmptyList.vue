<template>
  <div class="flex flex-col items-center justify-center h-64 space-y-2">
    <component :is="icon" :stroke-width="1" :size="50" />
    <h1 class="text-md font-semibold text-gray-800 dark:text-foreground">
      {{ title }}
    </h1>
    <p class="text-gray-600 dark:text-gray-300 text-center text-sm">
      {{ message }}
    </p>
  </div>
</template>

<script setup>
defineProps({
  icon: {
    type: Function,
    required: true
  },
  title: {
    type: String,
    required: true
  },
  message: {
    type: String,
    required: true
  }
})
</script>
