<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <Spinner v-if="isLoading" />
  <OrganizationForm
    :initialValues="organization"
    :submitForm="submitForm"
    :isLoading="formLoading"
    :submitLabel="$t('globals.messages.update')"
    v-else
  />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import OrganizationForm from '@/features/admin/organizations/OrganizationForm.vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'

const organization = ref({})
const { t } = useI18n()
const router = useRouter()
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()

const breadcrumbLinks = [
  { path: 'organization-list', label: t('globals.terms.organization', 2) },
  {
    path: '',
    label: t('globals.messages.edit', {
      name: t('globals.terms.organization', 1).toLowerCase()
    })
  }
]

const submitForm = (values) => {
  updateOrganization(values)
}

const updateOrganization = async (payload) => {
  try {
    formLoading.value = true
    await api.updateOrganization(organization.value.id, payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.organization', 1)
      })
    })
    router.push({ name: 'organization-detail', params: { id: organization.value.id } })
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
    const resp = await api.getOrganization(props.id)
    organization.value = resp.data.data
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
