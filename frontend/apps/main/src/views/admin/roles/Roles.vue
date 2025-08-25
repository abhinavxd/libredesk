<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view></router-view>
    </template>

    <template #help>
      <p>Manage roles and their permissions for fine-grained control over your support desk.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
