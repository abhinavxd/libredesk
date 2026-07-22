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
    color-class="bg-warning text-warning-foreground"
  />
  <BaseBanner
    v-else-if="connectionFailed"
    :text="$t('globals.messages.connectionFailedRefresh')"
    color-class="bg-destructive text-destructive-foreground"
  />
  <BaseBanner
    v-else-if="connected"
    :text="$t('globals.messages.connected')"
    color-class="bg-success text-success-foreground"
  />
  <BaseBanner
    v-else-if="connecting"
    :text="$t('globals.messages.connecting')"
    color-class="bg-warning text-warning-foreground"
  />
</template>
