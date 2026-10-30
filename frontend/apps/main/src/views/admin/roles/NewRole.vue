<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <RoleForm :initial-values="{}" :submitForm="submitForm" :isLoading="formLoading" :isNewForm="true" />
</template>

<script setup>
import { ref } from 'vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import RoleForm from '@/features/admin/roles/RoleForm.vue'
import api from '../../../api'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'
import { useRouter } from 'vue-router'

const emitter = useEmitter()
const i18n = useI18n()
const router = useRouter()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: 'role-list', label: i18n._('{count, plural, one {Role} other {Roles}}', 2) },
  {
    path: '',
    label: i18n._('New role')
  }
]

const submitForm = async (values) => {
  try {
    formLoading.value = true
    await api.createRole(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
    })
    router.push({ name: 'role-list' })
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
