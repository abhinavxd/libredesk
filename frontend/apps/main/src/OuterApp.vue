<template>
  <RouterView />
</template>

<script setup>
import { onMounted } from 'vue'
import { RouterView } from 'vue-router'
import { EMITTER_EVENTS } from './constants/emitterEvents.js'
import { useEmitter } from './composables/useEmitter'
import { toast as sooner } from 'vue-sonner'

const emitter = useEmitter()

onMounted(() => {
  initToaster()
})

const initToaster = () => {
  emitter.on(EMITTER_EVENTS.SHOW_TOAST, (message) => {
    if (message.variant === 'destructive') {
      sooner.error(message.description)
    } else {
      sooner.success(message.description)
    }
  })
}
</script>
