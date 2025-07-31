<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"/>
  <CustomAnswerForm 
    v-else
    :customAnswer="customAnswer" 
    @submit="onSubmit" 
    @cancel="onCancel"
    :formLoading="formLoading" 
  />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '../../../api'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '../../../utils/http'
import CustomAnswerForm from '@/features/admin/custom-answers/CustomAnswerForm.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const customAnswer = ref({})
const { t } = useI18n()
const router = useRouter()
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()

const breadcrumbLinks = [
  { path: 'custom-answer-list', label: t('globals.terms.customAnswer', 2) },
  {
    path: '',
    label: t('globals.messages.edit', {
      name: t('globals.terms.customAnswer', 1).toLowerCase()
    })
  }
]

const onSubmit = (values) => {
  updateCustomAnswer(values)
}

const onCancel = () => {
  router.push({ name: 'custom-answer-list' })
}

const updateCustomAnswer = async (payload) => {
  try {
    formLoading.value = true
    await api.updateAICustomAnswer(customAnswer.value.id, payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
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

onMounted(async () => {
  try {
    isLoading.value = true
    const resp = await api.getAICustomAnswer(props.id)
    customAnswer.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})
</script>