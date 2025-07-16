<template>
  <div class="flex items-center p-2 border-b border-border bg-background gap-3 relative">
    <div class="flex items-center gap-2 justify-start">
      <Button @click="$emit('goBack')" variant="ghost" size="sm">
        <ArrowLeft />
      </Button>
      <ChatTitle />
    </div>
    <div class="flex items-center gap-2 ml-auto">
      <!-- Expand/Collapse Button - only visible on desktop -->
      <Button 
        v-if="!widgetStore.isMobileFullScreen" 
        @click="widgetStore.toggleExpand" 
        variant="ghost" 
        size="sm"
        :title="widgetStore.isExpanded ? 'Collapse' : 'Expand'"
      >
        <Maximize2 v-if="!widgetStore.isExpanded" class="w-4 h-4" />
        <Minimize2 v-else class="w-4 h-4" />
      </Button>
      <!-- Close Button - only visible on mobile (full screen mode) -->
      <CloseWidgetButton />
    </div>
  </div>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { ArrowLeft, Maximize2, Minimize2 } from 'lucide-vue-next'
import ChatTitle from './ChatTitle.vue'
import CloseWidgetButton from './CloseWidgetButton.vue'
import { useWidgetStore } from '@widget/store/widget.js'

const widgetStore = useWidgetStore()

defineEmits(['goBack'])
</script>
