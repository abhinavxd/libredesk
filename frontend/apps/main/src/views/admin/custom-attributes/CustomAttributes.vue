<template>
  <div>
    <Spinner v-if="isLoading" />
    <AdminPageWithHelp>
      <template #content>
        <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
          <div class="flex justify-between mb-5">
            <div></div>
            <div class="flex justify-end mb-4">
              <Dialog v-model:open="dialogOpen">
                <DialogTrigger as-child @click="newCustomAttribute">
                  <Button class="ml-auto">
                    {{
                      $t('customAttribute.new')
                    }}
                  </Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[600px]">
                  <DialogHeader>
                    <DialogTitle>
                      {{
                        isEditing
                          ? $t('customAttribute.edit')
                          : $t('customAttribute.new')
                      }}
                    </DialogTitle>
                    <DialogDescription/>
                  </DialogHeader>
                  <CustomAttributesForm @submit.prevent="onSubmit" :form="form">
                    <template #footer>
                      <DialogFooter class="mt-10">
                        <Button type="submit" :isLoading="isLoading">
                          {{
                            isEditing ? $t('globals.messages.update') : $t('globals.messages.create')
                          }}
                        </Button>
                      </DialogFooter>
                    </template>
                  </CustomAttributesForm>
                </DialogContent>
              </Dialog>
            </div>
          </div>
          <div>
            <Tabs default-value="contact" v-model="appliesTo">
              <TabsList class="grid w-full grid-cols-2 mb-5">
                <TabsTrigger value="contact">
                  {{ $t('globals.terms.contact') }}
                </TabsTrigger>
                <TabsTrigger value="conversation">
                  {{ $t('globals.terms.conversation') }}
                </TabsTrigger>
              </TabsList>
              <TabsContent value="contact">
                <DataTable :columns="createColumns(t)" :data="customAttributes" :loading="isLoading" />
              </TabsContent>
              <TabsContent value="conversation">
                <DataTable :columns="createColumns(t)" :data="customAttributes" :loading="isLoading" />
              </TabsContent>
            </Tabs>
          </div>
        </div>
      </template>

      <template #help>
        <p>{{ $t('admin.customAttribute.help') }}</p>
      </template>
    </AdminPageWithHelp>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, onUnmounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import { createColumns } from '../../../features/admin/custom-attributes/dataTableColumns.js'
import CustomAttributesForm from '@/features/admin/custom-attributes/CustomAttributesForm.vue'
import { Button } from '@shared-ui/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '../../../features/admin/custom-attributes/formSchema.js'
import { Spinner } from '@shared-ui/components/ui/spinner'
import { useEmitter } from '../../../composables/useEmitter'
import { EMITTER_EVENTS } from '../../../constants/emitterEvents.js'
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
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { useI18n } from 'vue-i18n'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '../../../api'

const appliesTo = useStorage('appliesTo', 'contact')
const { t } = useI18n()
const customAttributes = ref([])
const isLoading = ref(false)
const emitter = useEmitter()
const dialogOpen = ref(false)
const isEditing = ref(false)

onMounted(async () => {
  fetchAll()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'custom-attributes') fetchAll()
  })

  // Listen to the edit model event, this is emitted from the dropdown menu
  // in the datatable.
  emitter.on(EMITTER_EVENTS.EDIT_MODEL, (data) => {
    if (data?.model === 'custom-attributes') {
      form.setValues(data.data)
      form.setErrors({})
      isEditing.value = true
      dialogOpen.value = true
    }
  })
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.REFRESH_LIST)
  emitter.off(EMITTER_EVENTS.EDIT_MODEL)
})

const newCustomAttribute = () => {
  form.resetForm()
  form.setErrors({})
  isEditing.value = false
  dialogOpen.value = true
}

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t)),
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
        description: t('globals.messages.savedSuccessfully')
      })
    } else {
      await api.createCustomAttribute(values)
      emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
        description: t('globals.messages.savedSuccessfully')
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
