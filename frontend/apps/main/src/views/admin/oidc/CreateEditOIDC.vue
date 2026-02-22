<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading" />
  <OIDCForm
    :initial-values="oidc"
    :submitForm="submitForm"
    :isNewForm="isNewForm"
    :class="{ 'opacity-50 transition-opacity duration-300': isLoading }"
    :isLoading="formLoading"
  />
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import api from '../../../api'
import OIDCForm from '@/features/admin/oidc/OIDCForm.vue'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const router = useRouter()
const { t } = useI18n()
const oidc = ref({
  provider: 'Google'
})
const emitter = useEmitter()
const isLoading = ref(false)
const formLoading = ref(false)
const props = defineProps({
  id: {
    type: String,
    required: false
  }
})

const submitForm = async (values) => {
  try {
    let toastDescription = ''
    if (props.id) {
      if (values.client_secret.includes('•')) {
        values.client_secret = ''
      }
      await api.updateOIDC(props.id, values)
      toastDescription = t('globals.messages.savedSuccessfully')
    } else {
      await api.createOIDC(values)
      router.push({ name: 'sso-list' })
      toastDescription = t('globals.messages.savedSuccessfully')
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Success',
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

const isNewForm = computed(() => {
  return props.id ? false : true
})

const breadcrumbLinks = [
  { path: 'sso-list', label: t('globals.terms.sso') },
  { path: '', label: breadCrumLabel() }
]

onMounted(async () => {
  if (props.id) {
    try {
      isLoading.value = true
      const resp = await api.getOIDC(props.id)
      oidc.value = resp.data.data
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
