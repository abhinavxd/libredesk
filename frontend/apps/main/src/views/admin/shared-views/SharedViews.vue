<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view></router-view>
    </template>
    <template #help>
      <p>Create shared views visible to all agents or specific teams.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
