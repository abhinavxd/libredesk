<template>
  <AdminSplitLayout>
    <template #content>
      <router-view></router-view>
    </template>
    <template #help>
      <p>{{ i18n._('Create shared views visible to all agents or specific teams.') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
