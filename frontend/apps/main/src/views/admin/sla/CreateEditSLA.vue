<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <LoadingOverlay :loading="isLoading">
    <SLAForm
      :initial-values="slaData"
      :submitForm="submitForm"
      :isLoading="formLoading"
    />
  </LoadingOverlay>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '../../../api'
import SLAForm from '@/features/admin/sla/SLAForm.vue'
import { useRouter } from 'vue-router'
import LoadingOverlay from '@/components/layout/LoadingOverlay.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { useI18n } from '@shared-ui/i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const i18n = useI18n()
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
        description: i18n._('Changes saved')
      })
    } else {
      await api.createSLA(values)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: i18n._('Changes saved')
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
  return props.id ? i18n._('Edit') : i18n.pgettext('New SLA policy', 'New')
}

const breadcrumbLinks = [
  { path: 'sla-list', label: i18n._('{count, plural, one {SLA} other {SLAs}}') },
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
