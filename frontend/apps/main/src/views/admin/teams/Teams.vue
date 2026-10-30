<template>
  <AdminSplitLayout>
    <template #content>
      <router-view></router-view>
    </template>

    <template #help>
      <p>{{ i18n._('Configure team settings including working hours and SLA policies.') }}</p>
      <p>{{ i18n._('Manage agent auto-assignment limits and more.') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
