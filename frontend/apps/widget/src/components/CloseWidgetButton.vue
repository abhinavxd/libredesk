<template>
  <Button
    v-if="widgetStore.isMobileFullScreen"
    @click="closeWidget"
    variant="ghost"
    aria-label="Close chat"
  >
    <X class="w-4 h-4" />
  </Button>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { X } from 'lucide-vue-next'
import { useWidgetStore } from '@widget/store/widget.js'

const widgetStore = useWidgetStore()

// Send message to parent window (widget.js) to close the widget
const closeWidget = () => {
  window.parent.postMessage(
    {
      type: 'CLOSE_WIDGET'
    },
    '*'
  )
}
</script>
