<template>
  <TooltipProvider :delay-duration="150">
    <Toaster class="pointer-events-auto" position="top-center" richColors />
    <RouterView />
  </TooltipProvider>
</template>

<script setup>
import { RouterView } from 'vue-router'
import { Toaster } from '@shared-ui/components/ui/sonner'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
</script>
