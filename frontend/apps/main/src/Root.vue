<template>
  <TooltipProvider :delay-duration="150">
    <Toaster
      class="pointer-events-auto"
      position="top-center"
      :theme="mode === 'auto' ? 'system' : mode"
    />
    <RouterView />
  </TooltipProvider>
</template>

<script setup>
import { useColorMode } from '@vueuse/core'
import { RouterView } from 'vue-router'
import { Toaster } from '@shared-ui/components/ui/sonner'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'

const mode = useColorMode()
</script>
