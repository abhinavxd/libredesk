<template>
  <div
    v-if="banner"
    role="status"
    aria-live="polite"
    aria-atomic="true"
    class="fixed top-3 left-1/2 -translate-x-1/2 z-50 flex items-center gap-2 rounded-full px-4 py-1.5 text-sm shadow-md"
    :class="banner.colorClass"
  >
    <div class="w-2 h-2 rounded-full bg-current animate-pulse"></div>
    {{ banner.text }}
  </div>
</template>

<script setup>
import { ref, computed, watch, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useOnline } from '@vueuse/core'
import { storeToRefs } from 'pinia'
import { useConnectionStore } from '@/stores/connection'

const RECONNECTING_DELAY_MS = 1500

const { t } = useI18n()
const isOnline = useOnline()
const { connecting, connectionFailed } = storeToRefs(useConnectionStore())
const showReconnecting = ref(false)
let reconnectingTimer = null

const banner = computed(() => {
  if (!isOnline.value)
    return {
      text: t('globals.messages.noInternetConnection'),
      colorClass: 'bg-warning text-warning-foreground'
    }
  if (connectionFailed.value)
    return {
      text: t('globals.messages.connectionFailedRefresh'),
      colorClass: 'bg-destructive text-destructive-foreground'
    }
  if (showReconnecting.value)
    return {
      text: t('globals.messages.connecting'),
      colorClass: 'bg-warning text-warning-foreground'
    }
  return null
})

// Most drops recover on the first retry, so hold the banner back instead of flashing it.
watch(connecting, (value) => {
  clearTimeout(reconnectingTimer)
  if (value) {
    reconnectingTimer = setTimeout(() => (showReconnecting.value = true), RECONNECTING_DELAY_MS)
  } else {
    showReconnecting.value = false
  }
})

onUnmounted(() => clearTimeout(reconnectingTimer))
</script>
