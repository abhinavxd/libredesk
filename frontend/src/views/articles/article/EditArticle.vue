<template>
  <div>
    <CustomBreadcrumb :links="breadcrumbLinks" class="mb-5" />
    <Spinner v-if="isLoading" />
    <ArticleForm
      :initialValues="article"
      :submitForm="submitForm"
      :isLoading="formLoading"
      v-else
    />
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '@/api'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { CustomBreadcrumb } from '@/components/ui/breadcrumb'
import { Spinner } from '@/components/ui/spinner'
import { useI18n } from 'vue-i18n'
import ArticleForm from '@/features/articles/article/ArticleForm.vue'

const article = ref({})
const { t } = useI18n()
const isLoading = ref(false)
const formLoading = ref(false)
const emitter = useEmitter()

const breadcrumbLinks = [
  { path: 'article-list', label: t('globals.terms.article', 2) },
  {
    path: '',
    label: t('globals.messages.edit', {
      name: t('globals.terms.article', 1).toLowerCase()
    })
  }
]

const submitForm = (values) => {
  updateArticle(values)
}

const updateArticle = async (payload) => {
  try {
    formLoading.value = true
    await api.updateArticle(article.value.id, payload)
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      description: t('globals.messages.updatedSuccessfully', {
        name: t('globals.terms.agent', 1)
      })
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    formLoading.value = false
  }
}

onMounted(async () => {
  try {
    isLoading.value = true
    const resp = await api.getArticle(props.id)
    article.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
})

const props = defineProps({
  id: {
    type: String,
    required: true
  }
})
</script>
