<template>
  <Button
    v-if="widgetStore.isMobileFullScreen"
    @click="closeWidget"
    variant="ghost"
    size="sm"
    :aria-label="$t('globals.messages.closeChat')"
  >
    <X class="w-4 h-4" />
  </Button>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { X } from 'lucide-vue-next'
import { useWidgetStore } from '@widget/store/widget.js'

const widgetStore = useWidgetStore()
const closeWidget = () => widgetStore.closeWidget()
</script>
