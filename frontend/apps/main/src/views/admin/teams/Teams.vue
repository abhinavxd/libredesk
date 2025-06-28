<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view></router-view>
    </template>

    <template #help>
      <p>Configure team settings including working hours and SLA policies.</p>
      <p>Manage agent auto-assignment limits and more.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
