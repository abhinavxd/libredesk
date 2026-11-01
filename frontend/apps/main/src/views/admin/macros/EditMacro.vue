<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <MacroForm
      :initialValues="macro"
      :submitForm="submitForm"
      :isLoading="formLoading"
      v-if="!isLoading"
    />
  </LoadingOverlay>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@main/api'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useEmitter } from '@main/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import MacroForm from '@main/features/admin/macros/MacroForm.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { useI18n } from '@shared-ui/i18n'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useMacroStore } from '@main/stores/macro'

const macro = ref({})
const i18n = useI18n()
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()
const macroStore = useMacroStore()

const breadcrumbLinks = [
  { path: 'macro-list', label: i18n._('{count, plural, one {Macro} other {Macros}}', 2) },
  { path: '', label: i18n._('Edit macro') }
]

const submitForm = (values) => {
  updateMacro(values)
}

const updateMacro = async (payload) => {
  try {
    formLoading.value = true
    await api.updateMacro(macro.value.id, payload)

    macroStore.clearCache()

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
    const resp = await api.getMacro(props.id)
    macro.value = resp.data.data
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
