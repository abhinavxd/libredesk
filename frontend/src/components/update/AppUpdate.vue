<template>
  <div
    v-if="appSettingsStore.settings['app.update']?.update?.is_new"
    class="p-2 mb-2 border-b bg-secondary text-secondary-foreground"
  >
    <div>
      {{ $t('update.newUpdateAvailable') }}:
      {{ appSettingsStore.settings['app.update'].update.release_version }} ({{
        appSettingsStore.settings['app.update'].update.release_date
      }})
      <a
        :href="appSettingsStore.settings['app.update'].update.url"
        target="_blank"
        nofollow
        noreferrer
        class="underline ml-2"
      >
        {{ $t('globals.messages.viewDetails') }}
      </a>
    </div>
    <div
      v-if="appSettingsStore.settings['app.update'].update.description"
      class="mt-1 text-xs opacity-90"
    >
      {{ appSettingsStore.settings['app.update'].update.description }}
    </div>
  </div>
</template>

<script setup>
import { useAppSettingsStore } from '@/stores/appSettings'
const appSettingsStore = useAppSettingsStore()
</script>
