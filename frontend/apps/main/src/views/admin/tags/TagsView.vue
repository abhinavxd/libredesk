<template>
  <div>
    <AdminSplitLayout>
      <template #content>
        <LoadingOverlay :loading="isLoading" reserve-height>
          <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
            <template #actions>
              <Importer
                :upload-fn="api.importTags"
                :get-status-fn="api.getTagImportStatus"
                @import-complete="getTags"
              >
                <template #csv-example>
                  <div
                    class="bg-muted p-3 rounded-md text-xs font-mono overflow-x-auto leading-relaxed"
                  >
                    <div>name</div>
                    <div>Bug</div>
                    <div>Feature Request</div>
                    <div>Billing</div>
                  </div>
                </template>
              </Importer>
              <Dialog v-model:open="dialogOpen">
                <DialogTrigger as-child @click="newTag">
                  <Button>{{ i18n._('New tag') }}</Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[425px]">
                  <DialogHeader>
                    <DialogTitle class="mb-1">
                      {{ isEditing ? i18n._('Edit tag') : i18n._('New tag') }}
                    </DialogTitle>
                    <DialogDescription>
                      {{
                        isEditing
                          ? i18n._('Change the tag name. Click save when you\'\'re done.')
                          : i18n._('Set tag name. Click save when you\'\'re done.')
                      }}
                    </DialogDescription>
                  </DialogHeader>
                  <TagsForm @submit.prevent="onSubmit">
                    <template #footer>
                      <DialogFooter class="mt-10">
                        <Button type="submit">{{
                          isEditing ? i18n._('Save') : i18n._('Create')
                        }}</Button>
                      </DialogFooter>
                    </template>
                  </TagsForm>
                </DialogContent>
              </Dialog>
            </template>
          </AdminToolbar>
          <div>
            <DataTable
              v-model:search="searchTerm"
              :searchable="false"
              :columns="createColumns(i18n, { onEdit: editTag })"
              :data="tags"
              :loading="isLoading"
            />
          </div>
        </LoadingOverlay>
      </template>

      <template #help>
        <p>{{ i18n._('Tags can be used to filter conversations and as conditions in automations.') }}</p>
      </template>
    </AdminSplitLayout>
  </div>
</template>

<script setup>
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'

import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { createColumns } from '@/features/admin/tags/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button/index.js'

import TagsForm from '@/features/admin/tags/TagsForm.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog/index.js'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '@/features/admin/tags/formSchema.js'
import { useEmitter } from '@/composables/useEmitter.js'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import { useI18n } from '@shared-ui/i18n'
import Importer from '@/components/importer/Importer.vue'
import api from '@/api/index.js'

const searchTerm = ref('')
const i18n = useI18n()
const isLoading = ref(false)
const tags = ref([])
const emitter = useEmitter()
const dialogOpen = ref(false)
const isEditing = ref(false)
const editingId = ref(null)

const refreshHandler = (data) => {
  if (data?.model === 'tags') getTags()
}
const editHandler = (data) => {
  if (data?.model === 'tags') {
    editTag(data.data)
  }
}

onMounted(() => {
  getTags()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n))
})

const editTag = (item) => {
  editingId.value = item.id
  form.setValues(item, false)
  form.setErrors({})
  isEditing.value = true
  dialogOpen.value = true
}

const newTag = () => {
  form.resetForm()
  form.setErrors({})
  isEditing.value = false
}

const getTags = async () => {
  isLoading.value = true
  const resp = await api.getTags()
  tags.value = resp.data.data
  isLoading.value = false
}

const onSubmit = form.handleSubmit(async (values) => {
  isLoading.value = true
  try {
    if (isEditing.value) {
      await api.updateTag(editingId.value, values)
    } else {
      await api.createTag(values)
    }
    dialogOpen.value = false
    getTags()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: i18n._('Changes saved')
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})
</script>
