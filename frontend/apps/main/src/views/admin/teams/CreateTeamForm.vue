<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <TeamForm :initial-values="{}" :submitForm="submitForm" :isLoading="formLoading" :isNewForm="true" />
</template>

<script setup>
import { ref } from 'vue'
import TeamForm from '@/features/admin/teams/TeamForm.vue'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { useEmitter } from '../../../composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { useRouter } from 'vue-router'
import api from '../../../api'
import { useI18n } from '@shared-ui/i18n'

const i18n = useI18n()
const formLoading = ref(false)
const router = useRouter()
const emitter = useEmitter()
const breadcrumbLinks = [
  { path: 'team-list', label: i18n._('{count, plural, one {Team} other {Teams}}', 2) },
  { path: '', label: i18n.pgettext('New team', 'New') }
]

const submitForm = (values) => {
  createTeam(values)
}

const createTeam = async (values) => {
  try {
    formLoading.value = true
    await api.createTeam(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
    })
    router.push({ name: 'team-list' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
