<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <div class="space-y-4">
        <div>
          <h3 class="font-semibold text-sm mb-2 text-foreground">{{ $t('ai.snippet.helpTitle') }}</h3>
          <p class="text-sm text-muted-foreground mb-3">{{ $t('ai.snippet.helpDescription') }}</p>
        </div>

        <div>
          <h4 class="font-medium text-sm mb-2 text-foreground">{{ $t('ai.snippet.helpHowItWorks') }}</h4>
          <ul class="text-sm text-muted-foreground space-y-1 list-disc list-inside">
            <li>{{ $t('ai.snippet.helpStep1') }}</li>
            <li>{{ $t('ai.snippet.helpStep2') }}</li>
          </ul>
        </div>

        <div>
          <h4 class="font-medium text-sm mb-2 text-foreground">{{ $t('ai.snippet.helpBestPractices') }}</h4>
          <ul class="text-sm text-muted-foreground space-y-1 list-disc list-inside">
            <li>{{ $t('ai.snippet.helpTip1') }}</li>
          </ul>
        </div>
      </div>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>