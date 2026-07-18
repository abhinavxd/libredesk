<template>
  <div class="flex flex-col items-center justify-center h-64 px-6 text-center select-none">
    <component :is="icon" :stroke-width="1" :size="40" class="text-muted-foreground opacity-40" />
    <p class="text-sm font-medium text-foreground mt-5">
      {{ title }}
    </p>
    <p class="text-sm text-muted-foreground mt-1.5 max-w-[15rem] leading-relaxed">
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
