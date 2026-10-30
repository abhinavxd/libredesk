<template>
  <AdminSplitLayout>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <p>{{ i18n._('Configure single sign-on with one or more OpenID Connect providers.') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/sso"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        {{ i18n._('Learn more') }}
      </a>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
