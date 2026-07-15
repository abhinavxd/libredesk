<template>
  <div class="flex flex-col items-center justify-center h-64 space-y-2">
    <component :is="icon" :stroke-width="1" :size="50" />
    <h1 class="text-md font-semibold text-foreground">
      {{ title }}
    </h1>
    <p class="text-muted-foreground text-center text-sm">
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
