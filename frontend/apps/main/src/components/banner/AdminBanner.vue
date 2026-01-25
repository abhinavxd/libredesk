<template>
  <div class="border-b">
    <!-- Update notification -->
    <div
      v-if="appSettingsStore.settings['app.update']?.update?.is_new"
      class="px-4 py-2.5 border-b border-border/50 last:border-b-0"
    >
      <div class="flex items-center gap-3">
        <div class="flex-shrink-0">
          <Download class="w-5 h-5 text-primary" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="flex items-center gap-2 text-sm text-foreground">
            <span>{{ $t('update.newUpdateAvailable') }}</span>
            <a
              :href="appSettingsStore.settings['app.update'].update.url"
              target="_blank"
              rel="nofollow noreferrer"
              class="font-semibold text-primary hover:text-primary/80 underline transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-1"
            >
              {{ appSettingsStore.settings['app.update'].update.release_version }}
            </a>
            <span class="text-muted-foreground">•</span>
            <span class="text-muted-foreground">
              {{ appSettingsStore.settings['app.update'].update.release_date }}
            </span>
          </div>

          <!-- Update description -->
          <div
            v-if="appSettingsStore.settings['app.update'].update.description"
            class="mt-2 text-xs text-muted-foreground"
          >
            {{ appSettingsStore.settings['app.update'].update.description }}
          </div>
        </div>
      </div>
    </div>

    <!-- Restart required notification -->
    <div
      v-if="appSettingsStore.settings['app.restart_required']"
      class="px-4 py-2.5 border-b border-border/50 last:border-b-0"
    >
      <div class="flex items-center gap-3">
        <div class="flex-shrink-0">
          <Info class="w-5 h-5 text-primary" />
        </div>
        <div class="min-w-0 flex-1">
          <div class="text-sm text-foreground">
            {{ $t('admin.banner.restartMessage') }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { Download, Info } from 'lucide-vue-next'
import { useAppSettingsStore } from '@/stores/appSettings'
const appSettingsStore = useAppSettingsStore()
</script>
