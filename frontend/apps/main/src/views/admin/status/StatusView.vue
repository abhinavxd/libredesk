<template>
  <div>
    <AdminSplitLayout>
      <template #content>
        <LoadingOverlay :loading="isLoading" reserve-height>
          <div class="flex justify-between mb-5">
            <div class="flex justify-end mb-4 w-full">
              <Dialog v-model:open="dialogOpen">
                <DialogTrigger as-child @click="newStatus">
                  <Button class="ml-auto">
                    {{
                      $t('status.new')
                    }}
                  </Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[425px]">
                  <DialogHeader>
                    <DialogTitle>
                      {{
                        isEditing
                          ? $t('status.edit')
                          : $t('status.new')
                      }}
                    </DialogTitle>
                    <DialogDescription>
                      {{ $t('admin.conversationStatus.name.description') }}
                    </DialogDescription>
                  </DialogHeader>
                  <StatusForm @submit.prevent="onSubmit">
                    <template #footer>
                      <DialogFooter class="mt-10">
                        <Button type="submit" :isLoading="isLoading" :disabled="isLoading">
                          {{ isEditing ? $t('globals.messages.save') : $t('globals.messages.create') }}
                        </Button>
                      </DialogFooter>
                    </template>
                  </StatusForm>
                </DialogContent>
              </Dialog>
            </div>
          </div>
          <div>
            <DataTable :columns="createColumns(t, { onEdit: editStatus })" :data="statuses" :loading="isLoading" :searchable="false" />
          </div>
        </LoadingOverlay>
      </template>

      <template #help>
        <p>{{ $t('admin.status.help') }}</p>
      </template>
    </AdminSplitLayout>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import { createColumns } from '../../../features/admin/status/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import StatusForm from '@/features/admin/status/StatusForm.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '../../../features/admin/status/formSchema.js'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from 'vue-i18n'
import api from '../../../api'

const { t } = useI18n()
const isLoading = ref(false)
const statuses = ref([])
const emit = useEmitter()
const dialogOpen = ref(false)
const isEditing = ref(false)
const editingId = ref(null)

const refreshHandler = (data) => {
  if (data?.model === 'status') getStatuses()
}
const editHandler = (data) => {
  if (data?.model === 'status') {
    editStatus(data.data)
  }
}

onMounted(() => {
  getStatuses()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emit.on(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emit.off(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const editStatus = (item) => {
  editingId.value = item.id
  form.setValues(item)
  form.setErrors({})
  isEditing.value = true
  dialogOpen.value = true
}

const newStatus = () => {
  form.resetForm()
  form.setErrors({})
  isEditing.value = false
}

const getStatuses = async () => {
  try {
    isLoading.value = true
    const resp = await api.getStatuses()
    statuses.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    isLoading.value = true
    if (isEditing.value) {
      await api.updateStatus(editingId.value, values)
    } else {
      await api.createStatus(values)
    }
    dialogOpen.value = false
    getStatuses()
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.savedSuccessfully')
    })
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})
</script>
