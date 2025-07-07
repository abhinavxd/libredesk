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
                      name: t('globals.terms.category')
                    })
                  }}</Button>
                </DialogTrigger>
                <DialogContent class="sm:max-w-[425px]">
                  <DialogHeader>
                    <DialogTitle class="mb-1">
                      {{
                        t('globals.messages.new', {
                          name: t('globals.terms.category')
                        })
                      }}
                    </DialogTitle>
                    <DialogDescription>
                      {{ t('articles.category.new.description') }}
                    </DialogDescription>
                  </DialogHeader>
                  <CategoryForm @submit.prevent="onSubmit">
                    <template #footer>
                      <DialogFooter class="mt-10">
                        <Button type="submit">{{ t('globals.messages.save') }}</Button>
                      </DialogFooter>
                    </template>
                  </CategoryForm>
                </DialogContent>
              </Dialog>
            </div>
          </div>
          <div>
            <DataTable :columns="createColumns(t)" :data="categories" />
          </div>
        </div>
      </template>

      <template #help>
        <p>Categories help you organize your articles. Create or edit categories here.</p>
      </template>
    </AdminPageWithHelp>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@/components/datatable/DataTable.vue'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { Spinner } from '@/components/ui/spinner'
import { createColumns } from '@/features/articles/category/dataTableColumns.js'
import { Button } from '@/components/ui/button'

import CategoryForm from '@/features/articles/category/CategoryForm.vue'
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
import { createFormSchema } from '../../../features/articles/category/formSchema.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import { useI18n } from 'vue-i18n'
import api from '@/api'

const { t } = useI18n()
const isLoading = ref(false)
const categories = ref([])
const emitter = useEmitter()
const dialogOpen = ref(false)

onMounted(() => {
  getCategories()
  emitter.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'article_category') getCategories()
  })
})

const form = useForm({
  validationSchema: toTypedSchema(createFormSchema(t))
})

const getCategories = async () => {
  isLoading.value = true
  const resp = await api.getArticleCategories()
  categories.value = resp.data.data
  isLoading.value = false
}

const onSubmit = form.handleSubmit(async (values) => {
  isLoading.value = true
  try {
    await api.createArticleCategory(values)
    dialogOpen.value = false
    getCategories()
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', { name: t('globals.terms.category') })
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
