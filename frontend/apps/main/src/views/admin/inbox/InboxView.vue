<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <div class="flex flex-col justify-evenly h-full gap-4">
        <div class="space-y-2">
          <p class="font-medium text-lg">{{ $t('globals.terms.email') }}</p>
          <p class="text-sm text-muted-foreground">{{ $t('admin.inbox.help.email') }}</p>
        </div>
        <div class="space-y-2">
          <p class="font-medium text-lg">{{ $t('globals.terms.liveChat') }}</p>
          <p class="text-sm text-muted-foreground">{{ $t('admin.inbox.help.livechat') }}</p>
        </div>
      </div>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
