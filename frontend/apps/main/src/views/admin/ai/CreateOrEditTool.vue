<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <LoadingOverlay :loading="isLoading">
    <ToolForm :initial-values="tool" :is-editing="!!id" :submit-form="submitForm" />
  </LoadingOverlay>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'
import ToolForm from '@/features/admin/ai/ToolForm.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  id: { type: String, required: false }
})

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()
const tool = ref({})
const isLoading = ref(false)

const breadcrumbLinks = [
  { path: 'ai-tools', label: t('admin.ai.tools') },
  { path: '', label: props.id ? t('admin.ai.tool.edit') : t('admin.ai.tool.new') }
]

const submitForm = async (values) => {
  try {
    if (props.id) {
      await api.updateAITool(props.id, values)
    } else {
      await api.createAITool(values)
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    router.push({ name: 'ai-tools' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

onMounted(async () => {
  if (!props.id) return
  try {
    isLoading.value = true
    const resp = await api.getAITool(props.id)
    tool.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})
</script>
