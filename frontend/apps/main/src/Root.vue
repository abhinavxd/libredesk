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
import { watch } from 'vue'
import { useColorMode } from '@vueuse/core'
import { RouterView } from 'vue-router'
import { Toaster } from '@shared-ui/components/ui/sonner'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import { useAppSettingsStore } from '@main/stores/appSettings'
import { setFavicon } from '@main/utils/favicon'

const mode = useColorMode()
const settingsStore = useAppSettingsStore()

// The site logo doubles as the favicon.
watch(() => settingsStore.siteLogo, (logo) => setFavicon(logo), { immediate: true })
</script>
