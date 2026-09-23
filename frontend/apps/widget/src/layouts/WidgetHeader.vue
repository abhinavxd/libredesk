<template>
  <div class="relative flex items-center justify-center p-4 border-b">
    <h3 class="text-base font-semibold text-foreground">{{ title }}</h3>
    <CloseWidgetButton class="absolute right-2 inset-y-0 my-auto" />
  </div>
</template>
<script setup>
import CloseWidgetButton from '@widget/components/CloseWidgetButton.vue'

defineProps(['title'])
</script>
