<template>
  <div class="flex h-14 shrink-0 items-center gap-3 border-b border-border px-3">
    <SidebarTrigger class="cursor-pointer" />
    <h1 class="ui-heading min-w-0 truncate">{{ i18n.pgettext('Search page title', 'Search') }}</h1>
  </div>
</template>

<script setup>
import { SidebarTrigger } from '@shared-ui/components/ui/sidebar'
import { useI18n } from '@shared-ui/i18n'

const i18n = useI18n()
</script>
