<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <p>Configure webhooks to receive real-time notifications when events occur in your Libredesk workspace.</p>
      <p>Webhooks allow you to integrate Libredesk with external services by sending HTTP POST requests when specific events happen.</p>
      <a
        href="https://libredesk.io/docs/webhooks/"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        <p>Learn more</p>
      </a>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
