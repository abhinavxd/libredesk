<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading" />
  <SharedViewForm
    :initialValues="sharedView"
    :submitForm="submitForm"
    :isLoading="formLoading"
    v-else
  />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import SharedViewForm from '@/features/admin/shared-views/SharedViewForm.vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { useI18n } from 'vue-i18n'
import { Spinner } from '@/components/ui/spinner'
import { useSharedViewStore } from '@/stores/sharedView'

const sharedView = ref({})
const { t } = useI18n()
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()
const sharedViewStore = useSharedViewStore()

const breadcrumbLinks = [
  { path: 'shared-view-list', label: t('globals.terms.sharedView', 2) },
  { path: '', label: t('globals.messages.edit', { name: t('globals.terms.sharedView') }) }
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
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.sharedView')
      })
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
