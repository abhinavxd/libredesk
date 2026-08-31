<template>
  <AdminSplitLayout>
    <template #content>
      <router-view></router-view>
    </template>

    <template #help>
      <p>{{ $t('admin.role.help') }}</p>
      <a
        href="https://docs.libredesk.io/roles/overview"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        {{ $t('globals.terms.learnMore') }}
      </a>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
