<template>
  <AdminPageWithHelp>
    <template #content>
      <router-view />
    </template>
    
    <template #help>
      <p>
        Business Hours allows you to set working hours for your entire helpdesk or for individual
        teams.
      </p>
      <p>
        SLA calculations are based on business hours. If a team has business hours set, the SLA will
        be calculated using that team's hours. Otherwise, it will fall back to the helpdesk's
        business hours.
      </p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
</script>