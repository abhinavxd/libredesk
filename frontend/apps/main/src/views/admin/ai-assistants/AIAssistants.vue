<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view></router-view>
    </template>

    <template #help>
      <p>Manage your AI assistants that can automatically respond to customer inquiries. Configure their product knowledge, response style, and system prompts to provide accurate and helpful support.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>