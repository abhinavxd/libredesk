<template>
  <LoadingOverlay :loading="isLoading" reserve-height>
    <div class="flex justify-end mb-4">
      <Dialog v-model:open="dialogOpen">
        <DialogTrigger as-child @click="newForm">
          <Button>{{ $t('admin.portalForm.new') }}</Button>
        </DialogTrigger>
        <DialogContent class="sm:max-w-[700px] max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>
              {{ current.id ? $t('admin.portalForm.edit') : $t('admin.portalForm.new') }}
            </DialogTitle>
            <DialogDescription />
          </DialogHeader>
          <TicketFormEditor v-model="current" />
          <DialogFooter class="mt-6">
            <Button :isLoading="isSaving" @click="save">
              {{ current.id ? $t('globals.messages.save') : $t('globals.messages.create') }}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
    <DataTable :columns="createColumns(t, { onEdit: editForm })" :data="forms" :loading="isLoading" />
  </LoadingOverlay>
</template>

<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import TicketFormEditor from '@main/features/admin/portal/forms/TicketFormEditor.vue'
import { createColumns } from '@main/features/admin/portal/forms/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import api from '@main/api'

const { t } = useI18n()
const emitter = useEmitter()
const forms = ref([])
const isLoading = ref(false)
const isSaving = ref(false)
const dialogOpen = ref(false)
const current = ref({ id: 0, name: '', ask_subject: true, fields: [] })

const blank = () => ({ id: 0, name: '', ask_subject: true, fields: [] })

const refreshHandler = (data) => {
  if (data?.model === 'portal-forms') fetchAll()
}
const editHandler = (data) => {
  if (data?.model === 'portal-forms') editForm(data.data)
}

onMounted(() => {
  fetchAll()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

const newForm = () => {
  current.value = blank()
  dialogOpen.value = true
}

const editForm = (form) => {
  current.value = JSON.parse(JSON.stringify({ ...blank(), ...form, fields: form.fields || [] }))
  dialogOpen.value = true
}

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getPortalForms()
    forms.value = resp.data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const save = async () => {
  try {
    isSaving.value = true
    const payload = {
      name: current.value.name,
      ask_subject: current.value.ask_subject,
      fields: current.value.fields
    }
    if (current.value.id) {
      await api.updatePortalForm(current.value.id, payload)
    } else {
      await api.createPortalForm(payload)
    }
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
    dialogOpen.value = false
    fetchAll()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isSaving.value = false
  }
}
</script>
