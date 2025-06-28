<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <p>Add email inboxes by configuring IMAP and SMTP settings.</p>
      <p>Each added inbox creates a new email channel for receiving customer emails.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
