<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading" />
  <SLAForm
    :initial-values="slaData"
    :submitForm="submitForm"
    :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
    :isLoading="formLoading"
  />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import SLAForm from '@/features/admin/sla/SLAForm.vue'
import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '@/utils/http'

const { t } = useI18n()
const slaData = ref({})
const emitter = useEmitter()
const isLoading = ref(false)
const formLoading = ref(false)
const router = useRouter()
const props = defineProps({
  id: {
    type: String,
    required: false
  }
})

const submitForm = async (values) => {
  try {
    formLoading.value = true
    if (props.id) {
      await api.updateSLA(props.id, values)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('globals.messages.updatedSuccessfully', {
          name: t('globals.terms.slaPolicy')
        })
      })
    } else {
      await api.createSLA(values)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('globals.messages.createdSuccessfully', {
          name: t('globals.terms.slaPolicy')
        })
      })
      router.push({ name: 'sla-list' })
    }
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
  { path: 'sla-list', label: t('globals.terms.sla') },
  { path: '', label: breadCrumLabel() }
]

onMounted(async () => {
  if (props.id) {
    try {
      isLoading.value = true
      const resp = await api.getSLA(props.id)
      slaData.value = resp.data.data
    } catch (error) {
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        variant: 'destructive',
        description: handleHTTPError(error).message
      })
    } finally {
      isLoading.value = false
    }
  }
})
</script>
