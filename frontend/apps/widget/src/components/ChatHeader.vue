<template>
  <div class="flex items-center p-2 border-b border-border bg-background gap-3 relative">
    <div class="flex items-center gap-2 justify-start">
      <Button @click="$emit('goBack')" variant="ghost" size="sm" :aria-label="$t('globals.messages.goBack')">
        <ArrowLeft v-if="!widgetStore.isRtl" class="w-4 h-4" />
        <ArrowRight v-else class="w-4 h-4" />
      </Button>
      <ChatTitle />
    </div>
    <div class="flex items-center gap-2 ml-auto rtl:ml-0 rtl:mr-auto">
      <Button 
        v-if="!widgetStore.isMobileFullScreen" 
        @click="widgetStore.toggleExpand" 
        variant="ghost" 
        size="sm"
        :title="widgetStore.isExpanded ? $t('globals.terms.collapse') : $t('globals.terms.expand')"
        :aria-label="widgetStore.isExpanded ? $t('globals.terms.collapse') : $t('globals.terms.expand')"
      >
        <Maximize2 v-if="!widgetStore.isExpanded" class="w-4 h-4" />
        <Minimize2 v-else class="w-4 h-4" />
      </Button>
    </div>
  </div>
</template>

<script setup>
import { Button } from '@shared-ui/components/ui/button'
import { ArrowLeft, ArrowRight, Maximize2, Minimize2 } from 'lucide-vue-next'
import ChatTitle from './ChatTitle.vue'
import { useWidgetStore } from '@widget/store/widget.js'

const widgetStore = useWidgetStore()

defineEmits(['goBack'])
</script>
