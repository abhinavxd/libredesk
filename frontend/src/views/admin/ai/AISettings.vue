<template>
  <AdminPageWithHelp>
    <template #content>
      <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
        <Spinner v-if="isLoading" />
        <AISettingsForm :initial-values="initialValues" :submit-form="submitForm" />
      </div>
    </template>

    <template #help>
      <p>{{ $t('admin.ai.help') }}</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import api from '@/api'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { useI18n } from 'vue-i18n'
import AISettingsForm from '@/features/admin/ai/AISettingsForm.vue'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { Spinner } from '@/components/ui/spinner'

const initialValues = ref({})
const { t } = useI18n()
const isLoading = ref(false)
const emitter = useEmitter()

onMounted(() => {
  fetchProvider()
})

const fetchProvider = async () => {
  try {
    isLoading.value = true
    const resp = await api.getAIProvider()
    const data = resp.data.data
    initialValues.value = {
      endpoint_url: data.endpoint_url || '',
      model: data.model || '',
      api_key: '',
      has_api_key: data.has_api_key || false
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const submitForm = async (values) => {
  try {
    await api.updateAIProvider({
      provider: 'openai',
      endpoint_url: values.endpoint_url || '',
      model: values.model || '',
      api_key: values.api_key || ''
    })
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully', {
        name: t('globals.terms.provider')
      })
    })
    await fetchProvider()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
</script>
