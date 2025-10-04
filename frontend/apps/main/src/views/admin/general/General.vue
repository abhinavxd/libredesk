<template>
  <AdminPageWithHelp>
    <template #content>
      <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
        <GeneralSettingForm
          :submitForm="submitForm"
          :initial-values="initialValues"
        />
        <Spinner v-if="isLoading" />
      </div>
    </template>
    <template #help>
      <p>General settings for your support desk like timezone, working hours, etc.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Spinner } from '@shared-ui/components/ui/spinner'
import GeneralSettingForm from '@/features/admin/general/GeneralSettingForm.vue'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import { useAppSettingsStore } from '@/stores/appSettings'
import api from '@/api'

const initialValues = ref({})
const isLoading = ref(false)
const settingsStore = useAppSettingsStore()

onMounted(async () => {
  isLoading.value = true
  await settingsStore.fetchSettings('general')
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
