<template>
  <div class="flex items-center space-x-4 px-2 h-12 border-b shrink-0">
    <SidebarTrigger class="cursor-pointer" />
    <span class="text-xl font-semibold">{{ t('globals.terms.search') }}</span>
  </div>
</template>

<script setup>
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>
