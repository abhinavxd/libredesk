<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <SharedViewForm
      :initialValues="sharedView"
      :submitForm="submitForm"
      :isLoading="formLoading"
      v-if="!isLoading"
    />
  </LoadingOverlay>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import SharedViewForm from '@/features/admin/shared-views/SharedViewForm.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { useI18n } from '@shared-ui/i18n'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useSharedViewStore } from '@/stores/sharedView'

const sharedView = ref({})
const i18n = useI18n()
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()
const sharedViewStore = useSharedViewStore()

const breadcrumbLinks = [
  { path: 'shared-view-list', label: i18n._('{count, plural, one {Shared view} other {Shared views}}', 2) },
  { path: '', label: i18n._('Edit shared view') }
]

const submitForm = (values) => {
  updateSharedView(values)
}

const updateSharedView = async (payload) => {
  try {
    formLoading.value = true
    await api.updateSharedView(sharedView.value.id, payload)

    // Reload shared views from server
    await sharedViewStore.refresh()

    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
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

onMounted(async () => {
  try {
    isLoading.value = true
    const resp = await api.getSharedView(props.id)
    sharedView.value = resp.data.data
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
