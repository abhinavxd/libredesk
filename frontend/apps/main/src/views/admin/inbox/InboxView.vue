<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <div class="flex flex-col justify-evenly h-full gap-4">
        <p class="font-medium text-2xl">{{ $t('admin.inbox.help.title') }}</p>
        <p>{{ $t('admin.inbox.help.description') }}</p>
        <ul class="list-disc list-inside space-y-2">
          <li>{{ $t('admin.inbox.help.email') }}</li>
          <li>{{ $t('admin.inbox.help.livechat') }}</li>
        </ul>
      </div>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
