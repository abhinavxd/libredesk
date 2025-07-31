<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <AIAssistantForm :submitForm="onSubmit" :initialValues="{}" :isNewForm="true" :isLoading="formLoading" />
</template>

<script setup>
import { ref } from 'vue'
import AIAssistantForm from '@/features/admin/ai-assistants/AIAssistantForm.vue'
import { handleHTTPError } from '../../../utils/http'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { useRouter } from 'vue-router'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '../../../api'

const { t } = useI18n()
const emitter = useEmitter()
const router = useRouter()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: 'ai-assistant-list', label: t('globals.terms.aiAssistant', 2) },
  {
    path: '',
    label: t('globals.messages.new', {
      name: t('globals.terms.aiAssistant', 1).toLowerCase()
    })
  }
]

const onSubmit = (values) => {
  createNewAIAssistant(values)
}

const createNewAIAssistant = async (values) => {
  try {
    formLoading.value = true
    await api.createAIAssistant(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', {
        name: t('globals.terms.aiAssistant', 1)
      })
    })
    router.push({ name: 'ai-assistant-list' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>