<template>
  <AdminSplitLayout>
    <template #content>
      <router-view></router-view>
    </template>
    <template #help>
      <p>{{ i18n._('Combine multiple conversation actions into single-click macros.') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
