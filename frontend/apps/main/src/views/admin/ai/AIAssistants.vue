<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <div class="flex justify-end mb-4">
          <Button @click="router.push({ name: 'new-ai-assistant' })">{{
            t('admin.ai.assistant.new')
          }}</Button>
        </div>
        <DataTable
          :columns="createAssistantColumns(t, { onEdit: editAssistant })"
          :data="assistants"
          :loading="isLoading"
        />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ t('admin.ai.assistantsHelp') }}</p>
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
import { ref, onMounted, onUnmounted } from 'vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import { createAssistantColumns } from '@/features/admin/ai/assistantColumns.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()
const router = useRouter()
const isLoading = ref(false)

const assistants = ref([])

const refreshHandler = (data) => {
  if (data?.model === 'ai_assistants') getAssistants()
}

const editHandler = (data) => {
  if (data?.model === 'ai_assistants') editAssistant(data.data)
}

const editAssistant = (item) => {
  router.push({ name: 'edit-ai-assistant', params: { id: item.id } })
}

onMounted(() => {
  getAssistants()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

const getAssistants = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAIAssistants()
    assistants.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}
</script>
