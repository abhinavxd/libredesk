<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <p>Configure single sign-on with one or more OpenID Connect providers.</p>
      <a
        href="https://libredesk.io/docs/sso/"
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
