<template>
  <AdminSplitLayout>
    <template #content>
      <LoadingOverlay :loading="isLoading">
        <GeneralSettingForm
          :submitForm="submitForm"
          :initial-values="initialValues"
          :available-languages="availableLanguages"
        />
      </LoadingOverlay>
    </template>
    <template #help>
      <p>{{ $t('admin.general.help') }}</p>
    </template>
  </AdminSplitLayout>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import LoadingOverlay from '@/components/layout/LoadingOverlay.vue'
import GeneralSettingForm from '@/features/admin/general/GeneralSettingForm.vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import { useAppSettingsStore } from '@/stores/appSettings'
import api from '@/api'

const initialValues = ref({})
const availableLanguages = ref([])
const isLoading = ref(false)
const settingsStore = useAppSettingsStore()

onMounted(async () => {
  isLoading.value = true
  const [, langsResp] = await Promise.all([
    settingsStore.fetchSettings('general'),
    api.getAvailableLanguages()
  ])
  availableLanguages.value = langsResp.data.data
  const data = settingsStore.settings
  isLoading.value = false
  initialValues.value = Object.keys(data).reduce((acc, key) => {
    // Remove 'app.' prefix
    const newKey = key.replace(/^app\./, '')
    acc[newKey] = data[key]
    return acc
  }, {})
})

const submitForm = async (values) => {
  // Prepend keys with `app.`
  const updatedValues = Object.fromEntries(
    Object.entries(values).map(([key, value]) => [`app.${key}`, value])
  )
  await api.updateSettings('general', updatedValues)
}
</script>
