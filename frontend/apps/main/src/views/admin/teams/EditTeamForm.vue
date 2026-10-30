<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <TeamForm
      :initial-values="team"
      :submitForm="submitForm"
      :isLoading="formLoading"
      v-if="!isLoading"
    />
  </LoadingOverlay>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import TeamForm from '@/features/admin/teams/TeamForm.vue'
import { CustomBreadcrumb } from '@shared-ui/components/ui/breadcrumb'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useI18n } from '@shared-ui/i18n'

const i18n = useI18n()

const team = ref({})
const emitter = useEmitter()
const formLoading = ref(false)
const isLoading = ref(false)

const breadcrumbLinks = [
  { path: 'team-list', label: i18n._('{count, plural, one {Team} other {Teams}}', 2) },
  { path: '', label: i18n._('Edit') }
]

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})

const submitForm = (values) => {
  updateTeam(values)
}

const updateTeam = async (payload) => {
  try {
    formLoading.value = true
    await api.updateTeam(team.value.id, payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
    })
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

onMounted(async () => {
  try {
    isLoading.value = true
    const resp = await api.getTeam(props.id)
    team.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})
</script>
