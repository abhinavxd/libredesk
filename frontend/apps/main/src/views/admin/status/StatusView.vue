<template>
  <div>
    <Spinner v-if="isLoading" />
    <AdminPageWithHelp>
      <template #content>
        <div :class="{ 'transition-opacity duration-300 opacity-50': isLoading }">
          <div class="flex justify-between mb-5">
            <div class="flex justify-end mb-4 w-full">
              <Dialog v-model:open="dialogOpen">
                <DialogTrigger as-child>
                  <Button class="ml-auto">
                    {{
                      $t('globals.messages.new', {
                        name: $t('globals.terms.status')
                      })
                    }}
                  </Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[425px]">
                  <DialogHeader>
                    <DialogTitle>
                      {{
                        $t('globals.messages.new', {
                          name: $t('globals.terms.status')
                        })
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
                          {{ $t('globals.messages.save') }}
                        </Button>
                      </DialogFooter>
                    </template>
                  </StatusForm>
                </DialogContent>
              </Dialog>
            </div>
          </div>
          <div>
            <DataTable :columns="createColumns(t)" :data="statuses" :loading="isLoading" />
          </div>
        </div>
      </template>

      <template #help>
        <p>Create custom conversation statuses to extend default workflow.</p>
      </template>
    </AdminPageWithHelp>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@main/components/datatable/DataTable.vue'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { createColumns } from '../../../features/admin/status/dataTableColumns.js'
import { Button } from '@shared-ui/components/ui/button'
import { Spinner } from '@shared-ui/components/ui/spinner'
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
import { useI18n } from 'vue-i18n'
import api from '../../../api'

const { t } = useI18n()
const isLoading = ref(false)
const statuses = ref([])
const emit = useEmitter()
const dialogOpen = ref(false)

onMounted(() => {
  getStatuses()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'status') getStatuses()
  })
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

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
    await api.createStatus(values)
    dialogOpen.value = false
    getStatuses()
  } catch (error) {
    console.error('Failed to create status:', error)
  } finally {
    isLoading.value = false
  }
})
</script>
