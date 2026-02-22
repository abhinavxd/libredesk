<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>
    
    <template #help>
      <p>{{ $t('admin.businessHour.help.description') }}</p>
      <p>{{ $t('admin.businessHour.help.detail') }}</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>