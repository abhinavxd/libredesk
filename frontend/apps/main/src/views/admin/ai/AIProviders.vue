<template>
  <AdminSplitLayout>
    <template #content>
      <div class="space-y-6">
        <ProviderForm
          type="completion"
          :title="t('admin.ai.completion')"
          :description="t('admin.ai.completionDescription')"
          :show-completion-fields="true"
        />
        <ProviderForm
          type="embedding"
          :title="t('admin.ai.embedding')"
          :description="t('admin.ai.embeddingDescription')"
          :show-embedding-fields="true"
        />
      </div>
    </template>

    <template #help>
      <p>{{ t('admin.ai.providersHelp') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/ai"
        target="_blank"
        rel="noopener noreferrer"
        class="link-style"
      >
        {{ t('globals.terms.learnMore') }}
      </a>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import ProviderForm from '@/features/admin/ai/ProviderForm.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
</script>
