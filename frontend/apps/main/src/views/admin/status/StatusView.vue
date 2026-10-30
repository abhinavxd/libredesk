<template>
  <div>
    <AdminSplitLayout>
      <template #content>
        <LoadingOverlay :loading="isLoading" reserve-height>
          <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
            <template #actions>
              <Dialog v-model:open="dialogOpen">
                <DialogTrigger as-child @click="newStatus">
                  <Button class="ml-auto">
                    {{ i18n._('New status') }}
                  </Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[425px]">
                  <DialogHeader>
                    <DialogTitle>
                      {{ isEditing ? i18n._('Edit status') : i18n._('New status') }}
                    </DialogTitle>
                    <DialogDescription>
                      {{ i18n._('Set status name. Click save when you\'\'re done.') }}
                    </DialogDescription>
                  </DialogHeader>
                  <StatusForm @submit.prevent="onSubmit">
                    <template #footer>
                      <DialogFooter class="mt-10">
                        <Button type="submit" :isLoading="isLoading" :disabled="isLoading">
                          {{
                            isEditing ? i18n._('Save') : i18n._('Create')
                          }}
                        </Button>
                      </DialogFooter>
                    </template>
                  </StatusForm>
                </DialogContent>
              </Dialog>
            </template>
          </AdminToolbar>
          <div>
            <DataTable
              v-model:search="searchTerm"
              :searchable="false"
              :columns="createColumns(i18n, { onEdit: editStatus })"
              :data="statuses"
              :loading="isLoading"
            />
          </div>
        </LoadingOverlay>
      </template>

      <template #help>
        <p>{{ i18n._('Create custom conversation statuses to extend default workflow.') }}</p>
      </template>
    </AdminSplitLayout>
  </div>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'

import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import { createColumns } from '@/features/admin/status/dataTableColumns.js'
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
import { createFormSchema } from '@/features/admin/status/formSchema.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'
import api from '@/api'

const searchTerm = ref('')
const i18n = useI18n()
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
  validationSchema: toTypedSchema(createFormSchema(i18n))
})

const editStatus = (item) => {
  editingId.value = item.id
  form.setValues(item, false)
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
      description: i18n._('Changes saved')
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
