<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view></router-view>
    </template>

    <template #help>
      <p>Manage your support agents, their roles and permissions and teams.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
