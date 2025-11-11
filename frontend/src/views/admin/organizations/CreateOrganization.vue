<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <OrganizationForm
    :submitForm="onSubmit"
    :initialValues="{}"
    :isNewForm="true"
    :isLoading="formLoading"
    :submitLabel="$t('globals.messages.create')"
  />
</template>

<script setup>
import { ref } from 'vue'
import OrganizationForm from '@/features/admin/organizations/OrganizationForm.vue'
import { handleHTTPError } from '@/utils/http'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const { t } = useI18n()
const emitter = useEmitter()
const router = useRouter()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: 'organization-list', label: t('globals.terms.organization', 2) },
  {
    path: '',
    label: t('globals.messages.new', {
      name: t('globals.terms.organization', 1).toLowerCase()
    })
  }
]

const onSubmit = (values) => {
  createNewOrganization(values)
}

const createNewOrganization = async (values) => {
  try {
    formLoading.value = true
    await api.createOrganization(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', {
        name: t('globals.terms.organization', 1)
      })
    })
    router.push({ name: 'organization-list' })
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
