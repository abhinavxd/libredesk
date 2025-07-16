<template>
  <div class="flex items-center justify-center p-4 border-b relative">
    <h3 class="text-base font-semibold text-foreground">{{ title }}</h3>
    <div class="absolute right-4">
      <CloseWidgetButton />
    </div>
  </div>
</template>
<script setup>
import CloseWidgetButton from '@widget/components/CloseWidgetButton.vue'

defineProps(['title'])
</script>
