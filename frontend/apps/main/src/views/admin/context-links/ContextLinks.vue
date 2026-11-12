<template>
  <AdminSplitLayout>
    <template #content>
      <router-view />
    </template>

    <template #help>
      <p>{{ i18n._('Context links appear in the conversation sidebar, letting agents open external tools like CRMs, billing systems, or internal dashboards with the contact\'\'s details passed automatically.') }}</p>
      <p>{{ i18n._(literalExampleMessage1) }}</p>
      <a
        href="https://docs.libredesk.io/configuration/context-links"
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
import { defineMessage as msg } from '@lingui/core/macro'

const literalExampleMessage1 = msg({ message: 'Use \'{{token}}\' for a fully encrypted payload that the external app decrypts with a shared secret. Or use individual variables like \'{{email}}\', \'{{phone}}\', \'{{external_user_id}}\' for plain URL links to trusted internal tools.' })
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
</script>
