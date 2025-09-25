<template>
  <div
    v-if="!isOnline"
    class="bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-300 px-4 py-2 text-center text-sm"
  >
    <div class="flex items-center justify-center gap-2">
      <div class="w-2 h-2 bg-amber-900 dark:bg-amber-300 rounded-full animate-pulse"></div>
      {{ $t('globals.messages.noInternetConnection') }}
    </div>
  </div>
</template>

<script setup>
import { useOnline } from '@vueuse/core'
const isOnline = useOnline()
</script>
