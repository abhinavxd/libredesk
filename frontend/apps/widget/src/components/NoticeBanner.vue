<template>
  <div class="bg-secondary/30 border border-secondary rounded-md p-4" v-if="noticeText" role="status">
    <div class="flex items-center justify-center gap-3">
      <AlertTriangle class="flex-shrink-0 text-secondary-foreground" size="16" />
      <p class="text-sm text-secondary-foreground">{{ noticeText }}</p>
    </div>
  </div>
</template>

<script setup>
import { AlertTriangle } from 'lucide-vue-next'

defineProps({
  noticeText: {
    type: String,
    default: ''
  }
})
</script>
