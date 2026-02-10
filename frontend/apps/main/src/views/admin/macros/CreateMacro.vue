<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <MacroForm :submitForm="onSubmit" :isLoading="formLoading" />
</template>

<script setup>
import { ref } from 'vue'
import MacroForm from '@main/features/admin/macros/MacroForm.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useRouter } from 'vue-router'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import { useMacroStore } from '@main/stores/macro'
import api from '@main/api'

const router = useRouter()
const emit = useEmitter()
const { t } = useI18n()
const macroStore = useMacroStore()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: 'macro-list', label: t('globals.terms.macro', 2) },
  {
    path: '',
    label: t('globals.messages.new', {
      name: t('globals.terms.macro')
    })
  }
]

const onSubmit = (values) => {
  createMacro(values)
}

const createMacro = async (values) => {
  try {
    formLoading.value = true
    await api.createMacro(values)
    
    await macroStore.loadMacros(true)
    
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', {
        name: t('globals.terms.macro')
      })
    })
    router.push({ name: 'macro-list' })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>