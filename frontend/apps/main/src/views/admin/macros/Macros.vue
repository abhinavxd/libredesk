<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view></router-view>
    </template>
    <template #help>
      <p>{{ $t('admin.macro.help') }}</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
