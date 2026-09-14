<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading" reserve-height>
        <div class="flex justify-end mb-4">
          <Dialog v-model:open="dialogOpen">
            <DialogTrigger as-child @click="newPrompt">
              <Button>{{ t('admin.ai.editorPrompt.new') }}</Button>
            </DialogTrigger>
            <DialogContent class="sm:max-w-2xl">
              <DialogHeader>
                <DialogTitle>
                  {{ isEditing ? t('admin.ai.editorPrompt.edit') : t('admin.ai.editorPrompt.new') }}
                </DialogTitle>
              </DialogHeader>
              <EditorPromptForm
                :initial-values="initialValues"
                :is-editing="isEditing"
                :submit-form="submitPrompt"
              />
            </DialogContent>
          </Dialog>
        </div>
        <DataTable
          :columns="createEditorPromptColumns(t, { onEdit: editPrompt })"
          :data="prompts"
          :loading="isLoading"
        />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ t('admin.ai.editorPromptsHelp') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { Button } from '@shared-ui/components/ui/button/index.js'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog/index.js'
import EditorPromptForm from '@/features/admin/ai/EditorPromptForm.vue'
import { createEditorPromptColumns } from '@/features/admin/ai/editorPromptColumns.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()
const isLoading = ref(false)
const prompts = ref([])
const dialogOpen = ref(false)
const isEditing = ref(false)
const initialValues = ref({})
const editingId = ref(null)

const refreshHandler = (data) => {
  if (data?.model === 'ai_editor_prompts') getPrompts()
}

const editHandler = (data) => {
  if (data?.model === 'ai_editor_prompts') editPrompt(data.data)
}

onMounted(() => {
  getPrompts()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

const getPrompts = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAiPrompts()
    prompts.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const newPrompt = () => {
  isEditing.value = false
  editingId.value = null
  initialValues.value = {}
}

const editPrompt = async (item) => {
  try {
    const resp = await api.getAIPrompt(item.id)
    isEditing.value = true
    editingId.value = item.id
    initialValues.value = resp.data.data
    dialogOpen.value = true
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const submitPrompt = async (values) => {
  try {
    if (isEditing.value) {
      await api.updateAIPrompt(editingId.value, values)
    } else {
      await api.createAIPrompt(values)
    }
    dialogOpen.value = false
    getPrompts()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
