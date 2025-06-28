<template>
  <div class="">
    <CustomBreadcrumb :links="breadcrumbLinks" />
    <ArticleForm
      :submitForm="onSubmit"
      :initialValues="{}"
      :isNewForm="true"
      :isLoading="formLoading"
      class="mt-5"
    />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { handleHTTPError } from '@/utils/http'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useI18n } from 'vue-i18n'
import api from '@/api'
import ArticleForm from '@/features/articles/article/ArticleForm.vue'

const { t } = useI18n()
const emitter = useEmitter()
const router = useRouter()
const formLoading = ref(false)
const breadcrumbLinks = [
  { path: 'new-article', label: t('globals.terms.article', 2) },
  {
    path: '',
    label: t('globals.messages.new', {
      name: t('globals.terms.article', 1).toLowerCase()
    })
  }
]

const onSubmit = (values) => {
  createNewArticle(values)
}

const createNewArticle = async (values) => {
  try {
    formLoading.value = true
    await api.createArticle(values)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.createdSuccessfully', {
        name: t('globals.terms.article', 1)
      })
    })
    router.push({ name: 'new-article' })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}
</script>
