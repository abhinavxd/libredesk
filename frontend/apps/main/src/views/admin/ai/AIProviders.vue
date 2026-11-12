<template>
  <AdminSplitLayout>
    <template #content>
      <div class="space-y-6">
        <ProviderForm
          type="completion"
          :title="i18n._('Completion')"
          :description="i18n._('The model used to draft replies and power copilot.')"
          :show-completion-fields="true"
        />
        <ProviderForm
          type="embedding"
          :title="i18n._('Embedding')"
          :description="i18n._('The model used to embed articles and snippets for search.')"
          :show-embedding-fields="true"
        />
      </div>
    </template>

    <template #help>
      <p>{{ i18n._('Works with any OpenAI-compatible endpoint.') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/ai#providers"
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
import ProviderForm from '@/features/admin/ai/ProviderForm.vue'
import { useI18n } from '@shared-ui/i18n'

const i18n = useI18n()
</script>
