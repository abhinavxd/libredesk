<template>
  <div class="flex items-center p-2 border-b border-border bg-background gap-3 relative">
    <div class="flex items-center gap-2 justify-start">
      <Button @click="$emit('goBack')" variant="ghost" size="sm">
        <ArrowLeft />
      </Button>
      <ChatTitle />
    </div>
    <!-- Close Button - only visible on mobile (full screen mode) -->
    <CloseWidgetButton class="absolute right-4" />
  </div>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { ArrowLeft } from 'lucide-vue-next'
import ChatTitle from './ChatTitle.vue'
import CloseWidgetButton from './CloseWidgetButton.vue'

defineEmits(['goBack'])
</script>
