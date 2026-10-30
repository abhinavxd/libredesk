<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <RoleForm
      :initial-values="role"
      :submitForm="submitForm"
      :isLoading="formLoading"
      v-if="!isLoading"
    />
  </LoadingOverlay>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import RoleForm from '@/features/admin/roles/RoleForm.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { useI18n } from '@shared-ui/i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const emitter = useEmitter()
const i18n = useI18n()
const role = ref({})
const isLoading = ref(false)
const formLoading = ref(false)
const props = defineProps({
  id: {
    type: String,
    required: true
  }
})

onMounted(async () => {
  try {
    isLoading.value = true
    const resp = await api.getRole(props.id)
    role.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})

const breadcrumbLinks = [
  { path: 'role-list', label: i18n._('{count, plural, one {Role} other {Roles}}', 2) },
  { path: '', label: i18n._('Edit role') }
]

const submitForm = async (values) => {
  try {
    formLoading.value = true
    await api.updateRole(props.id, values)
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
</script>
