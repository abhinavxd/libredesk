<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <LoadingOverlay :loading="isLoading">
    <OIDCForm
      :initial-values="oidc"
      :submitForm="submitForm"
      :isNewForm="isNewForm"
      :isLoading="formLoading"
    />
  </LoadingOverlay>
</template>

<script setup>
import { onMounted, ref, computed } from 'vue'
import api from '../../../api'
import OIDCForm from '@/features/admin/oidc/OIDCForm.vue'
import LoadingOverlay from '@/components/layout/LoadingOverlay.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'
import { useRouter } from 'vue-router'

const router = useRouter()
const i18n = useI18n()
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
    formLoading.value = true
    let toastDescription = ''
    if (props.id) {
      await api.updateOIDC(props.id, values)
      toastDescription = i18n._('Changes saved')
    } else {
      await api.createOIDC(values)
      router.push({ name: 'sso-list' })
      toastDescription = i18n._('Changes saved')
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
  return props.id ? i18n._('Edit') : i18n.pgettext('New SSO provider', 'New')
}

const isNewForm = computed(() => {
  return props.id ? false : true
})

const breadcrumbLinks = [
  { path: 'sso-list', label: i18n._('{count, plural, one {SSO} other {SSOs}}') },
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
