<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <p>
        Configure SLA policies to set response, resolution and next response time targets.
      </p>
      <p>
        SLAs help track team performance and ensure conversations are handled within expected timeframes.
        Breached SLAs trigger notifications to configured team members.
      </p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
