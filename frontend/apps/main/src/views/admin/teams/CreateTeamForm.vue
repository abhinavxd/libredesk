<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <TeamForm :initial-values="{}" :submitForm="submitForm" :isLoading="formLoading" />
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

const formLoading = ref(false)
const router = useRouter()
const emitter = useEmitter()
const breadcrumbLinks = [
  { path: 'team-list', label: 'Teams' },
  { path: '', label: 'New team' }
]

const submitForm = (values) => {
  createTeam(values)
}

const createTeam = async (values) => {
  try {
    formLoading.value = true
    await api.createTeam(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Success',
      description: "Team created successfully"
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
