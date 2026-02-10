<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading"></Spinner>
  <RoleForm :initial-values="role" :submitForm="submitForm" :isLoading="formLoading" v-else />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import RoleForm from '@/features/admin/roles/RoleForm.vue'
import { Spinner } from '@shared-ui/components/ui/spinner'
import api from '../../../api'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'

const emitter = useEmitter()
const { t } = useI18n()
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
  { path: 'role-list', label: 'Roles' },
  { path: '', label: 'Edit role' }
]

const submitForm = async (values) => {
  try {
    formLoading.value = true
    await api.updateRole(props.id, values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.role')
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
</script>
