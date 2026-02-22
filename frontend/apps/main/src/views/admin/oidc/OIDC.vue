<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <p>{{ $t('admin.oidc.help') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/sso"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        <p>{{ $t('globals.messages.learnMore') }}</p>
      </a>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>
