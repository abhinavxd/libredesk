<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <CustomAnswerForm @submit="onSubmit" @cancel="onCancel" :formLoading="formLoading" />
</template>

<script setup>
import { ref } from 'vue'
import CustomAnswerForm from '@/features/admin/custom-answers/CustomAnswerForm.vue'
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
  { path: 'custom-answer-list', label: t('globals.terms.customAnswer', 2) },
  {
    path: '',
    label: t('globals.messages.new', {
      name: t('globals.terms.customAnswer', 1).toLowerCase()
    })
  }
]

const onSubmit = (values) => {
  createNewCustomAnswer(values)
}

const onCancel = () => {
  router.push({ name: 'custom-answer-list' })
}

const createNewCustomAnswer = async (values) => {
  try {
    formLoading.value = true
    await api.createAICustomAnswer(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', {
        name: t('globals.terms.customAnswer', 1)
      })
    })
    router.push({ name: 'custom-answer-list' })
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