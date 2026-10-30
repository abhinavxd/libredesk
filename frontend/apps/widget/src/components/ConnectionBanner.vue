<script setup>
import { computed } from 'vue'
import { useI18n } from '@shared-ui/i18n'
import { useOnline } from '@vueuse/core'
import { storeToRefs } from 'pinia'
import { useWidgetStore } from '@widget/store/widget.js'
import BaseBanner from '@shared-ui/components/BaseBanner.vue'

const i18n = useI18n()
const isOnline = useOnline()
const { connectionFailed, connecting, connected } = storeToRefs(useWidgetStore())

const banner = computed(() => {
  if (!isOnline.value)
    return {
      text: i18n._('No internet connection'),
      colorClass: 'bg-warning text-warning-foreground'
    }
  if (connectionFailed.value)
    return {
      text: i18n._('Connection failed. Please refresh.'),
      colorClass: 'bg-destructive text-destructive-foreground'
    }
  if (connected.value)
    return {
      text: i18n._('Connected'),
      colorClass: 'bg-success text-success-foreground'
    }
  if (connecting.value)
    return {
      text: i18n._('Connecting...'),
      colorClass: 'bg-warning text-warning-foreground'
    }
  return null
})
</script>

<template>
  <Transition
    enter-active-class="transition-opacity duration-150 ease-out motion-reduce:duration-100"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition-opacity duration-100 ease-out"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <BaseBanner v-if="banner" :text="banner.text" :color-class="banner.colorClass" />
  </Transition>
</template>
