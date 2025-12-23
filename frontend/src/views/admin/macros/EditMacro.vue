<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading" />
  <MacroForm :initialValues="macro" :submitForm="submitForm" :isLoading="formLoading" v-else />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import MacroForm from '@/features/admin/macros/MacroForm.vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { useI18n } from 'vue-i18n'
import { Spinner } from '@/components/ui/spinner'
import { useMacroStore } from '@/stores/macro'

const macro = ref({})
const { t } = useI18n()
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()
const macroStore = useMacroStore()

const breadcrumbLinks = [
  { path: 'macro-list', label: t('globals.terms.macro', 2) },
  { path: '', label: t('globals.messages.edit', { name: t('globals.terms.macro') }) }
]

const submitForm = (values) => {
  updateMacro(values)
}

const updateMacro = async (payload) => {
  try {
    formLoading.value = true
    await api.updateMacro(macro.value.id, payload)
    
    // Reload macros from server
    await macroStore.loadMacros(true)
    
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.macro')
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