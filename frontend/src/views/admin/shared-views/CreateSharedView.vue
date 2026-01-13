<template>
  <div class="mb-5">
    <CustomBreadcrumb :links="breadcrumbLinks" />
  </div>
  <SharedViewForm :submitForm="onSubmit" :isLoading="formLoading" />
</template>

<script setup>
import { ref } from 'vue'
import SharedViewForm from '@/features/admin/shared-views/SharedViewForm.vue'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { handleHTTPError } from '@/utils/http'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import { useSharedViewStore } from '@/stores/sharedView'
import api from '@/api'

const router = useRouter()
const emit = useEmitter()
const { t } = useI18n()
const sharedViewStore = useSharedViewStore()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: 'shared-view-list', label: t('globals.terms.sharedView', 2) },
  {
    path: '',
    label: t('globals.messages.new', {
      name: t('globals.terms.sharedView').toLowerCase()
    })
  }
]

const onSubmit = (values) => {
  createSharedView(values)
}

const createSharedView = async (values) => {
  try {
    formLoading.value = true
    await api.createSharedView(values)

    await sharedViewStore.refresh()

    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', {
        name: t('globals.terms.sharedView')
      })
    })
    router.push({ name: 'shared-view-list' })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
