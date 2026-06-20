<script setup>
import { useOnline } from '@vueuse/core'
import { storeToRefs } from 'pinia'
import { useWidgetStore } from '@widget/store/widget.js'
import BaseBanner from './BaseBanner.vue'

const isOnline = useOnline()
const { connectionFailed, connecting, connected } = storeToRefs(useWidgetStore())
</script>

<template>
  <BaseBanner
    v-if="!isOnline"
    :text="$t('globals.messages.noInternetConnection')"
    color-class="bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-300"
  />
  <BaseBanner
    v-else-if="connectionFailed"
    :text="$t('globals.messages.connectionFailedRefresh')"
    color-class="bg-destructive text-destructive-foreground"
  />
  <BaseBanner
    v-else-if="connected"
    :text="$t('globals.messages.connected')"
    color-class="bg-green-100 text-green-900 dark:bg-green-950 dark:text-green-300"
  />
  <BaseBanner
    v-else-if="connecting"
    :text="$t('globals.messages.connecting')"
    color-class="bg-amber-100 text-amber-900 dark:bg-amber-950 dark:text-amber-300"
  />
</template>
