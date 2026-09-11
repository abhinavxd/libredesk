<template>
  <AdminSplitLayout>
    <template #content>
      <div class="mb-5">
        <CustomBreadcrumb :links="breadcrumbLinks" />
      </div>
      <LoadingOverlay :loading="isLoading">
        <GuidedFormForm :initial-values="guidedForm" :is-editing="!!id" :submit-form="submitForm" />
      </LoadingOverlay>
    </template>

    <template #help>
      <p>{{ t('admin.guidedForms.help') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import api from '@/api'
import GuidedFormForm from '@/features/admin/guidedForms/GuidedFormForm.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  id: { type: String, required: false }
})

const { t } = useI18n()
const router = useRouter()
const emitter = useEmitter()
const guidedForm = ref({})
const isLoading = ref(false)

const breadcrumbLinks = [
  { path: 'guided-forms', label: t('admin.guidedForms.title') },
  { path: '', label: props.id ? t('admin.guidedForms.edit') : t('admin.guidedForms.new') }
]

const submitForm = async (values) => {
  try {
    if (props.id) {
      await api.updateGuidedForm(props.id, values)
    } else {
      await api.createGuidedForm(values)
      router.push({ name: 'guided-forms' })
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const loadForm = async () => {
  if (!props.id) return
  try {
    isLoading.value = true
    const resp = await api.getGuidedForm(props.id)
    guidedForm.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

onMounted(loadForm)

watch(() => props.id, loadForm)
</script>
