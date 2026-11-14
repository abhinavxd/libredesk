<template>
  <div class="flex h-14 shrink-0 items-center gap-3 border-b border-border px-3">
    <SidebarTrigger class="cursor-pointer" />
    <h1 class="ui-heading min-w-0 truncate">{{ t('globals.terms.search') }}</h1>
  </div>
</template>

<script setup>
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>
