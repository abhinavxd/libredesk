<template>
  <AdminSplitLayout>
    <template #content>
      <router-view></router-view>
    </template>

    <template #help>
      <p>{{ i18n._('Manage support agents, roles, permissions and teams.') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
