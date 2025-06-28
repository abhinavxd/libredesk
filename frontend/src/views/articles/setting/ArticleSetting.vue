<template>
  <AdminPageWithHelp>
    <template #content>
      <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
        <ArticleSettingForm :submitForm="submitForm" :initial-values="initialValues" />
        <Spinner v-if="isLoading" />
      </div>
    </template>
    <template #help>
      <p>
        Customize the appearance and content settings for your articles including site title, fonts,
        and branding.
      </p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Spinner } from '@/components/ui/spinner'
import ArticleSettingForm from '@/features/articles/settings/ArticleSettingsForm.vue'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import api from '@/api'

const initialValues = ref({})
const isLoading = ref(false)

onMounted(async () => {
  isLoading.value = true
  const response = await api.getSettings('article')
  const data = response.data.data
  isLoading.value = false
  initialValues.value = Object.keys(data).reduce((acc, key) => {
    // Remove 'article.' prefix
    const newKey = key.replace(/^article\./, '')
    acc[newKey] = data[key]
    return acc
  }, {})
})

const submitForm = async (values) => {
  // Prepend keys with `article.`
  const updatedValues = Object.fromEntries(
    Object.entries(values).map(([key, value]) => [`article.${key}`, value])
  )
  await api.updateSettings('article', updatedValues)
}
</script>
