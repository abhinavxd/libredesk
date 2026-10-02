<script setup>
import { DEFAULT_SITE_TITLE } from '@main/stores/appSettings'

// logo is the admin's site logo; blank falls back to the Fernmail fern.
defineProps({
  name: { type: String, default: DEFAULT_SITE_TITLE },
  logo: { type: String, default: '' }
})
</script>

<template>
  <span class="inline-flex min-w-0 max-w-full items-center gap-2 font-semibold tracking-tight">
    <img
      v-if="logo"
      :src="logo"
      alt=""
      class="h-7 w-auto max-w-[8rem] shrink-0 object-contain"
    />
    <img v-else src="/images/fern.svg" alt="" class="h-7 w-7 shrink-0" />
    <span class="truncate">{{ name }}</span>
  </span>
</template>
