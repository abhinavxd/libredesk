<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading" />
  <TemplateForm
    :initial-values="template"
    :submitForm="submitForm"
    :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
    :isLoading="formLoading"
  />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '../../../api'
import TemplateForm from '@/features/admin/templates/TemplateForm.vue'
import { useRouter, useRoute } from 'vue-router'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import { useEmitter } from '../../../composables/useEmitter'

const template = ref({})
const { t } = useI18n()
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()
const router = useRouter()
const route = useRoute()

const props = defineProps({
  id: {
    type: String,
    required: false,
    default: null
  }
})

const submitForm = async (values) => {
  try {
    formLoading.value = true
    let toastDescription = ''
    if (props.id) {
      await api.updateTemplate(props.id, values)
      toastDescription = t('globals.messages.savedSuccessfully')
    } else {
      await api.createTemplate(values)
      toastDescription = t('globals.messages.savedSuccessfully')
      router.push({ name: 'templates' })
      emitter.emit(EMITTER_EVENTS.REFRESH_LIST, {
        model: 'templates'
      })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: toastDescription
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}

const breadCrumLabel = () => {
  return props.id ? t('globals.messages.edit') : t('globals.messages.new')
}

const breadcrumbLinks = [
  { path: 'templates', label: t('globals.terms.template') },
  { path: '', label: breadCrumLabel() }
]

onMounted(async () => {
  if (props.id) {
    try {
      isLoading.value = true
      const resp = await api.getTemplate(props.id)
      template.value = resp.data.data
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      isLoading.value = false
    }
  } else {
    template.value = {
      type: route.query.type
    }
  }
})
</script>
