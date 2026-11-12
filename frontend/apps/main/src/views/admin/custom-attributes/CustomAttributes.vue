<template>
  <div>
    <AdminSplitLayout>
      <template #content>
        <LoadingOverlay :loading="isLoading" reserve-height>
          <Tabs default-value="contact" v-model="appliesTo">
            <AdminToolbar v-model:search="searchTerm" searchable class="mb-5">
              <template #left>
                <TabsList class="w-max justify-start">
                  <TabsTrigger value="contact">{{ i18n.ngettext('Contact', 'Contacts', 1) }}</TabsTrigger>
                  <TabsTrigger value="conversation">{{
                    i18n.ngettext('Conversation', 'Conversations', 1)
                  }}</TabsTrigger>
                </TabsList>
              </template>
              <template #actions>
                <Dialog v-model:open="dialogOpen">
                  <DialogTrigger as-child @click="newCustomAttribute">
                    <Button type="button">
                      {{ i18n._('New custom attribute') }}
                    </Button>
                  </DialogTrigger>
                  <DialogContent class="sm:max-w-[600px]">
                    <DialogHeader>
                      <DialogTitle>
                        {{ isEditing ? i18n._('Edit custom attribute') : i18n._('New custom attribute') }}
                      </DialogTitle>
                      <DialogDescription />
                    </DialogHeader>
                    <CustomAttributesForm @submit.prevent="onSubmit" :form="form">
                      <template #footer>
                        <DialogFooter class="mt-10">
                          <Button type="submit" :isLoading="isLoading">
                            {{
                              isEditing
                                ? i18n._('Save')
                                : i18n._('Create')
                            }}
                          </Button>
                        </DialogFooter>
                      </template>
                    </CustomAttributesForm>
                  </DialogContent>
                </Dialog>
              </template>
            </AdminToolbar>
            <TabsContent value="contact">
              <DataTable
                v-model:search="searchTerm"
                :searchable="false"
                :columns="createColumns(i18n, { onEdit: editCustomAttribute })"
                :data="customAttributes"
                :loading="isLoading"
              />
            </TabsContent>
            <TabsContent value="conversation">
              <DataTable
                v-model:search="searchTerm"
                :searchable="false"
                :columns="createColumns(i18n, { onEdit: editCustomAttribute })"
                :data="customAttributes"
                :loading="isLoading"
              />
            </TabsContent>
          </Tabs>
        </LoadingOverlay>
      </template>

      <template #help>
        <p>{{ i18n._('Custom attributes help you set additional details about your contacts or conversations such as the subscription plan or the date of their first purchase.') }}</p>
      </template>
    </AdminSplitLayout>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import AdminToolbar from '@main/layouts/admin/AdminToolbar.vue'
import { createColumns } from '@main/features/admin/custom-attributes/dataTableColumns.js'
import CustomAttributesForm from '@/features/admin/custom-attributes/CustomAttributesForm.vue'
import { Button } from '@shared-ui/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '@main/features/admin/custom-attributes/formSchema.js'
import LoadingOverlay from '@main/components/layout/LoadingOverlay.vue'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents.js'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@shared-ui/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@shared-ui/components/ui/tabs'
import { useStorage } from '@vueuse/core'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import { useI18n } from '@shared-ui/i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@main/api'

const appliesTo = useStorage('appliesTo', 'contact')
const searchTerm = ref('')
const i18n = useI18n()
const customAttributes = ref([])
const isLoading = ref(false)
const emitter = useEmitter()
const dialogOpen = ref(false)
const isEditing = ref(false)

const refreshHandler = (data) => {
  if (data?.model === 'custom-attributes') fetchAll()
}
const editHandler = (data) => {
  if (data?.model === 'custom-attributes') {
    form.setValues(data.data, false)
    form.setErrors({})
    isEditing.value = true
    dialogOpen.value = true
  }
}

onMounted(async () => {
  fetchAll()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST, refreshHandler)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL, editHandler)
})

const editCustomAttribute = (item) => {
  form.setValues(item, false)
  form.setErrors({})
  isEditing.value = true
  dialogOpen.value = true
}

const newCustomAttribute = () => {
  form.resetForm()
  form.setErrors({})
  isEditing.value = false
  dialogOpen.value = true
}

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(i18n)),
  initialValues: {
    id: 0,
    name: '',
    data_type: 'text',
    applies_to: appliesTo.value,
    values: []
  }
})

const fetchAll = async () => {
  if (!appliesTo.value) return
  try {
    isLoading.value = true
    const resp = await api.getCustomAttributes(appliesTo.value)
    customAttributes.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    isLoading.value = true
    if (values.id) {
      await api.updateCustomAttribute(values.id, values)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: i18n._('Changes saved')
      })
    } else {
      await api.createCustomAttribute(values)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: i18n._('Changes saved')
      })
    }

    dialogOpen.value = false
    fetchAll()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})

watch(
  appliesTo,
  (newVal) => {
    searchTerm.value = ''
    form.resetForm({
      values: {
        ...form.values,
        applies_to: newVal
      }
    })
    fetchAll()
  },
  { immediate: true }
)
</script>
