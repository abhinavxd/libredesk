<template>
  <div class="border-b">
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
import { Info } from 'lucide-vue-next'
</script>
