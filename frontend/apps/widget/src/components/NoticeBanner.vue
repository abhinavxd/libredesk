<template>
  <div class="border rounded-lg p-4" v-if="noticeText">
    <div class="flex items-start gap-3">
      <AlertTriangle class="flex-shrink-0 mt-1" size="16" />
      <p class="text-sm">{{ noticeText }}</p>
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
