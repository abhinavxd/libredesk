<template>
  <AdminSplitLayout>
    <template #content>
      <router-view />
    </template>
    
    <template #help>
      <p>{{ i18n._('Business Hours allows you to set working hours for your entire helpdesk or for individual teams.') }}</p>
      <p>{{ i18n._('SLA calculations are based on business hours. If a team has business hours set, the SLA will be calculated using that team\'\'s hours. Otherwise, it will fall back to the helpdesk\'\'s business hours.') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
