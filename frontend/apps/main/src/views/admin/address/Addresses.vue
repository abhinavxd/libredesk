<template>
  <AdminSplitLayout>
    <template #content><router-view /></template>
    <template #help><p>{{ $t('address.help') }}</p></template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
