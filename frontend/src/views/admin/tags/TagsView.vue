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
                  <Button class="ml-auto">{{
                    t('globals.messages.new', {
                      name: t('globals.terms.tag')
                    })
                  }}</Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[425px]">
                  <DialogHeader>
                    <DialogTitle class="mb-1">
                      {{
                        t('globals.messages.new', {
                          name: t('globals.terms.tag')
                        })
                      }}
                    </DialogTitle>
                    <DialogDescription>
                      {{ t('admin.conversationTags.new.description') }}
                    </DialogDescription>
                  </DialogHeader>
                  <TagsForm @submit.prevent="onSubmit">
                    <template #footer>
                      <DialogFooter class="mt-10">
                        <Button type="submit">{{ t('globals.messages.save') }}</Button>
                      </DialogFooter>
                    </template>
                  </TagsForm>
                </DialogContent>
              </Dialog>
            </div>
          </div>
          <div>
            <DataTable :columns="createColumns(t)" :data="tags" />
          </div>
        </div>
      </template>

      <template #help>
        <p>Tags help you categorize your conversations. Create or edit tags here.</p>
      </template>
    </AdminPageWithHelp>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@/components/datatable/DataTable.vue'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { Spinner } from '@/components/ui/spinner'
import { createColumns } from '@/features/admin/tags/dataTableColumns.js'
import { Button } from '@/components/ui/button'

import TagsForm from '@/features/admin/tags/TagsForm.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/components/ui/dialog'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { createFormSchema } from '../../../features/admin/tags/formSchema.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const { t } = useI18n()
const isLoading = ref(false)
const tags = ref([])
const emitter = useEmitter()
const dialogOpen = ref(false)

onMounted(() => {
  getTags()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'tags') getTags()
  })
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const getTags = async () => {
  isLoading.value = true
  const resp = await api.getTags()
  tags.value = resp.data.data
  isLoading.value = false
}

const onSubmit = form.handleSubmit(async (values) => {
  isLoading.value = true
  try {
    await api.createTag(values)
    dialogOpen.value = false
    getTags()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', { name: t('globals.terms.tag') }),
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
