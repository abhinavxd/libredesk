<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <RoleForm :initial-values="{}" :submitForm="submitForm" :isLoading="formLoading" />
</template>

<script setup>
import { ref } from 'vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import RoleForm from '@/features/admin/roles/RoleForm.vue'
import api from '../../../api'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '../../../utils/http'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const emitter = useEmitter()
const { t } = useI18n()
const router = useRouter()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: 'role-list', label: t('globals.terms.role', 2) },
  {
    path: '',
    label: t('globals.messages.new', {
      name: t('globals.terms.role')
    })
  }
]

const submitForm = async (values) => {
  try {
    formLoading.value = true
    await api.createRole(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', {
        name: t('globals.terms.role')
      })
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
