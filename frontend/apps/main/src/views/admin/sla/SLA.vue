<template>
  <AdminSplitLayout>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <p>{{ i18n._('Configure SLA policies to set response, resolution and next response time targets.') }}</p>
      <p>{{ i18n._('SLAs help track team performance and ensure conversations are handled within expected timeframes. Breached SLAs trigger notifications to configured team members.') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
