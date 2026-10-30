<template>
  <AdminSplitLayout>
    <template #content>
      <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
      </div>
      <LoadingOverlay :loading="isLoading">
        <ToolForm :initial-values="tool" :is-editing="!!id" :submit-form="submitForm" />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ i18n._('Tools let the assistant call your own APIs to fetch or update data while replying. A tool is shared across every assistant granted access to it, and the model decides when to call it from the name and description, so keep those precise.') }}</p>
      <a
        href="https://docs.libredesk.io/configuration/ai#tools"
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
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'
import ToolForm from '@/features/admin/ai/ToolForm.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'

const props = defineProps({
  id: { type: String, required: false }
})

const i18n = useI18n()
const router = useRouter()
const emitter = useEmitter()
const tool = ref({})
const isLoading = ref(false)

const breadcrumbLinks = [
  { path: 'ai-tools', label: i18n._('Tools') },
  { path: '', label: props.id ? i18n._('Edit tool') : i18n._('New tool') }
]

const submitForm = async (values) => {
  try {
    if (props.id) {
      await api.updateAITool(props.id, values)
    } else {
      await api.createAITool(values)
      router.push({ name: 'ai-tools' })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
    })
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
