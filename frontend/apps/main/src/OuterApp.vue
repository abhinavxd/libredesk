<template>
  <RouterView />
</template>

<script setup>
import { onMounted, onUnmounted } from 'vue'
import { RouterView } from 'vue-router'
import { EMITTER_EVENTS } from './constants/emitterEvents.js'
import { useEmitter } from './composables/useEmitter'
import { toast as sooner } from 'vue-sonner'

const emitter = useEmitter()

const toastHandler = (message) => {
  if (!message.description) return
  if (message.variant === 'destructive') {
    sooner.error(message.description)
  } else {
    sooner.success(message.description)
  }
}

onMounted(() => {
  emitter.on(EMITTER_EVENTS.SHOW_TOAST, toastHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.SHOW_TOAST, toastHandler)
})
</script>
